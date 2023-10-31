package watchtower

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/klauspost/compress/zstd"
	batch "github.com/rocket-pool/batch-query"
	"github.com/rocket-pool/rocketpool-go/core"
	"github.com/rocket-pool/rocketpool-go/rewards"
	"github.com/rocket-pool/rocketpool-go/rocketpool"
	"github.com/rocket-pool/rocketpool-go/utils/eth"
	"github.com/rocket-pool/smartnode/rocketpool/common/beacon"
	"github.com/rocket-pool/smartnode/rocketpool/common/gas"
	rprewards "github.com/rocket-pool/smartnode/rocketpool/common/rewards"
	"github.com/rocket-pool/smartnode/rocketpool/common/services"
	"github.com/rocket-pool/smartnode/rocketpool/common/state"
	"github.com/rocket-pool/smartnode/rocketpool/common/tx"
	"github.com/rocket-pool/smartnode/rocketpool/common/wallet"
	"github.com/rocket-pool/smartnode/rocketpool/watchtower/utils"
	"github.com/rocket-pool/smartnode/shared/config"
	sharedtypes "github.com/rocket-pool/smartnode/shared/types"
	cfgtypes "github.com/rocket-pool/smartnode/shared/types/config"
	"github.com/rocket-pool/smartnode/shared/utils/eth1"
	hexutil "github.com/rocket-pool/smartnode/shared/utils/hex"
	"github.com/rocket-pool/smartnode/shared/utils/log"
	"github.com/web3-storage/go-w3s-client"
)

// Submit rewards Merkle Tree task
type SubmitRewardsTree_Stateless struct {
	sp               *services.ServiceProvider
	log              *log.ColorLogger
	errLog           *log.ColorLogger
	cfg              *config.RocketPoolConfig
	w                *wallet.LocalWallet
	rp               *rocketpool.RocketPool
	ec               core.ExecutionClient
	bc               beacon.Client
	rewardsPool      *rewards.RewardsPool
	lock             *sync.Mutex
	isRunning        bool
	generationPrefix string
	m                *state.NetworkStateManager
}

// Create submit rewards Merkle Tree task
func NewSubmitRewardsTree_Stateless(sp *services.ServiceProvider, logger log.ColorLogger, errorLogger log.ColorLogger, m *state.NetworkStateManager) *SubmitRewardsTree_Stateless {
	lock := &sync.Mutex{}
	return &SubmitRewardsTree_Stateless{
		sp:               sp,
		log:              &logger,
		errLog:           &errorLogger,
		lock:             lock,
		isRunning:        false,
		generationPrefix: "[Merkle Tree]",
		m:                m,
	}
}

// Submit rewards Merkle Tree
func (t *SubmitRewardsTree_Stateless) Run(nodeTrusted bool, state *state.NetworkState, beaconSlot uint64) error {
	// Check node trusted status
	if !nodeTrusted {
		if t.cfg.Smartnode.RewardsTreeMode.Value.(cfgtypes.RewardsMode) != cfgtypes.RewardsMode_Generate {
			return nil
		} else {
			// Create the state, since it's not done except for manual generators
			var err error
			state, err = t.m.GetStateForSlot(beaconSlot)
			if err != nil {
				return fmt.Errorf("error getting state for beacon slot %d: %w", beaconSlot, err)
			}
		}
	}

	// Log
	t.log.Println("Checking for rewards checkpoint...")

	// Check if a rewards interval has passed and needs to be calculated
	startTime := state.NetworkDetails.IntervalStart
	intervalTime := state.NetworkDetails.IntervalDuration

	// Calculate the end time, which is the number of intervals that have gone by since the current one's start
	genesisTime := time.Unix(int64(state.BeaconConfig.GenesisTime), 0)
	secondsSinceGenesis := time.Duration(state.BeaconConfig.SecondsPerSlot*state.BeaconSlotNumber) * time.Second
	stateTime := genesisTime.Add(secondsSinceGenesis)
	timeSinceStart := stateTime.Sub(startTime)
	intervalsPassed := timeSinceStart / intervalTime
	endTime := startTime.Add(intervalTime * intervalsPassed)
	if intervalsPassed == 0 {
		return nil
	}

	// Get the block and timestamp of the consensus block that best matches the end time
	snapshotBeaconBlock, elBlockNumber, err := t.getSnapshotConsensusBlock(endTime, state)
	if err != nil {
		return err
	}

	// Get the number of the EL block matching the CL snapshot block
	snapshotElBlockHeader, err := t.ec.HeaderByNumber(context.Background(), big.NewInt(int64(elBlockNumber)))
	if err != nil {
		return err
	}
	elBlockIndex := snapshotElBlockHeader.Number.Uint64()

	// Get the current interval
	currentIndex := state.NetworkDetails.RewardIndex
	currentIndexBig := big.NewInt(0).SetUint64(currentIndex)

	// Check if rewards generation is already running
	t.lock.Lock()
	if t.isRunning {
		t.log.Println("Tree generation is already running in the background.")
		t.lock.Unlock()
		return nil
	}
	t.lock.Unlock()

	// Get services
	t.cfg = t.sp.GetConfig()
	t.w = t.sp.GetWallet()
	t.rp = t.sp.GetRocketPool()
	t.ec = t.sp.GetEthClient()
	t.bc = t.sp.GetBeaconClient()
	nodeAddress, _ := t.w.GetAddress()
	t.rewardsPool, err = rewards.NewRewardsPool(t.rp)
	if err != nil {
		t.handleError(fmt.Errorf("error creating Rewards Pool binding: %w", err))
		return nil
	}

	// Get the expected file paths
	rewardsTreePath := t.cfg.Smartnode.GetRewardsTreePath(currentIndex, true)
	compressedRewardsTreePath := rewardsTreePath + config.RewardsTreeIpfsExtension
	minipoolPerformancePath := t.cfg.Smartnode.GetMinipoolPerformancePath(currentIndex, true)
	compressedMinipoolPerformancePath := minipoolPerformancePath + config.RewardsTreeIpfsExtension

	// Check if we can reuse an existing file for this interval
	if t.isExistingFileValid(rewardsTreePath, uint64(intervalsPassed)) {
		if !nodeTrusted {
			t.log.Printlnf("Merkle rewards tree for interval %d already exists at %s.", currentIndex, rewardsTreePath)
			return nil
		}

		// Return if this node has already submitted the tree for the current interval and there's a file present
		hasSubmitted, err := t.hasSubmittedTree(nodeAddress, currentIndexBig)
		if err != nil {
			return fmt.Errorf("error checking if Merkle tree submission has already been processed: %w", err)
		}
		if hasSubmitted {
			return nil
		}

		t.log.Printlnf("Merkle rewards tree for interval %d already exists at %s, attempting to resubmit...", currentIndex, rewardsTreePath)

		// Deserialize the file
		wrapperBytes, err := os.ReadFile(rewardsTreePath)
		if err != nil {
			return fmt.Errorf("error reading rewards tree file: %w", err)
		}

		proofWrapper, err := rprewards.DeserializeRewardsFile(wrapperBytes)
		if err != nil {
			return fmt.Errorf("error deserializing rewards tree file: %w", err)
		}

		// Upload the file
		cid, err := t.uploadFileToWeb3Storage(wrapperBytes, compressedRewardsTreePath, "compressed rewards tree")
		if err != nil {
			return fmt.Errorf("error uploading Merkle tree to Web3.Storage: %w", err)
		}
		t.log.Printlnf("Uploaded Merkle tree with CID %s", cid)

		// Submit to the contracts
		err = t.submitRewardsSnapshot(currentIndexBig, snapshotBeaconBlock, elBlockIndex, proofWrapper.GetHeader(), cid, big.NewInt(int64(intervalsPassed)))
		if err != nil {
			return fmt.Errorf("error submitting rewards snapshot: %w", err)
		}

		t.log.Printlnf("Successfully submitted rewards snapshot for interval %d.", currentIndex)
		return nil
	}

	// Generate the tree
	t.generateTree(intervalsPassed, nodeTrusted, currentIndex, snapshotBeaconBlock, elBlockIndex, startTime, endTime, snapshotElBlockHeader, rewardsTreePath, compressedRewardsTreePath, minipoolPerformancePath, compressedMinipoolPerformancePath)

	// Done
	return nil
}

func (t *SubmitRewardsTree_Stateless) handleError(err error) {
	t.errLog.Println(fmt.Errorf("%s %w", t.generationPrefix, err))
	t.errLog.Println("*** Rewards tree generation failed. ***")
	t.lock.Lock()
	t.isRunning = false
	t.lock.Unlock()
}

// Print a message from the tree generation goroutine
func (t *SubmitRewardsTree_Stateless) printMessage(message string) {
	t.log.Printlnf("%s %s", t.generationPrefix, message)
}

// Checks to see if an existing rewards file is still valid
func (t *SubmitRewardsTree_Stateless) isExistingFileValid(rewardsTreePath string, intervalsPassed uint64) bool {
	_, err := os.Stat(rewardsTreePath)
	if !os.IsNotExist(err) {
		// The file already exists, attempt to read it
		fileBytes, err := os.ReadFile(rewardsTreePath)
		if err != nil {
			t.log.Printlnf("WARNING: failed to read %s: %s\nRegenerating file...\n", rewardsTreePath, err.Error())
			return false
		}

		proofWrapper, err := rprewards.DeserializeRewardsFile(fileBytes)
		if err != nil {
			t.log.Printlnf("WARNING: failed to deserialize %s: %s\nRegenerating file...\n", rewardsTreePath, err.Error())
			return false
		}

		// Compare the number of intervals in it with the current number of intervals
		header := proofWrapper.GetHeader()
		if header.IntervalsPassed != intervalsPassed {
			t.log.Printlnf("Existing file for interval %d had %d intervals passed but %d have passed now, regenerating file...\n", header.Index, header.IntervalsPassed, intervalsPassed)
			return false
		}

		// File's good and it has the same number of intervals passed, so use it
		return true
	}

	return false
}

// Kick off the tree generation goroutine
func (t *SubmitRewardsTree_Stateless) generateTree(intervalsPassed time.Duration, nodeTrusted bool, currentIndex uint64, snapshotBeaconBlock uint64, elBlockIndex uint64, startTime time.Time, endTime time.Time, snapshotElBlockHeader *types.Header, rewardsTreePath string, compressedRewardsTreePath string, minipoolPerformancePath string, compressedMinipoolPerformancePath string) {
	go func() {
		t.lock.Lock()
		t.isRunning = true
		t.lock.Unlock()

		// Get an appropriate client
		client, err := eth1.GetBestApiClient(t.rp, t.cfg, t.printMessage, snapshotElBlockHeader.Number)
		if err != nil {
			t.handleError(err)
			return
		}

		// Generate the tree
		err = t.generateTreeImpl(client, intervalsPassed, nodeTrusted, currentIndex, snapshotBeaconBlock, elBlockIndex, startTime, endTime, snapshotElBlockHeader, rewardsTreePath, compressedRewardsTreePath, minipoolPerformancePath, compressedMinipoolPerformancePath)
		if err != nil {
			t.handleError(err)
		}

		t.lock.Lock()
		t.isRunning = false
		t.lock.Unlock()
	}()
}

// Implementation for rewards tree generation using a viable EC
func (t *SubmitRewardsTree_Stateless) generateTreeImpl(rp *rocketpool.RocketPool, intervalsPassed time.Duration, nodeTrusted bool, currentIndex uint64, snapshotBeaconBlock uint64, elBlockIndex uint64, startTime time.Time, endTime time.Time, snapshotElBlockHeader *types.Header, rewardsTreePath string, compressedRewardsTreePath string, minipoolPerformancePath string, compressedMinipoolPerformancePath string) error {
	// Log
	if uint64(intervalsPassed) > 1 {
		t.log.Printlnf("WARNING: %d intervals have passed since the last rewards checkpoint was submitted! Rolling them into one...", uint64(intervalsPassed))
	}
	t.log.Printlnf("Rewards checkpoint has passed, starting Merkle tree generation for interval %d in the background.\n%s Snapshot Beacon block = %d, EL block = %d, running from %s to %s", currentIndex, t.generationPrefix, snapshotBeaconBlock, elBlockIndex, startTime, endTime)

	// Create a new state gen manager
	mgr, err := state.NewNetworkStateManager(rp, t.cfg, rp.Client, t.bc, t.log)
	if err != nil {
		return fmt.Errorf("error creating network state manager for EL block %d, Beacon slot %d: %w", elBlockIndex, snapshotBeaconBlock, err)
	}

	// Create a new state for the target block
	state, err := mgr.GetStateForSlot(snapshotBeaconBlock)
	if err != nil {
		return fmt.Errorf("couldn't get network state for EL block %d, Beacon slot %d: %w", elBlockIndex, snapshotBeaconBlock, err)
	}

	// Generate the rewards file
	treegen, err := rprewards.NewTreeGenerator(t.log, t.generationPrefix, rp, t.cfg, t.bc, currentIndex, startTime, endTime, snapshotBeaconBlock, snapshotElBlockHeader, uint64(intervalsPassed), state, nil)
	if err != nil {
		return fmt.Errorf("error creating Merkle tree generator: %w", err)
	}
	rewardsFile, err := treegen.GenerateTree()
	if err != nil {
		return fmt.Errorf("error generating Merkle tree: %w", err)
	}
	for address, network := range rewardsFile.GetHeader().InvalidNetworkNodes {
		t.printMessage(fmt.Sprintf("WARNING: Node %s has invalid network %d assigned! Using 0 (mainnet) instead.", address.Hex(), network))
	}

	// Serialize the minipool performance file
	minipoolPerformanceBytes, err := rewardsFile.GetMinipoolPerformanceFile().Serialize()
	if err != nil {
		return fmt.Errorf("error serializing minipool performance file into JSON: %w", err)
	}

	// Write it to disk
	err = os.WriteFile(minipoolPerformancePath, minipoolPerformanceBytes, 0644)
	if err != nil {
		return fmt.Errorf("error saving minipool performance file to %s: %w", minipoolPerformancePath, err)
	}

	// Upload it if this is an Oracle DAO node
	if nodeTrusted {
		t.printMessage("Uploading minipool performance file to Web3.Storage...")
		minipoolPerformanceCid, err := t.uploadFileToWeb3Storage(minipoolPerformanceBytes, compressedMinipoolPerformancePath, "compressed minipool performance")
		if err != nil {
			return fmt.Errorf("error uploading minipool performance file to Web3.Storage: %w", err)
		}
		t.printMessage(fmt.Sprintf("Uploaded minipool performance file with CID %s", minipoolPerformanceCid))
		rewardsFile.SetMinipoolPerformanceFileCID(minipoolPerformanceCid)
	} else {
		t.printMessage("Saved minipool performance file.")
		rewardsFile.SetMinipoolPerformanceFileCID("---")
	}

	// Serialize the rewards tree to JSON
	wrapperBytes, err := rewardsFile.Serialize()
	if err != nil {
		return fmt.Errorf("error serializing proof wrapper into JSON: %w", err)
	}
	t.printMessage("Generation complete! Saving tree...")

	// Write the rewards tree to disk
	err = os.WriteFile(rewardsTreePath, wrapperBytes, 0644)
	if err != nil {
		return fmt.Errorf("error saving rewards tree file to %s: %w", rewardsTreePath, err)
	}

	// Only do the upload and submission process if this is an Oracle DAO node
	if nodeTrusted {
		// Upload the rewards tree file
		t.printMessage("Uploading to Web3.Storage and submitting results to the contracts...")
		cid, err := t.uploadFileToWeb3Storage(wrapperBytes, compressedRewardsTreePath, "compressed rewards tree")
		if err != nil {
			return fmt.Errorf("error uploading Merkle tree to Web3.Storage: %w", err)
		}
		t.printMessage(fmt.Sprintf("Uploaded Merkle tree with CID %s", cid))

		// Submit to the contracts
		err = t.submitRewardsSnapshot(big.NewInt(int64(currentIndex)), snapshotBeaconBlock, elBlockIndex, rewardsFile.GetHeader(), cid, big.NewInt(int64(intervalsPassed)))
		if err != nil {
			return fmt.Errorf("error submitting rewards snapshot: %w", err)
		}

		t.printMessage(fmt.Sprintf("Successfully submitted rewards snapshot for interval %d.", currentIndex))
	} else {
		t.printMessage(fmt.Sprintf("Successfully generated rewards snapshot for interval %d.", currentIndex))
	}

	return nil
}

// Submit rewards info to the contracts
func (t *SubmitRewardsTree_Stateless) submitRewardsSnapshot(index *big.Int, consensusBlock uint64, executionBlock uint64, rewardsFileHeader *sharedtypes.RewardsFileHeader, cid string, intervalsPassed *big.Int) error {
	treeRootBytes, err := hex.DecodeString(hexutil.RemovePrefix(rewardsFileHeader.MerkleRoot))
	if err != nil {
		return fmt.Errorf("error decoding merkle root: %w", err)
	}
	treeRoot := common.BytesToHash(treeRootBytes)

	// Create the arrays of rewards per network
	collateralRplRewards := []*big.Int{}
	oDaoRplRewards := []*big.Int{}
	smoothingPoolEthRewards := []*big.Int{}

	// Create the total rewards for each network
	network := uint64(0)
	for {
		networkRewards, exists := rewardsFileHeader.NetworkRewards[network]
		if !exists {
			break
		}

		collateralRplRewards = append(collateralRplRewards, &networkRewards.CollateralRpl.Int)
		oDaoRplRewards = append(oDaoRplRewards, &networkRewards.OracleDaoRpl.Int)
		smoothingPoolEthRewards = append(smoothingPoolEthRewards, &networkRewards.SmoothingPoolEth.Int)

		network++
	}

	// Get transactor
	opts, err := t.w.GetTransactor()
	if err != nil {
		return err
	}

	// Create the submission
	submission := rewards.RewardSubmission{
		RewardIndex:     index,
		ExecutionBlock:  big.NewInt(0).SetUint64(executionBlock),
		ConsensusBlock:  big.NewInt(0).SetUint64(consensusBlock),
		MerkleRoot:      treeRoot,
		MerkleTreeCID:   cid,
		IntervalsPassed: intervalsPassed,
		TreasuryRPL:     &rewardsFileHeader.TotalRewards.ProtocolDaoRpl.Int,
		NodeRPL:         collateralRplRewards,
		TrustedNodeRPL:  oDaoRplRewards,
		NodeETH:         smoothingPoolEthRewards,
		UserETH:         &rewardsFileHeader.TotalRewards.PoolStakerSmoothingPoolEth.Int,
	}

	// Get the tx info
	txInfo, err := t.rewardsPool.SubmitRewardSnapshot(submission, opts)
	if err != nil {
		if enableSubmissionAfterConsensus_RewardsTree && strings.Contains(err.Error(), "Can only submit snapshot for next period") {
			// Set a gas limit which will intentionally be too low and revert
			txInfo.GasInfo = core.GasInfo{
				EstGasLimit:  utils.RewardsSubmissionForcedGas,
				SafeGasLimit: utils.RewardsSubmissionForcedGas,
			}
			t.log.Println("Rewards period consensus has already been reached but submitting anyway for the health check.")
		} else {
			return fmt.Errorf("error getting TX for submitting the rewards tree: %w", err)
		}
	}
	if txInfo.SimError != "" {
		return fmt.Errorf("simulating TX for submitting the rewards tree failed: %s", txInfo.SimError)
	}

	// Print the gas info
	maxFee := eth.GweiToWei(utils.GetWatchtowerMaxFee(t.cfg))
	if !gas.PrintAndCheckGasInfo(txInfo.GasInfo, false, 0, t.log, maxFee, 0) {
		return nil
	}

	opts.GasFeeCap = maxFee
	opts.GasTipCap = eth.GweiToWei(utils.GetWatchtowerPrioFee(t.cfg))
	opts.GasLimit = txInfo.GasInfo.SafeGasLimit

	// Print TX info and wait for it to be included in a block
	err = tx.PrintAndWaitForTransaction(t.cfg, t.rp, t.log, txInfo, opts)
	if err != nil {
		return err
	}

	// Return
	return nil
}

// Compress and upload a file to Web3.Storage and get the CID for it
func (t *SubmitRewardsTree_Stateless) uploadFileToWeb3Storage(wrapperBytes []byte, compressedPath string, description string) (string, error) {
	// Get the API token
	apiToken := t.cfg.Smartnode.Web3StorageApiToken.Value.(string)
	if apiToken == "" {
		return "", fmt.Errorf("***ERROR***\nYou have not configured your Web3.Storage API token yet, so you cannot submit Merkle rewards trees.\nPlease get an API token from https://web3.storage and enter it in the Smartnode section of the `service config` TUI (or use `--smartnode-web3StorageApiToken` if you configure your system headlessly).")
	}

	// Create the client
	w3sClient, err := w3s.NewClient(w3s.WithToken(apiToken))
	if err != nil {
		return "", fmt.Errorf("error creating new Web3.Storage client: %w", err)
	}

	// Compress the file
	encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	compressedBytes := encoder.EncodeAll(wrapperBytes, make([]byte, 0, len(wrapperBytes)))

	// Create the compressed tree file
	compressedFile, err := os.Create(compressedPath)
	if err != nil {
		return "", fmt.Errorf("error creating %s file [%s]: %w", description, compressedPath, err)
	}
	defer compressedFile.Close()

	// Write the compressed data to the file
	_, err = compressedFile.Write(compressedBytes)
	if err != nil {
		return "", fmt.Errorf("error writing %s to %s: %w", description, compressedPath, err)
	}

	// Rewind it to the start
	compressedFile.Seek(0, 0)

	// Upload it
	cid, err := w3sClient.Put(context.Background(), compressedFile)
	if err != nil {
		return "", fmt.Errorf("error uploading %s: %w", description, err)
	}

	return cid.String(), nil
}

// Get the first finalized, successful consensus block that occurred after the given target time
func (t *SubmitRewardsTree_Stateless) getSnapshotConsensusBlock(endTime time.Time, state *state.NetworkState) (uint64, uint64, error) {
	// Get the beacon head
	beaconHead, err := t.bc.GetBeaconHead()
	if err != nil {
		return 0, 0, fmt.Errorf("error getting Beacon head: %w", err)
	}

	// Get the target block number
	eth2Config := state.BeaconConfig
	genesisTime := time.Unix(int64(eth2Config.GenesisTime), 0)
	totalTimespan := endTime.Sub(genesisTime)
	targetSlot := uint64(math.Ceil(totalTimespan.Seconds() / float64(eth2Config.SecondsPerSlot)))
	targetSlotEpoch := targetSlot / eth2Config.SlotsPerEpoch
	targetSlot = targetSlotEpoch*eth2Config.SlotsPerEpoch + (eth2Config.SlotsPerEpoch - 1) // The target slot becomes the last one in the Epoch
	requiredEpoch := targetSlotEpoch + 1                                                   // The smoothing pool requires 1 epoch beyond the target to be finalized, to check for late attestations

	// Check if the required epoch is finalized yet
	if beaconHead.FinalizedEpoch < requiredEpoch {
		return 0, 0, fmt.Errorf("Snapshot end time = %s, slot (epoch) = %d (%d)... waiting until epoch %d is finalized (currently %d).", endTime, targetSlot, targetSlotEpoch, requiredEpoch, beaconHead.FinalizedEpoch)
	}

	// Get the first successful block
	for {
		// Try to get the current block
		block, exists, err := t.bc.GetBeaconBlock(fmt.Sprint(targetSlot))
		if err != nil {
			return 0, 0, fmt.Errorf("error getting Beacon block %d: %w", targetSlot, err)
		}

		// If the block was missing, try the previous one
		if !exists {
			t.log.Printlnf("Slot %d was missing, trying the previous one...", targetSlot)
			targetSlot--
		} else {
			// Ok, we have the first proposed finalized block - this is the one to use for the snapshot!
			return targetSlot, block.ExecutionBlockNumber, nil
		}
	}
}

// Check whether the rewards tree for the current interval been submitted by the node
func (t *SubmitRewardsTree_Stateless) hasSubmittedTree(nodeAddress common.Address, index *big.Int) (bool, error) {
	indexBuffer := make([]byte, 32)
	index.FillBytes(indexBuffer)
	var result bool
	err := t.rp.Query(func(mc *batch.MultiCaller) error {
		t.rp.Storage.GetBool(mc, &result, crypto.Keccak256Hash([]byte("rewards.snapshot.submitted.node"), nodeAddress.Bytes(), indexBuffer))
		return nil
	}, nil)
	return result, err
}
