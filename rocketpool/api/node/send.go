package node

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	batch "github.com/rocket-pool/batch-query"
	"github.com/rocket-pool/rocketpool-go/core"
	"github.com/rocket-pool/rocketpool-go/tokens"

	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	"github.com/rocket-pool/smartnode/shared/types/api"
	"github.com/rocket-pool/smartnode/shared/utils/input"
)

// ===============
// === Factory ===
// ===============

type nodeSendContextFactory struct {
	handler *NodeHandler
}

func (f *nodeSendContextFactory) Create(vars map[string]string) (*nodeSendContext, error) {
	c := &nodeSendContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArg("amount", vars, input.ValidateBigInt, &c.amount),
		server.GetStringFromVars("token", vars, &c.token),
		server.ValidateArg("recipient", vars, input.ValidateAddress, &c.recipient),
	}
	return c, errors.Join(inputErrs...)
}

func (f *nodeSendContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterQuerylessRoute[*nodeSendContext, api.NodeSendData](
		router, "send", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type nodeSendContext struct {
	handler *NodeHandler

	amount    *big.Int
	token     string
	recipient common.Address
}

func (c *nodeSendContext) PrepareData(data *api.NodeSendData, opts *bind.TransactOpts) error {
	sp := c.handler.serviceProvider
	rp := sp.GetRocketPool()
	ec := sp.GetEthClient()
	nodeAddress, _ := sp.GetWallet().GetAddress()

	// Requirements
	err := sp.RequireNodeAddress()
	if err != nil {
		return err
	}

	// Get the contract (nil in the case of ETH)
	var tokenContract tokens.IErc20Token
	if c.token == "eth" {
		tokenContract = nil
	} else if strings.HasPrefix(c.token, "0x") {
		// Arbitrary token - make sure the contract address is legal
		if !common.IsHexAddress(c.token) {
			return fmt.Errorf("[%s] is not a valid token address", c.token)
		}
		tokenAddress := common.HexToAddress(c.token)

		// Make a binding for it
		tokenContract, err := tokens.NewErc20Contract(rp, tokenAddress, ec, nil)
		if err != nil {
			return fmt.Errorf("error creating ERC20 contract binding: %w", err)
		}
		data.TokenSymbol = tokenContract.Details.Symbol
		data.TokenName = tokenContract.Details.Name
	} else {
		var err error
		switch c.token {
		case "rpl":
			tokenContract, err = tokens.NewTokenRpl(rp)
		case "fsrpl":
			tokenContract, err = tokens.NewTokenRplFixedSupply(rp)
		case "reth":
			tokenContract, err = tokens.NewTokenReth(rp)
		default:
			return fmt.Errorf("[%s] is not a valid token name", c.token)
		}
		if err != nil {
			return fmt.Errorf("error creating %s token binding: %w", c.token, err)
		}
	}

	// Get the balance
	if tokenContract != nil {
		err := rp.Query(func(mc *batch.MultiCaller) error {
			tokenContract.BalanceOf(mc, &data.Balance, nodeAddress)
			return nil
		}, nil)
		if err != nil {
			return fmt.Errorf("error getting token balance: %w", err)
		}
	} else {
		// ETH balance
		var err error
		data.Balance, err = ec.BalanceAt(context.Background(), nodeAddress, nil)
		if err != nil {
			return fmt.Errorf("error getting ETH balance: %w", err)
		}
	}

	// Check the balance
	data.InsufficientBalance = (data.Balance.Cmp(common.Big0) == 0)
	data.CanSend = !(data.InsufficientBalance)

	// Get the TX Info
	if data.CanSend {
		var txInfo *core.TransactionInfo
		var err error
		if tokenContract != nil {
			txInfo, err = tokenContract.Transfer(c.recipient, c.amount, opts)
		} else {
			// ETH transfers
			newOpts := &bind.TransactOpts{
				From:  opts.From,
				Nonce: opts.Nonce,
				Value: c.amount,
			}
			txInfo, err = core.NewTransactionInfoRaw(ec, c.recipient, nil, newOpts)
		}
		if err != nil {
			return fmt.Errorf("error getting TX info for Transfer: %w", err)
		}
		data.TxInfo = txInfo
	}

	return nil
}
