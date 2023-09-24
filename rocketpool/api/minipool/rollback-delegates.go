package minipool

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"github.com/rocket-pool/rocketpool-go/core"
	"github.com/rocket-pool/rocketpool-go/minipool"
	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	"github.com/rocket-pool/smartnode/shared/types/api"
	"github.com/rocket-pool/smartnode/shared/utils/input"
)

// ===============
// === Factory ===
// ===============

type minipoolRollbackDelegatesContextFactory struct {
	handler *MinipoolHandler
}

func (f *minipoolRollbackDelegatesContextFactory) Create(vars map[string]string) (*minipoolRollbackDelegatesContext, error) {
	c := &minipoolRollbackDelegatesContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArg("addresses", vars, input.ValidateAddresses, &c.minipoolAddresses),
	}
	return c, errors.Join(inputErrs...)
}

func (f *minipoolRollbackDelegatesContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterQuerylessRoute[*minipoolRollbackDelegatesContext, api.BatchTxInfoData](
		router, "delegate/rollback", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type minipoolRollbackDelegatesContext struct {
	handler           *MinipoolHandler
	minipoolAddresses []common.Address
}

func (c *minipoolRollbackDelegatesContext) PrepareData(data *api.BatchTxInfoData, opts *bind.TransactOpts) error {
	return prepareMinipoolBatchTxData(c.handler.serviceProvider, c.minipoolAddresses, data, c.CreateTx, "rollback-delegate")
}

func (c *minipoolRollbackDelegatesContext) CreateTx(mp minipool.IMinipool, opts *bind.TransactOpts) (*core.TransactionInfo, error) {
	return mp.Common().DelegateRollback(opts)
}
