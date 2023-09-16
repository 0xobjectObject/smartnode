package faucet

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	batch "github.com/rocket-pool/batch-query"
	"github.com/rocket-pool/rocketpool-go/rocketpool"
	"github.com/rocket-pool/smartnode/rocketpool/common/contracts"
	"github.com/rocket-pool/smartnode/shared/types/api"
)

// ===============
// === Factory ===
// ===============

type faucetWithdrawContextFactory struct {
	handler *FaucetHandler
}

func (f *faucetWithdrawContextFactory) Create(vars map[string]string) (*faucetWithdrawContext, error) {
	c := &faucetWithdrawContext{
		handler: f.handler,
	}
	return c, nil
}

// ===============
// === Context ===
// ===============

type faucetWithdrawContext struct {
	handler     *FaucetHandler
	rp          *rocketpool.RocketPool
	f           *contracts.RplFaucet
	nodeAddress common.Address

	allowance *big.Int
}

func (c *faucetWithdrawContext) Initialize() error {
	sp := c.handler.serviceProvider
	c.rp = sp.GetRocketPool()
	c.f = sp.GetRplFaucet()
	c.nodeAddress, _ = sp.GetWallet().GetAddress()

	// Requirements
	return errors.Join(
		sp.RequireNodeRegistered(),
		sp.RequireRplFaucet(),
	)
}

func (c *faucetWithdrawContext) GetState(mc *batch.MultiCaller) {
	c.f.GetBalance(mc)
	c.f.GetAllowanceFor(mc, &c.allowance, c.nodeAddress)
	c.f.GetWithdrawalFee(mc)
}

func (c *faucetWithdrawContext) PrepareData(data *api.FaucetWithdrawRplData, opts *bind.TransactOpts) error {
	// Get node account balance
	nodeAccountBalance, err := c.rp.Client.BalanceAt(context.Background(), c.nodeAddress, nil)
	if err != nil {
		return fmt.Errorf("error getting node account balance: %w", err)
	}

	// Populate the response
	data.InsufficientFaucetBalance = (c.f.Details.Balance.Cmp(big.NewInt(0)) == 0)
	data.InsufficientAllowance = (c.allowance.Cmp(big.NewInt(0)) == 0)
	data.InsufficientNodeBalance = (nodeAccountBalance.Cmp(c.f.Details.WithdrawalFee) < 0)
	data.CanWithdraw = !(data.InsufficientFaucetBalance || data.InsufficientAllowance || data.InsufficientNodeBalance)

	if data.CanWithdraw && opts != nil {
		opts.Value = c.f.Details.WithdrawalFee

		// Get withdrawal amount
		var amount *big.Int
		balance := c.f.Details.Balance
		if balance.Cmp(c.allowance) > 0 {
			amount = c.allowance
		} else {
			amount = balance
		}

		txInfo, err := c.f.Withdraw(opts, amount)
		if err != nil {
			return fmt.Errorf("error getting TX info for Withdraw: %w", err)
		}
		data.TxInfo = txInfo
	}
	return nil
}
