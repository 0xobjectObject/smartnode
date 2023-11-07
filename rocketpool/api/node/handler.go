package node

import (
	"github.com/gorilla/mux"

	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	"github.com/rocket-pool/smartnode/rocketpool/common/services"
)

type NodeHandler struct {
	serviceProvider *services.ServiceProvider
	factories       []server.IContextFactory
}

func NewNodeHandler(serviceProvider *services.ServiceProvider) *NodeHandler {
	h := &NodeHandler{
		serviceProvider: serviceProvider,
	}
	h.factories = []server.IContextFactory{
		&nodeBalanceContextFactory{h},
		&nodeBurnContextFactory{h},
		&nodeCheckCollateralContextFactory{h},
		&nodeClaimAndStakeContextFactory{h},
		&nodeClearSnapshotDelegateContextFactory{h},
		&nodeConfirmPrimaryWithdrawalAddressContextFactory{h},
		&nodeConfirmRplWithdrawalAddressContextFactory{h},
		&nodeCreateVacantMinipoolContextFactory{h},
		&nodeDepositContextFactory{h},
		&nodeDistributeContextFactory{h},
		&nodeRewardsContextFactory{h},
		&nodeGetRewardsInfoContextFactory{h},
		&nodeGetSnapshotProposalsContextFactory{h},
		&nodeGetSnapshotVotingPowerContextFactory{h},
		&nodeInitializeFeeDistributorContextFactory{h},
		&nodeRegisterContextFactory{h},
		&nodeResolveEnsContextFactory{h},
		&nodeSendMessageContextFactory{h},
		&nodeSendContextFactory{h},
		&nodeSetPrimaryWithdrawalAddressContextFactory{h},
		&nodeSetRplWithdrawalAddressContextFactory{h},
		&nodeSetSnapshotDelegateContextFactory{h},
		&nodeSetSmoothingPoolRegistrationStatusContextFactory{h},
		&nodeSetStakeRplForAllowedContextFactory{h},
		&nodeSetTimezoneContextFactory{h},
		&nodeStakeRplContextFactory{h},
		&nodeStatusContextFactory{h},
		&nodeSwapRplContextFactory{h},
		&nodeWithdrawRplContextFactory{h},
	}
	return h
}

func (h *NodeHandler) RegisterRoutes(router *mux.Router) {
	subrouter := router.PathPrefix("/node").Subrouter()
	for _, factory := range h.factories {
		factory.RegisterRoute(subrouter)
	}
}
