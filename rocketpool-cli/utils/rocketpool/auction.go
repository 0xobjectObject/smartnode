package rocketpool

import (
	"fmt"
	"math/big"
	"net/http"

	"github.com/rocket-pool/smartnode/shared/types/api"
)

type AuctionRequester struct {
	client *http.Client
}

func NewAuctionRequester(client *http.Client) *AuctionRequester {
	return &AuctionRequester{
		client: client,
	}
}

func (r *AuctionRequester) GetName() string {
	return "Auction"
}
func (r *AuctionRequester) GetRoute() string {
	return "auction"
}
func (r *AuctionRequester) GetClient() *http.Client {
	return r.client
}

// Bid on a lot
func (r *AuctionRequester) BidOnLot(lotIndex uint64, amountWei *big.Int) (*api.ApiResponse[api.AuctionBidOnLotData], error) {
	args := map[string]string{
		"index":  fmt.Sprint(lotIndex),
		"amount": amountWei.String(),
	}
	return sendGetRequest[api.AuctionBidOnLotData](r, "bid-lot", "BidOnLot", args)
}

// Claim RPL from a lot
func (r *AuctionRequester) ClaimFromLot(lotIndex uint64) (*api.ApiResponse[api.AuctionClaimFromLotData], error) {
	args := map[string]string{
		"index": fmt.Sprint(lotIndex),
	}
	return sendGetRequest[api.AuctionClaimFromLotData](r, "claim-lot", "ClaimFromLot", args)
}

// Create a new lot
func (r *AuctionRequester) CreateLot() (*api.ApiResponse[api.AuctionCreateLotData], error) {
	return sendGetRequest[api.AuctionCreateLotData](r, "create-lot", "CreateLot", nil)
}

// Get RPL lots for auction
func (r *AuctionRequester) Lots() (*api.ApiResponse[api.AuctionLotsData], error) {
	return sendGetRequest[api.AuctionLotsData](r, "lots", "Lots", nil)
}

// Recover unclaimed RPL from a lot (returning it to the auction contract)
func (r *AuctionRequester) RecoverUnclaimedRplFromLot(lotIndex uint64) (*api.ApiResponse[api.AuctionRecoverRplFromLotData], error) {
	args := map[string]string{
		"index": fmt.Sprint(lotIndex),
	}
	return sendGetRequest[api.AuctionRecoverRplFromLotData](r, "recover-lot", "RecoverUnclaimedRplFromLot", args)
}

// Get RPL auction status
func (r *AuctionRequester) Status() (*api.ApiResponse[api.AuctionStatusData], error) {
	return sendGetRequest[api.AuctionStatusData](r, "status", "Status", nil)
}
