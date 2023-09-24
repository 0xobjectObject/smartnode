package minipool

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	batch "github.com/rocket-pool/batch-query"
	"github.com/rocket-pool/rocketpool-go/core"
	"github.com/rocket-pool/rocketpool-go/minipool"
	"github.com/rocket-pool/rocketpool-go/rocketpool"
	"github.com/rocket-pool/rocketpool-go/types"
	eth2types "github.com/wealdtech/go-eth2-types/v2"

	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	"github.com/rocket-pool/smartnode/rocketpool/common/wallet"
	"github.com/rocket-pool/smartnode/shared/types/api"
	"github.com/rocket-pool/smartnode/shared/utils/input"
	"github.com/rocket-pool/smartnode/shared/utils/validator"
)

// ===============
// === Factory ===
// ===============

type minipoolImportKeyContextFactory struct {
	handler *MinipoolHandler
}

func (f *minipoolImportKeyContextFactory) Create(vars map[string]string) (*minipoolImportKeyContext, error) {
	c := &minipoolImportKeyContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArg("address", vars, input.ValidateAddress, &c.minipoolAddress),
		server.ValidateArg("mnemonic", vars, input.ValidateWalletMnemonic, &c.mnemonic),
	}
	return c, errors.Join(inputErrs...)
}

func (f *minipoolImportKeyContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterSingleStageRoute[*minipoolImportKeyContext, api.SuccessData](
		router, "import-key", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type minipoolImportKeyContext struct {
	handler     *MinipoolHandler
	rp          *rocketpool.RocketPool
	w           *wallet.LocalWallet
	nodeAddress common.Address
	mp          minipool.IMinipool

	minipoolAddress common.Address
	mnemonic        string
}

func (c *minipoolImportKeyContext) Initialize() error {
	sp := c.handler.serviceProvider
	c.rp = sp.GetRocketPool()
	c.w = sp.GetWallet()
	c.nodeAddress, _ = c.w.GetAddress()

	// Requirements
	err := errors.Join(
		sp.RequireNodeRegistered(),
		sp.RequireWalletReady(),
	)
	if err != nil {
		return err
	}

	// Bindings
	mpMgr, err := minipool.NewMinipoolManager(c.rp)
	if err != nil {
		return fmt.Errorf("error creating minipool manager binding: %w", err)
	}
	c.mp, err = mpMgr.CreateMinipoolFromAddress(c.minipoolAddress, false, nil)
	if err != nil {
		return fmt.Errorf("error creating minipool binding: %w", err)
	}
	return nil
}

func (c *minipoolImportKeyContext) GetState(mc *batch.MultiCaller) {
	mpCommon := c.mp.Common()
	core.AddQueryablesToMulticall(mc,
		mpCommon.NodeAddress,
		mpCommon.Pubkey,
	)
}

func (c *minipoolImportKeyContext) PrepareData(data *api.SuccessData, opts *bind.TransactOpts) error {
	// Validate minipool owner
	mpCommon := c.mp.Common()
	if mpCommon.NodeAddress.Get() != c.nodeAddress {
		return fmt.Errorf("minipool %s does not belong to the node", c.minipoolAddress.Hex())
	}

	// Get minipool validator pubkey
	pubkey := mpCommon.Pubkey.Get()
	emptyPubkey := types.ValidatorPubkey{}
	if pubkey == emptyPubkey {
		return fmt.Errorf("minipool %s does not have a validator pubkey associated with it", c.minipoolAddress.Hex())
	}

	// Get the index for this validator based on the mnemonic
	index := uint(0)
	validatorKeyPath := validator.ValidatorKeyPath
	var validatorKey *eth2types.BLSPrivateKey
	for index < validatorKeyRetrievalLimit {
		key, err := validator.GetPrivateKey(c.mnemonic, index, validatorKeyPath)
		if err != nil {
			return fmt.Errorf("error deriving key for index %d: %w", index, err)
		}
		candidatePubkey := key.PublicKey().Marshal()
		if bytes.Equal(pubkey[:], candidatePubkey) {
			validatorKey = key
			break
		}
		index++
	}
	if validatorKey == nil {
		return fmt.Errorf("couldn't find the validator key for this mnemonic after %d tries", validatorKeyRetrievalLimit)
	}

	// Save the keystore to disk
	derivationPath := fmt.Sprintf(validatorKeyPath, index)
	err := c.w.StoreValidatorKey(validatorKey, derivationPath)
	if err != nil {
		return fmt.Errorf("error saving keystore: %w", err)
	}

	data.Success = true
	return nil
}
