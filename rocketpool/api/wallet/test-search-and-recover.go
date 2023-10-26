package wallet

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	"github.com/rocket-pool/smartnode/rocketpool/common/wallet"
	"github.com/rocket-pool/smartnode/shared/types/api"
	"github.com/rocket-pool/smartnode/shared/utils/input"
)

// ===============
// === Factory ===
// ===============

type walletTestSearchAndRecoverContextFactory struct {
	handler *WalletHandler
}

func (f *walletTestSearchAndRecoverContextFactory) Create(vars map[string]string) (*walletTestSearchAndRecoverContext, error) {
	c := &walletTestSearchAndRecoverContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArg("mnemonic", vars, input.ValidateWalletMnemonic, &c.mnemonic),
		server.ValidateArg("address", vars, input.ValidateAddress, &c.address),
		server.ValidateOptionalArg("skip-validator-key-recovery", vars, input.ValidateBool, &c.skipValidatorKeyRecovery, nil),
	}
	return c, errors.Join(inputErrs...)
}

func (f *walletTestSearchAndRecoverContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterQuerylessGet[*walletTestSearchAndRecoverContext, api.WalletSearchAndRecoverData](
		router, "test-search-and-recover", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type walletTestSearchAndRecoverContext struct {
	handler                  *WalletHandler
	skipValidatorKeyRecovery bool
	mnemonic                 string
	address                  common.Address
}

func (c *walletTestSearchAndRecoverContext) PrepareData(data *api.WalletSearchAndRecoverData, opts *bind.TransactOpts) error {
	sp := c.handler.serviceProvider
	cfg := sp.GetConfig()
	rp := sp.GetRocketPool()

	if !c.skipValidatorKeyRecovery {
		err := sp.RequireEthClientSynced()
		if err != nil {
			return err
		}
	}

	// Try each derivation path across all of the iterations
	var recoveredWallet *wallet.LocalWallet
	paths := []string{
		wallet.DefaultNodeKeyPath,
		wallet.LedgerLiveNodeKeyPath,
		wallet.MyEtherWalletNodeKeyPath,
	}
	for i := uint(0); i < findIterations; i++ {
		for j := 0; j < len(paths); j++ {
			var err error
			derivationPath := paths[j]
			recoveredWallet, err = wallet.TestRecovery(derivationPath, i, c.mnemonic, cfg.Smartnode.GetChainID())
			if err != nil {
				return fmt.Errorf("error recovering wallet with path [%s], index [%d]: %w", derivationPath, i, err)
			}

			// Get recovered account
			recoveredAddress, _ := recoveredWallet.GetAddress()
			if recoveredAddress == c.address {
				// We found the correct derivation path and index
				data.FoundWallet = true
				data.DerivationPath = derivationPath
				data.Index = i
				break
			}
		}
		if data.FoundWallet {
			break
		}
	}

	if !data.FoundWallet {
		return fmt.Errorf("exhausted all derivation paths and indices from 0 to %d, wallet not found", findIterations)
	}
	data.AccountAddress, _ = recoveredWallet.GetAddress()

	// Recover validator keys
	if !c.skipValidatorKeyRecovery {
		var err error
		data.ValidatorKeys, err = wallet.RecoverMinipoolKeys(cfg, rp, recoveredWallet, true)
		if err != nil {
			return fmt.Errorf("error recovering minipool keys: %w", err)
		}
	}

	return nil
}
