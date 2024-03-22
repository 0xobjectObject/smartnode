package minipool

import (
	"errors"
	"net/url"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"github.com/rocket-pool/node-manager-core/api/server"
	"github.com/rocket-pool/node-manager-core/api/types"
	"github.com/rocket-pool/node-manager-core/eth"
	"github.com/rocket-pool/node-manager-core/utils/input"
	"github.com/rocket-pool/rocketpool-go/minipool"
)

// ===============
// === Factory ===
// ===============

type minipoolUpgradeDelegatesContextFactory struct {
	handler *MinipoolHandler
}

func (f *minipoolUpgradeDelegatesContextFactory) Create(args url.Values) (*minipoolUpgradeDelegatesContext, error) {
	c := &minipoolUpgradeDelegatesContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArgBatch("addresses", args, minipoolAddressBatchSize, input.ValidateAddress, &c.minipoolAddresses),
	}
	return c, errors.Join(inputErrs...)
}

func (f *minipoolUpgradeDelegatesContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterQuerylessGet[*minipoolUpgradeDelegatesContext, types.BatchTxInfoData](
		router, "delegate/upgrade", f, f.handler.serviceProvider.ServiceProvider,
	)
}

// ===============
// === Context ===
// ===============

type minipoolUpgradeDelegatesContext struct {
	handler           *MinipoolHandler
	minipoolAddresses []common.Address
}

func (c *minipoolUpgradeDelegatesContext) PrepareData(data *types.BatchTxInfoData, opts *bind.TransactOpts) error {
	return prepareMinipoolBatchTxData(c.handler.serviceProvider, c.minipoolAddresses, data, c.CreateTx, "upgrade-delegate")
}

func (c *minipoolUpgradeDelegatesContext) CreateTx(mp minipool.IMinipool, opts *bind.TransactOpts) (*eth.TransactionInfo, error) {
	return mp.Common().DelegateUpgrade(opts)
}
