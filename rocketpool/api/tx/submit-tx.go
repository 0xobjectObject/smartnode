package tx

import (
	"errors"
	"fmt"
	_ "time/tzdata"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/gorilla/mux"

	"github.com/rocket-pool/smartnode/rocketpool/common/server"
	"github.com/rocket-pool/smartnode/shared/types/api"
)

// ===============
// === Factory ===
// ===============

type txSubmitTxContextFactory struct {
	handler *TxHandler
}

func (f *txSubmitTxContextFactory) Create(body api.SubmitTxBody) (*txSubmitTxContext, error) {
	c := &txSubmitTxContext{
		handler: f.handler,
		body:    body,
	}
	// Validate the submission
	if body.Submission.TxInfo == nil {
		return nil, fmt.Errorf("submission TX info must be set")
	}
	if body.Submission.GasLimit == 0 {
		return nil, fmt.Errorf("submission gas limit must be set")
	}
	if body.MaxFee == nil {
		return nil, fmt.Errorf("submission max fee must be set")
	}
	if body.MaxPriorityFee == nil {
		return nil, fmt.Errorf("submission max priority fee must be set")
	}
	return c, nil
}

func (f *txSubmitTxContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterQuerylessPost[*txSubmitTxContext, api.SubmitTxBody, api.TxData](
		router, "submit-tx", f, f.handler.serviceProvider,
	)
}

// ===============
// === Context ===
// ===============

type txSubmitTxContext struct {
	handler *TxHandler
	body    api.SubmitTxBody
}

func (c *txSubmitTxContext) PrepareData(data *api.TxData, opts *bind.TransactOpts) error {
	sp := c.handler.serviceProvider
	rp := sp.GetRocketPool()

	err := errors.Join(
		sp.RequireWalletReady(),
	)
	if err != nil {
		return err
	}

	if c.body.Nonce != nil {
		opts.Nonce = c.body.Nonce
	}
	opts.GasLimit = c.body.Submission.GasLimit
	opts.GasFeeCap = c.body.MaxFee
	opts.GasTipCap = c.body.MaxPriorityFee

	tx, err := rp.ExecuteTransaction(c.body.Submission.TxInfo, opts)
	if err != nil {
		return fmt.Errorf("error submitting transaction: %w", err)
	}
	data.TxHash = tx.Hash()
	return nil
}
