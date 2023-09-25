package odao

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	batch "github.com/rocket-pool/batch-query"
	"github.com/rocket-pool/rocketpool-go/core"
	"github.com/rocket-pool/rocketpool-go/dao/oracle"
	"github.com/rocket-pool/rocketpool-go/dao/proposals"
	"github.com/rocket-pool/rocketpool-go/rocketpool"
	rptypes "github.com/rocket-pool/rocketpool-go/types"

	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	"github.com/rocket-pool/smartnode/shared/types/api"
	"github.com/rocket-pool/smartnode/shared/utils/input"
)

// ===============
// === Factory ===
// ===============

type oracleDaoVoteContextFactory struct {
	handler *OracleDaoHandler
}

func (f *oracleDaoVoteContextFactory) Create(vars map[string]string) (*oracleDaoVoteContext, error) {
	c := &oracleDaoVoteContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArg("id", vars, input.ValidateUint, &c.id),
		server.ValidateArg("support", vars, input.ValidateBool, &c.support),
	}
	return c, errors.Join(inputErrs...)
}

func (f *oracleDaoVoteContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterSingleStageRoute[*oracleDaoVoteContext, api.OracleDaoVoteData](
		router, "vote", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type oracleDaoVoteContext struct {
	handler     *OracleDaoHandler
	rp          *rocketpool.RocketPool
	nodeAddress common.Address

	id         uint64
	support    bool
	odaoMember *oracle.OracleDaoMember
	dpm        *proposals.DaoProposalManager
	prop       *proposals.OracleDaoProposal
	hasVoted   bool
}

func (c *oracleDaoVoteContext) Initialize() error {
	sp := c.handler.serviceProvider
	c.rp = sp.GetRocketPool()
	c.nodeAddress, _ = sp.GetWallet().GetAddress()

	// Requirements
	err := sp.RequireOnOracleDao()
	if err != nil {
		return err
	}

	// Bindings
	c.odaoMember, err = oracle.NewOracleDaoMember(c.rp, c.nodeAddress)
	if err != nil {
		return fmt.Errorf("error creating oracle DAO member binding: %w", err)
	}
	c.dpm, err = proposals.NewDaoProposalManager(c.rp)
	if err != nil {
		return fmt.Errorf("error creating proposal manager binding: %w", err)
	}
	prop, err := c.dpm.CreateProposalFromID(c.id, nil)
	if err != nil {
		return fmt.Errorf("error creating proposal binding: %w", err)
	}
	var success bool
	c.prop, success = proposals.GetProposalAsOracle(prop)
	if !success {
		return fmt.Errorf("proposal %d is not an Oracle DAO proposal", c.id)
	}
	return nil
}

func (c *oracleDaoVoteContext) GetState(mc *batch.MultiCaller) {
	core.AddQueryablesToMulticall(mc,
		c.dpm.ProposalCount,
		c.prop.State,
		c.odaoMember.JoinedTime,
		c.prop.CreatedTime,
	)
	c.prop.GetMemberHasVoted(mc, &c.hasVoted, c.nodeAddress)
}

func (c *oracleDaoVoteContext) PrepareData(data *api.OracleDaoVoteData, opts *bind.TransactOpts) error {
	data.DoesNotExist = (c.prop.ID > c.dpm.ProposalCount.Formatted())
	data.InvalidState = (c.prop.State.Formatted() != rptypes.ProposalState_Active)
	data.AlreadyVoted = c.hasVoted
	data.JoinedAfterCreated = (c.odaoMember.JoinedTime.Formatted().Sub(c.prop.CreatedTime.Formatted()) >= 0)
	data.CanVote = !(data.DoesNotExist || data.InvalidState || data.JoinedAfterCreated || data.AlreadyVoted)

	// Get the tx
	if data.CanVote && opts != nil {
		txInfo, err := c.prop.VoteOn(c.support, opts)
		if err != nil {
			return fmt.Errorf("error getting TX info for VoteOn: %w", err)
		}
		data.TxInfo = txInfo
	}
	return nil
}
