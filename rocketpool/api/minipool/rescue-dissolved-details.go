package minipool

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	batch "github.com/rocket-pool/batch-query"
	"github.com/rocket-pool/rocketpool-go/minipool"
	"github.com/rocket-pool/rocketpool-go/node"
	"github.com/rocket-pool/rocketpool-go/types"
	"github.com/rocket-pool/rocketpool-go/utils/eth"

	"github.com/rocket-pool/smartnode/rocketpool/common/beacon"
	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	sharedtypes "github.com/rocket-pool/smartnode/shared/types"
	"github.com/rocket-pool/smartnode/shared/types/api"
)

// ===============
// === Factory ===
// ===============

type minipoolRescueDissolvedDetailsContextFactory struct {
	handler *MinipoolHandler
}

func (f *minipoolRescueDissolvedDetailsContextFactory) Create(vars map[string]string) (*minipoolRescueDissolvedDetailsContext, error) {
	c := &minipoolRescueDissolvedDetailsContext{
		handler: f.handler,
	}
	return c, nil
}

func (f *minipoolRescueDissolvedDetailsContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterMinipoolRoute[*minipoolRescueDissolvedDetailsContext, api.MinipoolRescueDissolvedDetailsData](
		router, "rescue-dissolved/details", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type minipoolRescueDissolvedDetailsContext struct {
	handler *MinipoolHandler
	bc      beacon.Client
}

func (c *minipoolRescueDissolvedDetailsContext) Initialize() error {
	sp := c.handler.serviceProvider
	c.bc = sp.GetBeaconClient()

	// Requirements
	err := errors.Join(
		sp.RequireNodeRegistered(),
		sp.RequireBeaconClientSynced(),
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *minipoolRescueDissolvedDetailsContext) GetState(node *node.Node, mc *batch.MultiCaller) {
}

func (c *minipoolRescueDissolvedDetailsContext) CheckState(node *node.Node, response *api.MinipoolRescueDissolvedDetailsData) bool {
	return true
}

func (c *minipoolRescueDissolvedDetailsContext) GetMinipoolDetails(mc *batch.MultiCaller, mp minipool.IMinipool, index int) {
	mp.GetFinalised(mc)
	mp.GetStatus(mc)
	mp.GetPubkey(mc)
}

func (c *minipoolRescueDissolvedDetailsContext) PrepareData(addresses []common.Address, mps []minipool.IMinipool, data *api.MinipoolRescueDissolvedDetailsData) error {
	// Get the rescue details
	pubkeys := []types.ValidatorPubkey{}
	detailsMap := map[types.ValidatorPubkey]int{}
	details := make([]api.MinipoolRescueDissolvedDetails, len(addresses))
	for i, mp := range mps {
		mpCommon := mp.GetCommonDetails()
		mpDetails := api.MinipoolRescueDissolvedDetails{
			Address:       mpCommon.Address,
			MinipoolState: mpCommon.Status.Formatted(),
			IsFinalized:   mpCommon.IsFinalised,
		}

		if mpDetails.MinipoolState != types.MinipoolStatus_Dissolved || mpDetails.IsFinalized {
			mpDetails.InvalidElState = true
		} else {
			pubkeys = append(pubkeys, mpCommon.Pubkey)
			detailsMap[mpCommon.Pubkey] = i
		}

		details[i] = mpDetails
	}

	// Get the statuses on Beacon
	beaconStatuses, err := c.bc.GetValidatorStatuses(pubkeys, nil)
	if err != nil {
		return fmt.Errorf("error getting validator statuses on Beacon: %w", err)
	}

	// Do a complete viability check
	for pubkey, beaconStatus := range beaconStatuses {
		i := detailsMap[pubkey]
		mpDetails := &details[i]
		mpDetails.BeaconState = beaconStatus.Status
		mpDetails.InvalidBeaconState = beaconStatus.Status != sharedtypes.ValidatorState_PendingInitialized

		if !mpDetails.InvalidBeaconState {
			beaconBalanceGwei := big.NewInt(0).SetUint64(beaconStatus.Balance)
			mpDetails.BeaconBalance = big.NewInt(0).Mul(beaconBalanceGwei, big.NewInt(1e9))

			// Make sure it doesn't already have 32 ETH in it
			requiredBalance := eth.EthToWei(32)
			if mpDetails.BeaconBalance.Cmp(requiredBalance) >= 0 {
				mpDetails.HasFullBalance = true
			}
		}

		mpDetails.CanRescue = !(mpDetails.IsFinalized || mpDetails.InvalidElState || mpDetails.InvalidBeaconState || mpDetails.HasFullBalance)
	}

	data.Details = details
	return nil
}
