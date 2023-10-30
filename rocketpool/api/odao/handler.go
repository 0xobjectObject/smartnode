package odao

import (
	"github.com/gorilla/mux"

	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	"github.com/rocket-pool/smartnode/rocketpool/common/services"
)

type OracleDaoHandler struct {
	serviceProvider *services.ServiceProvider
	factories       []server.IContextFactory
}

func NewOracleDaoHandler(serviceProvider *services.ServiceProvider) *OracleDaoHandler {
	h := &OracleDaoHandler{
		serviceProvider: serviceProvider,
	}
	h.factories = []server.IContextFactory{
		&oracleDaoStatusContextFactory{h},
		&oracleDaoCancelProposalContextFactory{h},
		&oracleDaoExecuteProposalContextFactory{h},
		&oracleDaoSettingsContextFactory{h},
		&oracleDaoJoinContextFactory{h},
		&oracleDaoLeaveContextFactory{h},
		&oracleDaoMembersContextFactory{h},
		&oracleDaoProposalsContextFactory{h},
		&oracleDaoVoteContextFactory{h},
		&oracleDaoProposeInviteContextFactory{h},
		&oracleDaoProposeKickContextFactory{h},
		&oracleDaoProposeLeaveContextFactory{h},
		&oracleDaoProposeSettingContextFactory{h},
	}
	return h
}

func (h *OracleDaoHandler) RegisterRoutes(router *mux.Router) {
	subrouter := router.PathPrefix("/odao").Subrouter()
	for _, factory := range h.factories {
		factory.RegisterRoute(subrouter)
	}
}
