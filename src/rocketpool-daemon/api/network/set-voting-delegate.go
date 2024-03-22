package network

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"

	"github.com/rocket-pool/node-manager-core/api/server"
	"github.com/rocket-pool/node-manager-core/api/types"
	"github.com/rocket-pool/node-manager-core/utils/input"
	"github.com/rocket-pool/rocketpool-go/node"
)

// ===============
// === Factory ===
// ===============

type networkSetVotingDelegateContextFactory struct {
	handler *NetworkHandler
}

func (f *networkSetVotingDelegateContextFactory) Create(args url.Values) (*nodeSetSnapshotDelegateContext, error) {
	c := &nodeSetSnapshotDelegateContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArg("delegate", args, input.ValidateAddress, &c.delegate),
	}
	return c, errors.Join(inputErrs...)
}

func (f *networkSetVotingDelegateContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterQuerylessGet[*nodeSetSnapshotDelegateContext, types.TxInfoData](
		router, "voting-delegate/set", f, f.handler.serviceProvider.ServiceProvider,
	)
}

// ===============
// === Context ===
// ===============

type nodeSetSnapshotDelegateContext struct {
	handler *NetworkHandler

	delegate common.Address
}

func (c *nodeSetSnapshotDelegateContext) PrepareData(data *types.TxInfoData, opts *bind.TransactOpts) error {
	sp := c.handler.serviceProvider
	rp := sp.GetRocketPool()
	nodeAddress, _ := sp.GetWallet().GetAddress()

	// Requirements
	err := sp.RequireNodeRegistered()
	if err != nil {
		return nil
	}

	// Binding
	node, err := node.NewNode(rp, nodeAddress)
	if err != nil {
		return fmt.Errorf("error creating node %s binding: %w", nodeAddress.Hex(), err)
	}

	// Get TX info
	data.TxInfo, err = node.SetVotingDelegate(c.delegate, opts)
	if err != nil {
		return fmt.Errorf("error getting TX info for SetVotingDelegate: %w", err)
	}
	return nil
}
