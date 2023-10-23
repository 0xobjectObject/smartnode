package contracts

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	batch "github.com/rocket-pool/batch-query"
	"github.com/rocket-pool/rocketpool-go/core"
)

const (
	snapshotDelegationAbiString string = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegate\",\"type\":\"address\"}],\"name\":\"ClearDelegate\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegate\",\"type\":\"address\"}],\"name\":\"SetDelegate\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"clearDelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"delegation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"delegate\",\"type\":\"address\"}],\"name\":\"setDelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"
)

// ===============
// === Structs ===
// ===============

// Binding for Snapshot Delegation
type SnapshotDelegation struct {

	// === Internal fields ===
	contract *core.Contract
}

// ====================
// === Constructors ===
// ====================

// Creates a new Snapshot Delegation contract binding
func NewSnapshotDelegation(address common.Address, client core.ExecutionClient) (*SnapshotDelegation, error) {
	// Parse the ABI
	var err error
	faucetOnce.Do(func() {
		var parsedAbi abi.ABI
		parsedAbi, err = abi.JSON(strings.NewReader(snapshotDelegationAbiString))
		if err == nil {
			snapshotAbi = parsedAbi
		}
	})
	if err != nil {
		return nil, fmt.Errorf("error parsing snapshot delegation ABI: %w", err)
	}

	// Create the contract
	contract := &core.Contract{
		Contract: bind.NewBoundContract(address, snapshotAbi, client, client, client),
		Address:  &address,
		ABI:      &snapshotAbi,
		Client:   client,
	}

	return &SnapshotDelegation{
		contract: contract,
	}, nil
}

// =============
// === Calls ===
// =============

// Get the delegate for the provided address
func (c *SnapshotDelegation) Delegation(mc *batch.MultiCaller, out *common.Address, address common.Address, id common.Hash) {
	core.AddCall(mc, c.contract, out, "delegation", address, id)
}

// ====================
// === Transactions ===
// ====================

// Get info for setting the snapshot delegate
func (c *SnapshotDelegation) SetDelegate(id common.Hash, delegate common.Address, opts *bind.TransactOpts) (*core.TransactionInfo, error) {
	return core.NewTransactionInfo(c.contract, "setDelegate", opts, id, delegate)
}

// Get info for clearing the snapshot delegate
func (c *SnapshotDelegation) ClearDelegate(id common.Hash, opts *bind.TransactOpts) (*core.TransactionInfo, error) {
	return core.NewTransactionInfo(c.contract, "clearDelegate", opts, id)
}
