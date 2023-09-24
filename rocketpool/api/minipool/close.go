package minipool

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	batch "github.com/rocket-pool/batch-query"
	"github.com/rocket-pool/rocketpool-go/core"
	"github.com/rocket-pool/rocketpool-go/minipool"
	"github.com/rocket-pool/rocketpool-go/node"
	"github.com/rocket-pool/rocketpool-go/types"

	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	"github.com/rocket-pool/smartnode/shared/types/api"
	"github.com/rocket-pool/smartnode/shared/utils/input"
)

// ===============
// === Factory ===
// ===============

type minipoolCloseContextFactory struct {
	handler *MinipoolHandler
}

func (f *minipoolCloseContextFactory) Create(vars map[string]string) (*minipoolCloseContext, error) {
	c := &minipoolCloseContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArg("addresses", vars, input.ValidateAddresses, &c.minipoolAddresses),
	}
	return c, errors.Join(inputErrs...)
}

func (f *minipoolCloseContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterMinipoolRoute[*minipoolCloseContext, api.BatchTxInfoData](
		router, "close", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type minipoolCloseContext struct {
	handler *MinipoolHandler

	minipoolAddresses []common.Address
}

func (c *minipoolCloseContext) Initialize() error {
	sp := c.handler.serviceProvider

	// Requirements
	return errors.Join(
		sp.RequireNodeRegistered(),
		sp.RequireWalletReady(),
	)
}

func (c *minipoolCloseContext) GetState(node *node.Node, mc *batch.MultiCaller) {
}

func (c *minipoolCloseContext) CheckState(node *node.Node, response *api.BatchTxInfoData) bool {
	return true
}

func (c *minipoolCloseContext) GetMinipoolDetails(mc *batch.MultiCaller, mp minipool.IMinipool, index int) {
	mp.Common().Status.AddToQuery(mc)
	mpv3, success := minipool.GetMinipoolAsV3(mp)
	if success {
		mpv3.HasUserDistributed.AddToQuery(mc)
	}
}

func (c *minipoolCloseContext) PrepareData(addresses []common.Address, mps []minipool.IMinipool, data *api.BatchTxInfoData) error {
	return prepareMinipoolBatchTxData(c.handler.serviceProvider, addresses, data, c.CreateTx, "close")
}

func (c *minipoolCloseContext) CreateTx(mp minipool.IMinipool, opts *bind.TransactOpts) (*core.TransactionInfo, error) {
	mpCommon := mp.Common()
	minipoolAddress := mpCommon.Address
	mpv3, isMpv3 := minipool.GetMinipoolAsV3(mp)

	// If it's dissolved, just close it
	if mpCommon.Status.Formatted() == types.MinipoolStatus_Dissolved {
		// Get gas estimate
		txInfo, err := mpCommon.Close(opts)
		if err != nil {
			return nil, fmt.Errorf("error simulating close for minipool %s: %w", minipoolAddress.Hex(), err)
		}
		return txInfo, nil
	}

	// Check if it's an upgraded Atlas-era minipool
	if isMpv3 {
		if mpv3.HasUserDistributed.Get() {
			// It's already been distributed so just finalize it
			txInfo, err := mpv3.Finalise(opts)
			if err != nil {
				return nil, fmt.Errorf("error simulating finalise for minipool %s: %w", minipoolAddress.Hex(), err)
			}
			return txInfo, nil
		}

		// Do a distribution, which will finalize it
		txInfo, err := mpv3.DistributeBalance(opts, false)
		if err != nil {
			return nil, fmt.Errorf("error simulation distribute balance for minipool %s: %w", minipoolAddress.Hex(), err)
		}
		return txInfo, nil
	}

	// Handle old minipools
	return nil, fmt.Errorf("cannot create v3 binding for minipool %s, version %d", minipoolAddress.Hex(), mpCommon.Version)
}
