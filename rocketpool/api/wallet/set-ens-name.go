package wallet

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/gorilla/mux"
	"github.com/rocket-pool/rocketpool-go/core"
	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	"github.com/rocket-pool/smartnode/shared/types/api"
	ens "github.com/wealdtech/go-ens/v3"
)

// ===============
// === Factory ===
// ===============

type walletSetEnsNameContextFactory struct {
	handler *WalletHandler
}

func (f *walletSetEnsNameContextFactory) Create(vars map[string]string) (*walletSetEnsNameContext, error) {
	c := &walletSetEnsNameContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.GetStringFromVars("name", vars, &c.name),
	}
	return c, errors.Join(inputErrs...)
}

func (f *walletSetEnsNameContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterQuerylessRoute[*walletSetEnsNameContext, api.WalletSetEnsNameData](
		router, "set-ens-name", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type walletSetEnsNameContext struct {
	handler *WalletHandler
	name    string
}

func (c *walletSetEnsNameContext) PrepareData(data *api.WalletSetEnsNameData, opts *bind.TransactOpts) error {
	sp := c.handler.serviceProvider
	ec := sp.GetEthClient()
	nodeAddress, _ := sp.GetWallet().GetAddress()

	// Requirements
	err := sp.RequireNodeAddress()
	if err != nil {
		return err
	}

	// Name validation
	if c.name == "" {
		return fmt.Errorf("name cannot be blank")
	}

	// The ENS name must resolve to the wallet address
	resolvedAddress, err := ens.Resolve(ec, c.name)
	if err != nil {
		return fmt.Errorf("error resolving '%s' to an address: %w", c.name, err)
	}

	if resolvedAddress != nodeAddress {
		return fmt.Errorf("%s currently resolves to the address %s instead of the node wallet address %s", c.name, resolvedAddress.Hex(), nodeAddress.Hex())
	}

	// Check if the name is already in use
	resolvedName, err := ens.ReverseResolve(ec, nodeAddress)
	if err != nil && err.Error() != "not a resolver" {
		// Handle errors unrelated to the address not being an ENS resolver
		return fmt.Errorf("error reverse resolving %s to an ENS name: %w", nodeAddress.Hex(), err)
	} else if resolvedName == c.name {
		return fmt.Errorf("the ENS record already points to the name '%s'", c.name)
	}

	// Get the raw TX from the ENS lib
	registrar, err := ens.NewReverseRegistrar(ec)
	if err != nil {
		return fmt.Errorf("error creating reverse registrar binding: %w", err)
	}
	opts.NoSend = true
	tx, err := registrar.SetName(opts, c.name)
	if err != nil {
		return fmt.Errorf("error constructing SetName TX: %w", err)
	}

	// Derive the TXInfo
	txInfo, err := core.NewTransactionInfoRaw(ec, *tx.To(), tx.Data(), opts)
	if err != nil {
		return fmt.Errorf("error getting TX info for SetName: %w", err)
	}
	data.TxInfo = txInfo
	return nil
}
