package node

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	batch "github.com/rocket-pool/batch-query"
	"github.com/rocket-pool/rocketpool-go/core"
	"github.com/rocket-pool/rocketpool-go/dao/oracle"
	"github.com/rocket-pool/rocketpool-go/dao/protocol"
	"github.com/rocket-pool/rocketpool-go/deposit"
	"github.com/rocket-pool/rocketpool-go/minipool"
	"github.com/rocket-pool/rocketpool-go/node"
	"github.com/rocket-pool/rocketpool-go/rocketpool"
	rptypes "github.com/rocket-pool/rocketpool-go/types"
	"github.com/rocket-pool/rocketpool-go/utils/eth"

	prdeposit "github.com/prysmaticlabs/prysm/v3/contracts/deposit"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/rocket-pool/smartnode/rocketpool/common/beacon"
	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	"github.com/rocket-pool/smartnode/rocketpool/common/wallet"
	rputils "github.com/rocket-pool/smartnode/rocketpool/utils/rp"
	"github.com/rocket-pool/smartnode/shared/config"
	"github.com/rocket-pool/smartnode/shared/types/api"
	"github.com/rocket-pool/smartnode/shared/utils/input"
	sharedutils "github.com/rocket-pool/smartnode/shared/utils/rp"
	"github.com/rocket-pool/smartnode/shared/utils/validator"
	eth2types "github.com/wealdtech/go-eth2-types/v2"
)

// ===============
// === Factory ===
// ===============

type nodeDepositContextFactory struct {
	handler *NodeHandler
}

func (f *nodeDepositContextFactory) Create(vars map[string]string) (*nodeDepositContext, error) {
	c := &nodeDepositContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArg("amount-wei", vars, input.ValidateBigInt, &c.amountWei),
		server.ValidateArg("min-node-fee", vars, input.ValidateFraction, &c.minNodeFee),
		server.ValidateArg("salt", vars, input.ValidateBigInt, &c.salt),
	}
	return c, errors.Join(inputErrs...)
}

func (f *nodeDepositContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterSingleStageRoute[*nodeDepositContext, api.NodeDepositData](
		router, "deposit", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type nodeDepositContext struct {
	handler *NodeHandler
	cfg     *config.RocketPoolConfig
	rp      *rocketpool.RocketPool
	bc      beacon.Client
	w       *wallet.LocalWallet

	amountWei   *big.Int
	minNodeFee  float64
	salt        *big.Int
	node        *node.Node
	depositPool *deposit.DepositPoolManager
	pSettings   *protocol.ProtocolDaoSettings
	oSettings   *oracle.OracleDaoSettings
	mpMgr       *minipool.MinipoolManager
}

func (c *nodeDepositContext) Initialize() error {
	sp := c.handler.serviceProvider
	c.cfg = sp.GetConfig()
	c.rp = sp.GetRocketPool()
	c.bc = sp.GetBeaconClient()
	c.w = sp.GetWallet()
	nodeAddress, _ := c.w.GetAddress()

	// Requirements
	err := errors.Join(
		sp.RequireNodeRegistered(),
		sp.RequireWalletReady(),
	)
	if err != nil {
		return err
	}

	// Bindings
	c.node, err = node.NewNode(c.rp, nodeAddress)
	if err != nil {
		return fmt.Errorf("error creating node %s binding: %w", nodeAddress.Hex(), err)
	}
	c.depositPool, err = deposit.NewDepositPoolManager(c.rp)
	if err != nil {
		return fmt.Errorf("error getting deposit pool binding: %w", err)
	}
	pMgr, err := protocol.NewProtocolDaoManager(c.rp)
	if err != nil {
		return fmt.Errorf("error getting pDAO manager binding: %w", err)
	}
	c.pSettings = pMgr.Settings
	oMgr, err := oracle.NewOracleDaoManager(c.rp)
	if err != nil {
		return fmt.Errorf("error getting oDAO manager binding: %w", err)
	}
	c.oSettings = oMgr.Settings
	c.mpMgr, err = minipool.NewMinipoolManager(c.rp)
	if err != nil {
		return fmt.Errorf("error getting minipool manager binding: %w", err)
	}
	return nil
}

func (c *nodeDepositContext) GetState(mc *batch.MultiCaller) {
	core.AddQueryablesToMulticall(mc,
		c.node.Credit,
		c.depositPool.Balance,
		c.pSettings.Node.IsDepositingEnabled,
		c.oSettings.Minipool.ScrubPeriod,
	)
}

func (c *nodeDepositContext) PrepareData(data *api.NodeDepositData, opts *bind.TransactOpts) error {
	// Initial population
	data.CreditBalance = c.node.Credit.Get()
	data.DepositDisabled = !c.pSettings.Node.IsDepositingEnabled.Get()
	data.DepositBalance = c.depositPool.Balance.Get()
	data.ScrubPeriod = c.oSettings.Minipool.ScrubPeriod.Formatted()

	// Get Beacon config
	eth2Config, err := c.bc.GetEth2Config()
	if err != nil {
		return fmt.Errorf("error getting Beacon config: %w", err)
	}

	// Adjust the salt
	if c.salt.Cmp(big.NewInt(0)) == 0 {
		nonce, err := c.rp.Client.NonceAt(context.Background(), c.node.Address, nil)
		if err != nil {
			return fmt.Errorf("error getting node's latest nonce: %w", err)
		}
		c.salt.SetUint64(nonce)
	}

	// Check node balance
	data.NodeBalance, err = c.rp.Client.BalanceAt(context.Background(), c.node.Address, nil)
	if err != nil {
		return fmt.Errorf("error getting node's ETH balance: %w", err)
	}

	// Check the node's collateral
	collateral, err := sharedutils.CheckCollateral(c.rp, c.node.Address, nil)
	if err != nil {
		return fmt.Errorf("error checking node collateral: %w", err)
	}
	ethMatched := collateral.EthMatched
	ethMatchedLimit := collateral.EthMatchedLimit
	pendingMatchAmount := collateral.PendingMatchAmount

	// Check for insufficient balance
	totalBalance := big.NewInt(0).Add(data.NodeBalance, data.CreditBalance)
	data.InsufficientBalance = (c.amountWei.Cmp(totalBalance) > 0)

	// Check if the credit balance can be used
	data.CanUseCredit = (data.DepositBalance.Cmp(eth.EthToWei(1)) >= 0)

	// Check data
	validatorEthWei := eth.EthToWei(ValidatorEth)
	matchRequest := big.NewInt(0).Sub(validatorEthWei, c.amountWei)
	availableToMatch := big.NewInt(0).Sub(ethMatchedLimit, ethMatched)
	availableToMatch.Sub(availableToMatch, pendingMatchAmount)
	data.InsufficientRplStake = (availableToMatch.Cmp(matchRequest) == -1)

	// Update response
	data.CanDeposit = !(data.InsufficientBalance || data.InsufficientRplStake || data.InvalidAmount || data.DepositDisabled)
	if data.CanDeposit && !data.CanUseCredit && data.NodeBalance.Cmp(c.amountWei) < 0 {
		// Can't use credit and there's not enough ETH in the node wallet to deposit so error out
		data.InsufficientBalanceWithoutCredit = true
		data.CanDeposit = false
	}

	// Return if depositing won't work
	if !data.CanDeposit {
		return nil
	}

	// Make sure ETH2 is on the correct chain
	depositContractInfo, err := rputils.GetDepositContractInfo(c.rp, c.cfg, c.bc)
	if err != nil {
		return fmt.Errorf("error verifying the EL and BC are on the same chain: %w", err)
	}
	if depositContractInfo.RPNetwork != depositContractInfo.BeaconNetwork ||
		depositContractInfo.RPDepositContract != depositContractInfo.BeaconDepositContract {
		return fmt.Errorf("FATAL: Beacon network mismatch! Expected %s on chain %d, but beacon is using %s on chain %d.",
			depositContractInfo.RPDepositContract.Hex(),
			depositContractInfo.RPNetwork,
			depositContractInfo.BeaconDepositContract.Hex(),
			depositContractInfo.BeaconNetwork)
	}

	// Get how much credit to use
	if data.CanUseCredit {
		remainingAmount := big.NewInt(0).Sub(c.amountWei, data.CreditBalance)
		if remainingAmount.Cmp(big.NewInt(0)) > 0 {
			// Send the remaining amount if the credit isn't enough to cover the whole deposit
			opts.Value = remainingAmount
		}
	} else {
		opts.Value = c.amountWei
	}

	// Get the next available validator key without saving it
	validatorKey, index, err := c.w.GetNextValidatorKey()
	if err != nil {
		return fmt.Errorf("error getting next available validator key: %w", err)
	}
	data.Index = index

	// Get the next minipool address
	var minipoolAddress common.Address
	err = c.rp.Query(func(mc *batch.MultiCaller) error {
		c.node.GetExpectedMinipoolAddress(mc, &minipoolAddress, c.salt)
		return nil
	}, nil)
	if err != nil {
		return fmt.Errorf("error getting expected minipool address: %w", err)
	}
	data.MinipoolAddress = minipoolAddress

	// Get the withdrawal credentials
	var withdrawalCredentials common.Hash
	err = c.rp.Query(func(mc *batch.MultiCaller) error {
		c.mpMgr.GetMinipoolWithdrawalCredentials(mc, &withdrawalCredentials, minipoolAddress)
		return nil
	}, nil)
	if err != nil {
		return fmt.Errorf("error getting minipool withdrawal credentials: %w", err)
	}

	// Get validator deposit data and associated parameters
	depositAmount := uint64(1e9) // 1 ETH in gwei
	depositData, depositDataRoot, err := validator.GetDepositData(validatorKey, withdrawalCredentials, eth2Config, depositAmount)
	if err != nil {
		return fmt.Errorf("error getting deposit data: %w", err)
	}
	pubkey := rptypes.BytesToValidatorPubkey(depositData.PublicKey)
	signature := rptypes.BytesToValidatorSignature(depositData.Signature)
	data.ValidatorPubkey = pubkey

	// Make sure a validator with this pubkey doesn't already exist
	status, err := c.bc.GetValidatorStatus(pubkey, nil)
	if err != nil {
		return fmt.Errorf("Error checking for existing validator status: %w\nYour funds have not been deposited for your own safety.", err)
	}
	if status.Exists {
		return fmt.Errorf("**** ALERT ****\n"+
			"Your minipool %s has the following as a validator pubkey:\n\t%s\n"+
			"This key is already in use by validator %d on the Beacon chain!\n"+
			"Rocket Pool will not allow you to deposit this validator for your own safety so you do not get slashed.\n"+
			"PLEASE REPORT THIS TO THE ROCKET POOL DEVELOPERS.\n"+
			"***************\n", minipoolAddress.Hex(), pubkey.Hex(), status.Index)
	}

	// Do a final sanity check
	err = validateDepositInfo(eth2Config, uint64(depositAmount), pubkey, withdrawalCredentials, signature)
	if err != nil {
		return fmt.Errorf("FATAL: Your deposit failed the validation safety check: %w\n"+
			"For your safety, this deposit will not be submitted and your ETH will not be staked.\n"+
			"PLEASE REPORT THIS TO THE ROCKET POOL DEVELOPERS and include the following information:\n"+
			"\tDomain Type: 0x%s\n"+
			"\tGenesis Fork Version: 0x%s\n"+
			"\tGenesis Validator Root: 0x%s\n"+
			"\tDeposit Amount: %d gwei\n"+
			"\tValidator Pubkey: %s\n"+
			"\tWithdrawal Credentials: %s\n"+
			"\tSignature: %s\n",
			err,
			hex.EncodeToString(eth2types.DomainDeposit[:]),
			hex.EncodeToString(eth2Config.GenesisForkVersion),
			hex.EncodeToString(eth2types.ZeroGenesisValidatorsRoot),
			depositAmount,
			pubkey.Hex(),
			withdrawalCredentials.Hex(),
			signature.Hex(),
		)
	}

	// Get tx info
	var txInfo *core.TransactionInfo
	var funcName string
	if data.CanUseCredit {
		txInfo, err = c.node.DepositWithCredit(c.amountWei, c.minNodeFee, pubkey, signature, depositDataRoot, c.salt, minipoolAddress, opts)
		funcName = "DepositWithCredit"
	} else {
		txInfo, err = c.node.Deposit(c.amountWei, c.minNodeFee, pubkey, signature, depositDataRoot, c.salt, minipoolAddress, opts)
		funcName = "Deposit"
	}
	if err != nil {
		return fmt.Errorf("error getting TX info for %s: %w", funcName, err)
	}
	data.TxInfo = txInfo

	return nil
}

func validateDepositInfo(eth2Config beacon.Eth2Config, depositAmount uint64, pubkey rptypes.ValidatorPubkey, withdrawalCredentials common.Hash, signature rptypes.ValidatorSignature) error {

	// Get the deposit domain based on the eth2 config
	depositDomain, err := signing.ComputeDomain(eth2types.DomainDeposit, eth2Config.GenesisForkVersion, eth2types.ZeroGenesisValidatorsRoot)
	if err != nil {
		return err
	}

	// Create the deposit struct
	depositData := new(ethpb.Deposit_Data)
	depositData.Amount = depositAmount
	depositData.PublicKey = pubkey.Bytes()
	depositData.WithdrawalCredentials = withdrawalCredentials.Bytes()
	depositData.Signature = signature.Bytes()

	// Validate the signature
	err = prdeposit.VerifyDepositSignature(depositData, depositDomain)
	return err

}
