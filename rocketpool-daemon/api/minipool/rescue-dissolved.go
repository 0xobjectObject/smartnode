package minipool

import (
	"errors"
	"fmt"
	"math/big"
	"net/url"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"github.com/rocket-pool/node-manager-core/eth"
	"github.com/rocket-pool/rocketpool-go/beacon"
	"github.com/rocket-pool/rocketpool-go/minipool"
	"github.com/rocket-pool/rocketpool-go/rocketpool"
	"github.com/rocket-pool/rocketpool-go/types"
	rpbeacon "github.com/rocket-pool/smartnode/rocketpool-daemon/common/beacon"
	"github.com/rocket-pool/smartnode/rocketpool-daemon/common/server"
	"github.com/rocket-pool/smartnode/rocketpool-daemon/common/validator"
	"github.com/rocket-pool/smartnode/rocketpool-daemon/common/wallet"
	"github.com/rocket-pool/smartnode/shared/types/api"
	"github.com/rocket-pool/smartnode/shared/utils/input"
)

// ===============
// === Factory ===
// ===============

type minipoolRescueDissolvedContextFactory struct {
	handler *MinipoolHandler
}

func (f *minipoolRescueDissolvedContextFactory) Create(args url.Values) (*minipoolRescueDissolvedContext, error) {
	c := &minipoolRescueDissolvedContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArgBatch("addresses", args, minipoolAddressBatchSize, input.ValidateAddress, &c.minipoolAddresses),
		server.ValidateArgBatch("deposit-amounts", args, minipoolAddressBatchSize, input.ValidateBigInt, &c.depositAmounts),
	}
	return c, errors.Join(inputErrs...)
}

func (f *minipoolRescueDissolvedContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterQuerylessGet[*minipoolRescueDissolvedContext, api.BatchTxInfoData](
		router, "rescue-dissolved", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type minipoolRescueDissolvedContext struct {
	handler           *MinipoolHandler
	minipoolAddresses []common.Address
	depositAmounts    []*big.Int
	rp                *rocketpool.RocketPool
	w                 *wallet.LocalWallet
	bc                rpbeacon.Client

	mpMgr *minipool.MinipoolManager
}

func (c *minipoolRescueDissolvedContext) PrepareData(data *api.BatchTxInfoData, opts *bind.TransactOpts) error {
	// Sanity check
	if len(c.minipoolAddresses) != len(c.depositAmounts) {
		return fmt.Errorf("addresses and deposit amounts must have the same length (%d vs. %d)", len(c.minipoolAddresses), len(c.depositAmounts))
	}

	sp := c.handler.serviceProvider
	c.rp = sp.GetRocketPool()
	c.w = sp.GetWallet()
	c.bc = sp.GetBeaconClient()

	// Requirements
	err := errors.Join(
		sp.RequireNodeRegistered(),
		sp.RequireBeaconClientSynced(),
	)
	if err != nil {
		return err
	}

	// Bindings
	c.mpMgr, err = minipool.NewMinipoolManager(c.rp)
	if err != nil {
		return fmt.Errorf("error creating minipool manager binding: %w", err)
	}

	// Get the TXs
	txInfos := make([]*eth.TransactionInfo, len(c.minipoolAddresses))
	for i, address := range c.minipoolAddresses {
		amount := c.depositAmounts[i]
		opts.Value = amount
		txInfo, err := c.getDepositTx(address, amount, opts)
		if err != nil {
			return fmt.Errorf("error simulating deposit transaction for minipool %s: %w", address.Hex(), err)
		}
		txInfos[i] = txInfo
	}

	data.TxInfos = txInfos
	return nil
}

// Create a transaction for submitting a rescue deposit, optionally simulating it only for gas estimation
func (c *minipoolRescueDissolvedContext) getDepositTx(minipoolAddress common.Address, amount *big.Int, opts *bind.TransactOpts) (*eth.TransactionInfo, error) {
	beaconDeposit, err := beacon.NewBeaconDeposit(c.rp)
	if err != nil {
		return nil, fmt.Errorf("error creating Beacon deposit contract binding: %w", err)
	}

	// Create minipool
	mp, err := c.mpMgr.CreateMinipoolFromAddress(minipoolAddress, false, nil)
	if err != nil {
		return nil, err
	}
	mpCommon := mp.Common()

	// Get eth2 config
	eth2Config, err := c.bc.GetEth2Config()
	if err != nil {
		return nil, err
	}

	// Get the contract state
	err = c.rp.Query(nil, nil, mpCommon.WithdrawalCredentials, mpCommon.Pubkey)
	if err != nil {
		return nil, fmt.Errorf("error getting contract state: %w", err)
	}

	// Get minipool withdrawal credentials and keys
	withdrawalCredentials := mpCommon.WithdrawalCredentials.Get()
	validatorPubkey := mpCommon.Pubkey.Get()
	validatorKey, err := c.w.GetValidatorKeyByPubkey(validatorPubkey)
	if err != nil {
		return nil, fmt.Errorf("error getting validator private key for pubkey %s: %w", validatorPubkey.Hex(), err)
	}

	// Get validator deposit data
	amountGwei := big.NewInt(0).Div(amount, big.NewInt(1e9)).Uint64()
	depositData, depositDataRoot, err := validator.GetDepositData(validatorKey, withdrawalCredentials, eth2Config, amountGwei)
	if err != nil {
		return nil, err
	}
	signature := types.BytesToValidatorSignature(depositData.Signature)

	// Get the tx info
	txInfo, err := beaconDeposit.Deposit(opts, validatorPubkey, withdrawalCredentials, signature, depositDataRoot)
	if err != nil {
		return nil, fmt.Errorf("error performing rescue deposit: %s", err.Error())
	}
	return txInfo, nil
}