package node

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	batch "github.com/rocket-pool/batch-query"
	"github.com/rocket-pool/rocketpool-go/core"
	"github.com/rocket-pool/rocketpool-go/dao/oracle"
	"github.com/rocket-pool/rocketpool-go/dao/protocol"
	"github.com/rocket-pool/rocketpool-go/minipool"
	"github.com/rocket-pool/rocketpool-go/node"
	"github.com/rocket-pool/rocketpool-go/rocketpool"
	rptypes "github.com/rocket-pool/rocketpool-go/types"
	"github.com/rocket-pool/rocketpool-go/utils/eth"
	"github.com/rocket-pool/smartnode/rocketpool/common/beacon"
	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	rputils "github.com/rocket-pool/smartnode/rocketpool/utils/rp"
	"github.com/rocket-pool/smartnode/shared/config"
	sharedtypes "github.com/rocket-pool/smartnode/shared/types"
	"github.com/rocket-pool/smartnode/shared/types/api"
	cfgtypes "github.com/rocket-pool/smartnode/shared/types/config"
	"github.com/rocket-pool/smartnode/shared/utils/input"
)

// ===============
// === Factory ===
// ===============

type nodeCreateVacantMinipoolContextFactory struct {
	handler *NodeHandler
}

func (f *nodeCreateVacantMinipoolContextFactory) Create(vars map[string]string) (*nodeCreateVacantMinipoolContext, error) {
	c := &nodeCreateVacantMinipoolContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArg("amount-wei", vars, input.ValidateBigInt, &c.amountWei),
		server.ValidateArg("min-node-fee", vars, input.ValidateFraction, &c.minNodeFee),
		server.ValidateArg("salt", vars, input.ValidateBigInt, &c.salt),
		server.ValidateArg("pubkey", vars, input.ValidatePubkey, &c.pubkey),
	}
	return c, errors.Join(inputErrs...)
}

func (f *nodeCreateVacantMinipoolContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterQuerylessRoute[*nodeCreateVacantMinipoolContext, api.NodeCreateVacantMinipoolData](
		router, "create-vacant-minipool", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type nodeCreateVacantMinipoolContext struct {
	handler *NodeHandler
	cfg     *config.RocketPoolConfig
	rp      *rocketpool.RocketPool
	bc      beacon.Client

	amountWei  *big.Int
	minNodeFee float64
	salt       *big.Int
	pubkey     rptypes.ValidatorPubkey
	node       *node.Node
	pSettings  *protocol.ProtocolDaoSettings
	oSettings  *oracle.OracleDaoSettings
	mpMgr      *minipool.MinipoolManager
}

func (c *nodeCreateVacantMinipoolContext) Initialize() error {
	sp := c.handler.serviceProvider
	c.cfg = sp.GetConfig()
	c.rp = sp.GetRocketPool()
	c.bc = sp.GetBeaconClient()
	nodeAddress, _ := sp.GetWallet().GetAddress()

	// Requirements
	err := sp.RequireNodeRegistered()
	if err != nil {
		return err
	}

	// Bindings
	c.node, err = node.NewNode(c.rp, nodeAddress)
	if err != nil {
		return fmt.Errorf("error creating node %s binding: %w", nodeAddress.Hex(), err)
	}
	pMgr, err := protocol.NewProtocolDaoManager(c.rp)
	if err != nil {
		return fmt.Errorf("error getting pDAO manager binding: %w", err)
	}
	c.pSettings = pMgr.Settings
	oMgr, err := oracle.NewOracleDaoManager(c.rp)
	if err != nil {
		return fmt.Errorf("error getting oDAO manager binding: %w", err)
	}
	c.oSettings = oMgr.Settings
	c.mpMgr, err = minipool.NewMinipoolManager(c.rp)
	if err != nil {
		return fmt.Errorf("error getting minipool manager binding: %w", err)
	}
	return nil
}

func (c *nodeCreateVacantMinipoolContext) GetState(mc *batch.MultiCaller) {
	core.AddQueryablesToMulticall(mc,
		c.node.EthMatched,
		c.node.EthMatchedLimit,
		c.pSettings.Node.AreVacantMinipoolsEnabled,
		c.oSettings.Minipool.PromotionScrubPeriod,
	)
}

func (c *nodeCreateVacantMinipoolContext) PrepareData(data *api.NodeCreateVacantMinipoolData, opts *bind.TransactOpts) error {
	// Initial population
	data.DepositDisabled = !c.pSettings.Node.AreVacantMinipoolsEnabled.Get()
	data.ScrubPeriod = c.oSettings.Minipool.PromotionScrubPeriod.Formatted()

	// Adjust the salt
	if c.salt.Cmp(common.Big0) == 0 {
		nonce, err := c.rp.Client.NonceAt(context.Background(), c.node.Address, nil)
		if err != nil {
			return fmt.Errorf("error getting node's latest nonce: %w", err)
		}
		c.salt.SetUint64(nonce)
	}

	// Get the next minipool address
	err := c.rp.Query(func(mc *batch.MultiCaller) error {
		c.node.GetExpectedMinipoolAddress(mc, &data.MinipoolAddress, c.salt)
		return nil
	}, nil)
	if err != nil {
		return fmt.Errorf("error getting expected minipool address: %w", err)
	}

	// Get the withdrawal credentials
	err = c.rp.Query(func(mc *batch.MultiCaller) error {
		c.mpMgr.GetMinipoolWithdrawalCredentials(mc, &data.WithdrawalCredentials, data.MinipoolAddress)
		return nil
	}, nil)
	if err != nil {
		return fmt.Errorf("error getting minipool withdrawal credentials: %w", err)
	}

	// Check data
	validatorEthWei := eth.EthToWei(ValidatorEth)
	matchRequest := big.NewInt(0).Sub(validatorEthWei, c.amountWei)
	availableToMatch := big.NewInt(0).Sub(c.node.EthMatchedLimit.Get(), c.node.EthMatched.Get())
	data.InsufficientRplStake = (availableToMatch.Cmp(matchRequest) == -1)

	// Update response
	data.CanDeposit = !(data.InsufficientRplStake || data.InvalidAmount || data.DepositDisabled)
	if !data.CanDeposit {
		return nil
	}
	// Make sure ETH2 is on the correct chain
	depositContractInfo, err := rputils.GetDepositContractInfo(c.rp, c.cfg, c.bc)
	if err != nil {
		return fmt.Errorf("error verifying the EL and BC are on the same chain: %w", err)
	}
	if depositContractInfo.RPNetwork != depositContractInfo.BeaconNetwork ||
		depositContractInfo.RPDepositContract != depositContractInfo.BeaconDepositContract {
		return fmt.Errorf("FATAL: Beacon network mismatch! Expected %s on chain %d, but beacon is using %s on chain %d.",
			depositContractInfo.RPDepositContract.Hex(),
			depositContractInfo.RPNetwork,
			depositContractInfo.BeaconDepositContract.Hex(),
			depositContractInfo.BeaconNetwork)
	}

	// Check if the pubkey is for an existing active_ongoing validator
	validatorStatus, err := c.bc.GetValidatorStatus(c.pubkey, nil)
	if err != nil {
		return fmt.Errorf("error checking status of existing validator: %w", err)
	}
	if !validatorStatus.Exists {
		return fmt.Errorf("validator %s does not exist on the Beacon chain. If you recently created it, please wait until the Consensus layer has processed your deposits.", c.pubkey.Hex())
	}
	if validatorStatus.Status != sharedtypes.ValidatorState_ActiveOngoing {
		return fmt.Errorf("validator %s must be in the active_ongoing state to be migrated, but it is currently in %s.", c.pubkey.Hex(), string(validatorStatus.Status))
	}
	if c.cfg.Smartnode.Network.Value.(cfgtypes.Network) != cfgtypes.Network_Devnet && validatorStatus.WithdrawalCredentials[0] != 0x00 {
		return fmt.Errorf("validator %s already has withdrawal credentials [%s], which are not BLS credentials.", c.pubkey.Hex(), validatorStatus.WithdrawalCredentials.Hex())
	}

	// Convert the existing balance from gwei to wei
	balanceWei := big.NewInt(0).SetUint64(validatorStatus.Balance)
	balanceWei.Mul(balanceWei, big.NewInt(1e9))

	// Get tx info
	txInfo, err := c.node.CreateVacantMinipool(c.amountWei, c.minNodeFee, c.pubkey, c.salt, data.MinipoolAddress, balanceWei, opts)
	if err != nil {
		return fmt.Errorf("error getting TX info for CreateVacantMinipool: %w", err)
	}
	data.TxInfo = txInfo
	return nil
}
