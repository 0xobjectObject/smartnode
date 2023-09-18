package odao

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	batch "github.com/rocket-pool/batch-query"
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

type oracleDaoCancelProposalContextFactory struct {
	handler *OracleDaoHandler
}

func (f *oracleDaoCancelProposalContextFactory) Create(vars map[string]string) (*oracleDaoCancelProposalContext, error) {
	c := &oracleDaoCancelProposalContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArg("id", vars, input.ValidatePositiveUint, &c.id),
	}
	return c, errors.Join(inputErrs...)
}

func (f *oracleDaoCancelProposalContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterSingleStageRoute[*oracleDaoCancelProposalContext, api.OracleDaoCancelProposalData](
		router, "cancel-proposal", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type oracleDaoCancelProposalContext struct {
	handler     *OracleDaoHandler
	rp          *rocketpool.RocketPool
	nodeAddress common.Address

	id         uint64
	odaoMember *oracle.OracleDaoMember
	dpm        *proposals.DaoProposalManager
	prop       *proposals.OracleDaoProposal
}

func (c *oracleDaoCancelProposalContext) Initialize() error {
	sp := c.handler.serviceProvider
	c.rp = sp.GetRocketPool()
	c.nodeAddress, _ = sp.GetWallet().GetAddress()

	// Requirements
	err := sp.RequireNodeRegistered()
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

func (c *oracleDaoCancelProposalContext) GetState(mc *batch.MultiCaller) {
	c.dpm.GetProposalCount(mc)
	c.odaoMember.GetExists(mc)
	c.prop.GetState(mc)
	c.prop.GetProposerAddress(mc)
}

func (c *oracleDaoCancelProposalContext) PrepareData(data *api.OracleDaoCancelProposalData, opts *bind.TransactOpts) error {
	// Verify oDAO status
	if !c.odaoMember.Details.Exists {
		return errors.New("The node is not a member of the oracle DAO.")
	}

	// Check proposal details
	state := c.prop.Details.State.Formatted()
	data.DoesNotExist = (c.id > c.dpm.Details.ProposalCount.Formatted())
	data.InvalidState = !(state == rptypes.Pending || state == rptypes.Active)
	data.InvalidProposer = !(c.nodeAddress == c.prop.Details.ProposerAddress)
	data.CanCancel = !(data.DoesNotExist || data.InvalidState || data.InvalidProposer)

	// Get the tx
	if data.CanCancel && opts != nil {
		txInfo, err := c.prop.Cancel(opts)
		if err != nil {
			return fmt.Errorf("error getting TX info for CancelProposal: %w", err)
		}
		data.TxInfo = txInfo
	}
	return nil
}
