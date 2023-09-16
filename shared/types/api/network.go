package api

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type NodeFeeData struct {
	NodeFee       float64 `json:"nodeFee"`
	MinNodeFee    float64 `json:"minNodeFee"`
	TargetNodeFee float64 `json:"targetNodeFee"`
	MaxNodeFee    float64 `json:"maxNodeFee"`
}

type RplPriceData struct {
	RplPrice                    *big.Int `json:"rplPrice"`
	RplPriceBlock               uint64   `json:"rplPriceBlock"`
	MinPer8EthMinipoolRplStake  *big.Int `json:"minPer8EthMinipoolRplStake"`
	MaxPer8EthMinipoolRplStake  *big.Int `json:"maxPer8EthMinipoolRplStake"`
	MinPer16EthMinipoolRplStake *big.Int `json:"minPer16EthMinipoolRplStake"`
	MaxPer16EthMinipoolRplStake *big.Int `json:"maxPer16EthMinipoolRplStake"`
}

type NetworkStatsData struct {
	TotalValueLocked          *big.Int       `json:"totalValueLocked"`
	DepositPoolBalance        *big.Int       `json:"depositPoolBalance"`
	MinipoolCapacity          *big.Int       `json:"minipoolCapacity"`
	StakerUtilization         *big.Int       `json:"stakerUtilization"`
	NodeFee                   *big.Int       `json:"nodeFee"`
	NodeCount                 uint64         `json:"nodeCount"`
	InitializedMinipoolCount  uint64         `json:"initializedMinipoolCount"`
	PrelaunchMinipoolCount    uint64         `json:"prelaunchMinipoolCount"`
	StakingMinipoolCount      uint64         `json:"stakingMinipoolCount"`
	WithdrawableMinipoolCount uint64         `json:"withdrawableMinipoolCount"`
	DissolvedMinipoolCount    uint64         `json:"dissolvedMinipoolCount"`
	FinalizedMinipoolCount    uint64         `json:"finalizedMinipoolCount"`
	RplPrice                  *big.Int       `json:"rplPrice"`
	TotalRplStaked            *big.Int       `json:"totalRplStaked"`
	EffectiveRplStaked        *big.Int       `json:"effectiveRplStaked"`
	RethPrice                 *big.Int       `json:"rethPrice"`
	SmoothingPoolNodes        uint64         `json:"smoothingPoolNodes"`
	SmoothingPoolAddress      common.Address `json:"SmoothingPoolAddress"`
	SmoothingPoolBalance      *big.Int       `json:"smoothingPoolBalance"`
}

type NetworkTimezonesData struct {
	TimezoneCounts map[string]uint64 `json:"timezoneCounts"`
	TimezoneTotal  uint64            `json:"timezoneTotal"`
	NodeTotal      uint64            `json:"nodeTotal"`
}

type CanNetworkGenerateRewardsTreeData struct {
	CurrentIndex   uint64 `json:"currentIndex"`
	TreeFileExists bool   `json:"treeFileExists"`
}

type NetworkDaoProposalsData struct {
	AccountAddress          common.Address         `json:"accountAddress"`
	VotingDelegate          common.Address         `json:"votingDelegate"`
	ActiveSnapshotProposals []SnapshotProposal     `json:"activeSnapshotProposals"`
	ProposalVotes           []SnapshotProposalVote `json:"proposalVotes"`
}

type GetLatestDelegateData struct {
	Address common.Address `json:"address"`
}
