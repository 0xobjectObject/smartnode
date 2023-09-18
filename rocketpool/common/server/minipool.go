package server

import (
	"context"
	"fmt"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	batch "github.com/rocket-pool/batch-query"
	"github.com/rocket-pool/rocketpool-go/minipool"
	"github.com/rocket-pool/rocketpool-go/node"
	"github.com/rocket-pool/smartnode/rocketpool/common/services"
	"github.com/rocket-pool/smartnode/shared/types/api"
)

const (
	minipoolBatchSize int = 100
)

// Wrapper for callbacks used by functions that will query all of the node's minipools - they follow this pattern:
// Create bindings, query the chain, return prematurely if some state isn't correct, query all of the minipools, and process them to
// populate a response.
// Structs implementing this will handle the caller-specific functionality.
type IMinipoolCallContext[DataType any] interface {
	// Initialize the context with any bootstrapping, requirements checks, or bindings it needs to set up
	Initialize() error

	// Used to get any supplemental state required during initialization - anything in here will be fed into an rp.Query() multicall
	GetState(node *node.Node, mc *batch.MultiCaller)

	// Check the initialized state after being queried to see if the response needs to be updated and the query can be ended prematurely
	// Return true if the function should continue, or false if it needs to end and just return the response as-is
	CheckState(node *node.Node, data *DataType) bool

	// Get whatever details of the given minipool are necessary; this will be passed into an rp.BatchQuery call, one run per minipool
	// belonging to the node
	GetMinipoolDetails(mc *batch.MultiCaller, mp minipool.Minipool, index int)

	// Prepare the response data using all of the provided artifacts
	PrepareData(addresses []common.Address, mps []minipool.Minipool, data *DataType) error
}

// Interface for minipool call context factories - these will be invoked during route handling to create the
// unique context for the route
type IMinipoolCallContextFactory[ContextType IMinipoolCallContext[DataType], DataType any] interface {
	// Create the context for the route
	Create(vars map[string]string) (ContextType, error)
}

// Registers a new route with the router, which will invoke the provided factory to create and execute the context
// for the route when it's called; use this for complex calls that will iterate over and query each minipool in the node
func RegisterMinipoolRoute[ContextType IMinipoolCallContext[DataType], DataType any](
	router *mux.Router,
	functionName string,
	factory IMinipoolCallContextFactory[ContextType, DataType],
	serviceProvider *services.ServiceProvider,
) {
	router.HandleFunc(fmt.Sprintf("/%s", functionName), func(w http.ResponseWriter, r *http.Request) {
		// Create the handler and deal with any input validation errors
		vars := mux.Vars(r)
		context, err := factory.Create(vars)
		if err != nil {
			handleInputError(w, err)
			return
		}

		// Run the context's processing routine
		response, err := runMinipoolRoute[DataType](context, serviceProvider)
		handleResponse(w, response, err)
	})
}

// Create a scaffolded generic minipool query, with caller-specific functionality where applicable
func runMinipoolRoute[DataType any](ctx IMinipoolCallContext[DataType], serviceProvider *services.ServiceProvider) (*api.ApiResponse[DataType], error) {
	// Common requirements
	err := serviceProvider.RequireNodeRegistered()
	if err != nil {
		return nil, err
	}

	// Get the services
	w := serviceProvider.GetWallet()
	rp := serviceProvider.GetRocketPool()
	nodeAddress, _ := serviceProvider.GetWallet().GetAddress()
	walletStatus := w.GetStatus()

	// Get the latest block for consistency
	latestBlock, err := rp.Client.BlockNumber(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error getting latest block number: %w", err)
	}
	opts := &bind.CallOpts{
		BlockNumber: big.NewInt(int64(latestBlock)),
	}

	// Create the bindings
	node, err := node.NewNode(rp, nodeAddress)
	if err != nil {
		return nil, fmt.Errorf("error creating node %s binding: %w", nodeAddress.Hex(), err)
	}

	// Supplemental function-specific bindings
	err = ctx.Initialize()
	if err != nil {
		return nil, err
	}

	// Get contract state
	err = rp.Query(func(mc *batch.MultiCaller) error {
		node.GetMinipoolCount(mc)
		ctx.GetState(node, mc)
		return nil
	}, opts)
	if err != nil {
		return nil, fmt.Errorf("error getting contract state: %w", err)
	}

	// Create the response and data
	data := new(DataType)
	response := &api.ApiResponse[DataType]{
		WalletStatus: walletStatus,
		Data:         data,
	}

	// Supplemental function-specific check to see if minipool processing should continue
	if !ctx.CheckState(node, data) {
		return response, nil
	}

	// Get the minipool addresses for this node
	addresses, err := node.GetMinipoolAddresses(node.Details.MinipoolCount.Formatted(), opts)
	if err != nil {
		return nil, fmt.Errorf("error getting minipool addresses: %w", err)
	}

	// Create each minipool binding
	mps, err := minipool.CreateMinipoolsFromAddresses(rp, addresses, false, opts)
	if err != nil {
		return nil, fmt.Errorf("error creating minipool bindings: %w", err)
	}

	// Get the relevant details
	err = rp.BatchQuery(len(addresses), minipoolBatchSize, func(mc *batch.MultiCaller, i int) error {
		ctx.GetMinipoolDetails(mc, mps[i], i) // Supplemental function-specific minipool details
		return nil
	}, opts)
	if err != nil {
		return nil, fmt.Errorf("error getting minipool details: %w", err)
	}

	// Supplemental function-specific response construction
	err = ctx.PrepareData(addresses, mps, data)
	if err != nil {
		return nil, err
	}

	// Return
	return response, nil
}