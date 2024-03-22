package network

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/gorilla/mux"
	"github.com/rocket-pool/node-manager-core/api/server"
	"github.com/rocket-pool/node-manager-core/api/types"
	"github.com/rocket-pool/node-manager-core/utils/input"
	"github.com/rocket-pool/smartnode/rocketpool-daemon/common/rewards"
)

// ===============
// === Factory ===
// ===============

type networkDownloadRewardsContextFactory struct {
	handler *NetworkHandler
}

func (f *networkDownloadRewardsContextFactory) Create(args url.Values) (*networkDownloadRewardsContext, error) {
	c := &networkDownloadRewardsContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArg("interval", args, input.ValidateUint, &c.interval),
	}
	return c, errors.Join(inputErrs...)
}

func (f *networkDownloadRewardsContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterQuerylessGet[*networkDownloadRewardsContext, types.SuccessData](
		router, "download-rewards-file", f, f.handler.serviceProvider.ServiceProvider,
	)
}

// ===============
// === Context ===
// ===============

type networkDownloadRewardsContext struct {
	handler *NetworkHandler

	interval uint64
}

func (c *networkDownloadRewardsContext) PrepareData(data *types.SuccessData, opts *bind.TransactOpts) error {
	sp := c.handler.serviceProvider
	rp := sp.GetRocketPool()
	cfg := sp.GetConfig()
	nodeAddress, _ := sp.GetWallet().GetAddress()

	// Requirements
	err := sp.RequireNodeRegistered()
	if err != nil {
		return err
	}

	// Get the event info for the interval
	intervalInfo, err := rewards.GetIntervalInfo(rp, cfg, nodeAddress, c.interval, nil)
	if err != nil {
		return fmt.Errorf("error getting interval %d info: %w", c.interval, err)
	}

	// Download the rewards file
	err = rewards.DownloadRewardsFile(cfg, &intervalInfo)
	if err != nil {
		return fmt.Errorf("error downloading interval %d rewards file: %w", c.interval, err)
	}
	return nil
}
