package node

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/gorilla/mux"

	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	"github.com/rocket-pool/smartnode/shared/types/api"
)

// ===============
// === Factory ===
// ===============

type nodeClearSnapshotDelegateContextFactory struct {
	handler *NodeHandler
}

func (f *nodeClearSnapshotDelegateContextFactory) Create(vars map[string]string) (*nodeClearSnapshotDelegateContext, error) {
	c := &nodeClearSnapshotDelegateContext{
		handler: f.handler,
	}
	return c, nil
}

func (f *nodeClearSnapshotDelegateContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterQuerylessRoute[*nodeClearSnapshotDelegateContext, api.TxInfoData](
		router, "clear-snapshot-delegate", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type nodeClearSnapshotDelegateContext struct {
	handler *NodeHandler
}

func (c *nodeClearSnapshotDelegateContext) PrepareData(data *api.TxInfoData, opts *bind.TransactOpts) error {
	sp := c.handler.serviceProvider
	cfg := sp.GetConfig()
	snapshot := sp.GetSnapshotDelegation()
	idHash := cfg.Smartnode.GetVotingSnapshotID()

	var err error
	data.TxInfo, err = snapshot.ClearDelegate(idHash, opts)
	if err != nil {
		return fmt.Errorf("error getting TX info for ClearDelegate: %w", err)
	}
	return nil
}
