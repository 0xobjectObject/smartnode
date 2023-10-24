package rp

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	batch "github.com/rocket-pool/batch-query"
	"github.com/rocket-pool/rocketpool-go/core"
	"github.com/rocket-pool/rocketpool-go/dao/oracle"
	"github.com/rocket-pool/rocketpool-go/minipool"
	"github.com/rocket-pool/rocketpool-go/node"
	"github.com/rocket-pool/rocketpool-go/rocketpool"
	"github.com/rocket-pool/rocketpool-go/types"
	"github.com/rocket-pool/smartnode/shared/services/beacon"
)

const (
	minipoolPubkeyBatchSize        int = 500
	minipoolReduceDetailsBatchSize int = 200
)

func GetNodeValidatorIndices(rp *rocketpool.RocketPool, bc beacon.Client, nodeAddress common.Address, opts *bind.CallOpts) ([]string, error) {
	// Create the bindings
	node, err := node.NewNode(rp, nodeAddress)
	if err != nil {
		return nil, fmt.Errorf("error getting node %s binding: %w", nodeAddress.Hex(), err)
	}
	mpMgr, err := minipool.NewMinipoolManager(rp)
	if err != nil {
		return nil, fmt.Errorf("error creating minipool manager binding: %w", err)
	}

	// Get contract state
	err = rp.Query(func(mc *batch.MultiCaller) error {
		node.ValidatingMinipoolCount.AddToQuery(mc)
		return nil
	}, opts)
	if err != nil {
		return nil, fmt.Errorf("error getting contract state: %w", err)
	}

	// Get the validating addresses
	addresses, err := node.GetValidatingMinipoolAddresses(node.ValidatingMinipoolCount.Formatted(), opts)
	if err != nil {
		return nil, fmt.Errorf("error getting validating minipool addresses: %w", err)
	}

	// Create the minipools
	minipools, err := mpMgr.CreateMinipoolsFromAddresses(addresses, false, opts)
	if err != nil {
		return nil, fmt.Errorf("error getting validating minipools: %w", err)
	}

	// Get the list of pubkeys
	pubkeys := make([]types.ValidatorPubkey, len(addresses))
	err = rp.BatchQuery(len(addresses), minipoolPubkeyBatchSize, func(mc *batch.MultiCaller, i int) error {
		minipools[i].Common().Pubkey.AddToQuery(mc)
		return nil
	}, opts)
	if err != nil {
		return nil, fmt.Errorf("error getting validating pubkeys: %w", err)
	}

	// Populate the slice of pubkeys
	for i, mp := range minipools {
		pubkeys[i] = mp.Common().Pubkey.Get()
	}

	// Remove zero pubkeys
	zeroPubkey := types.ValidatorPubkey{}
	filteredPubkeys := []types.ValidatorPubkey{}
	for _, pubkey := range pubkeys {
		if !bytes.Equal(pubkey[:], zeroPubkey[:]) {
			filteredPubkeys = append(filteredPubkeys, pubkey)
		}
	}
	pubkeys = filteredPubkeys

	// Get validator statuses by pubkeys
	statuses, err := bc.GetValidatorStatuses(pubkeys, nil)
	if err != nil {
		return nil, fmt.Errorf("Error getting validator statuses: %w", err)
	}

	// Enumerate validators statuses and fill indices array
	validatorIndices := make([]string, 0, len(statuses)+1)
	for _, status := range statuses {
		validatorIndices = append(validatorIndices, status.Index)
	}
	return validatorIndices, nil
}

type CollateralAmounts struct {
	EthMatched         *big.Int
	EthMatchedLimit    *big.Int
	PendingMatchAmount *big.Int
}

// Checks the given node's current matched ETH, its limit on matched ETH, and how much ETH is preparing to be matched by pending bond reductions
func CheckCollateral(rp *rocketpool.RocketPool, nodeAddress common.Address, opts *bind.CallOpts) (*CollateralAmounts, error) {
	// Create the bindings
	node, err := node.NewNode(rp, nodeAddress)
	if err != nil {
		return nil, fmt.Errorf("error getting node %s binding: %w", nodeAddress.Hex(), err)
	}
	mpMgr, err := minipool.NewMinipoolManager(rp)
	if err != nil {
		return nil, fmt.Errorf("error getting minipool manager binding: %w", err)
	}

	// Get the minipool count
	err = rp.Query(nil, opts, node.MinipoolCount)
	if err != nil {
		return nil, fmt.Errorf("error getting minipool count: %w", err)
	}

	// Get the minipool addresses
	addresses, err := node.GetMinipoolAddresses(node.MinipoolCount.Formatted(), opts)
	if err != nil {
		return nil, fmt.Errorf("error getting minipool addresses: %w", err)
	}

	// Create the minipool bindings
	mps, err := mpMgr.CreateMinipoolsFromAddresses(addresses, false, opts)
	if err != nil {
		return nil, fmt.Errorf("error creating minipool bindings: %w", err)
	}

	// Get the minipool details
	err = rp.BatchQuery(len(addresses), minipoolReduceDetailsBatchSize, func(mc *batch.MultiCaller, i int) error {
		mpv3, isMpv3 := minipool.GetMinipoolAsV3(mps[i])
		if isMpv3 {
			core.AddQueryablesToMulticall(mc,
				mpv3.ReduceBondTime,
				mpv3.IsBondReduceCancelled,
				mpv3.NodeDepositBalance,
				mpv3.ReduceBondValue,
			)
		}
		return nil
	}, opts)
	if err != nil {
		return nil, fmt.Errorf("error getting minipool details: %w", err)
	}

	return CheckCollateralWithMinipoolCache(rp, nodeAddress, mps, opts)
}

// Checks the given node's current matched ETH, its limit on matched ETH, and how much ETH is preparing to be matched by pending bond reductions
func CheckCollateralWithMinipoolCache(rp *rocketpool.RocketPool, nodeAddress common.Address, minipools []minipool.IMinipool, opts *bind.CallOpts) (*CollateralAmounts, error) {
	// Get the relevant header
	var blockHeader *ethtypes.Header
	var err error
	if opts != nil {
		blockHeader, err = rp.Client.HeaderByNumber(context.Background(), opts.BlockNumber)
	} else {
		blockHeader, err = rp.Client.HeaderByNumber(context.Background(), nil)
	}
	if err != nil {
		return nil, fmt.Errorf("error getting latest block header: %w", err)
	}

	// Get the time and set up opts from the header
	blockTime := time.Unix(int64(blockHeader.Time), 0)
	if opts == nil {
		opts = &bind.CallOpts{
			BlockNumber: blockHeader.Number,
		}
	}

	// Create the bindings
	node, err := node.NewNode(rp, nodeAddress)
	if err != nil {
		return nil, fmt.Errorf("error getting node %s binding: %w", nodeAddress.Hex(), err)
	}
	oMgr, err := oracle.NewOracleDaoManager(rp)
	if err != nil {
		return nil, fmt.Errorf("error getting oracle DAO manager binding: %w", err)
	}
	oSettings := oMgr.Settings

	// Get contract state
	err = rp.Query(func(mc *batch.MultiCaller) error {
		core.AddQueryablesToMulticall(mc,
			node.EthMatched,
			node.EthMatchedLimit,
			oSettings.Minipool.BondReductionWindowStart,
			oSettings.Minipool.BondReductionWindowLength,
		)
		return nil
	}, opts)
	if err != nil {
		return nil, fmt.Errorf("error getting contract state: %w", err)
	}

	reductionWindowStart := oSettings.Minipool.BondReductionWindowStart.Formatted()
	reductionWindowLength := oSettings.Minipool.BondReductionWindowLength.Formatted()
	reductionWindowEnd := reductionWindowStart + reductionWindowLength

	// Calculate the deltas
	totalDelta := big.NewInt(0)
	zeroTime := time.Unix(0, 0)
	for _, mp := range minipools {
		mpv3, isMpv3 := minipool.GetMinipoolAsV3(mp)
		if !isMpv3 {
			continue
		}
		mpCommon := mp.Common()
		reduceBondTime := mpv3.ReduceBondTime.Formatted()
		reduceBondCancelled := mpv3.IsBondReduceCancelled.Get()

		// Ignore minipools that don't have a bond reduction pending
		timeSinceReductionStart := blockTime.Sub(reduceBondTime)
		if reduceBondTime == zeroTime ||
			reduceBondCancelled ||
			timeSinceReductionStart > reductionWindowEnd {
			continue
		}

		// Calculate the bond delta from the pending reduction
		oldBond := mpCommon.NodeDepositBalance.Get()
		newBond := mpv3.ReduceBondValue.Get()
		mpDelta := big.NewInt(0).Sub(oldBond, newBond)
		totalDelta.Add(totalDelta, mpDelta)
	}

	return &CollateralAmounts{
		EthMatched:         node.EthMatched.Get(),
		EthMatchedLimit:    node.EthMatchedLimit.Get(),
		PendingMatchAmount: totalDelta,
	}, nil
}
