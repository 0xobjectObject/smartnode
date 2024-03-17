package network

import (
	"github.com/urfave/cli"

	"github.com/rocket-pool/smartnode/shared/utils/api"
	cliutils "github.com/rocket-pool/smartnode/shared/utils/cli"
)

// Register subcommands
func RegisterSubcommands(command *cli.Command, name string, aliases []string) {
	command.Subcommands = append(command.Subcommands, cli.Command{
		Name:    name,
		Aliases: aliases,
		Usage:   "Manage Rocket Pool network parameters",
		Subcommands: []cli.Command{

			{
				Name:      "node-fee",
				Aliases:   []string{"f"},
				Usage:     "Get the current network node commission rate",
				UsageText: "rocketpool api network node-fee",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(getNodeFee(c))
					return nil

				},
			},

			{
				Name:      "rpl-price",
				Aliases:   []string{"p"},
				Usage:     "Get the current network RPL price in ETH",
				UsageText: "rocketpool api network rpl-price",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(getRplPrice(c))
					return nil

				},
			},

			{
				Name:      "stats",
				Aliases:   []string{"s"},
				Usage:     "Get stats about the Rocket Pool network and its tokens",
				UsageText: "rocketpool api network stats",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(getStats(c))
					return nil

				},
			},

			{
				Name:      "timezone-map",
				Aliases:   []string{"t"},
				Usage:     "Get the table of node operators by timezone",
				UsageText: "rocketpool api network stats",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(getTimezones(c))
					return nil

				},
			},

			{
				Name:      "can-generate-rewards-tree",
				Usage:     "Check if the rewards tree for the provided interval can be generated",
				UsageText: "rocketpool api network can-generate-rewards-tree index",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}

					index, err := cliutils.ValidateUint("index", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(canGenerateRewardsTree(c, index))
					return nil

				},
			},

			{
				Name:      "generate-rewards-tree",
				Usage:     "Set a request marker for the watchtower to generate the rewards tree for the given interval",
				UsageText: "rocketpool api network generate-rewards-tree index",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}

					index, err := cliutils.ValidateUint("index", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(generateRewardsTree(c, index))
					return nil

				},
			},

			{
				Name:      "dao-proposals",
				Aliases:   []string{"d"},
				Usage:     "Get the currently active DAO proposals",
				UsageText: "rocketpool api network dao-proposals",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(getActiveDAOProposals(c))
					return nil

				},
			},

			{
				Name:      "download-rewards-file",
				Aliases:   []string{"drf"},
				Usage:     "Download a rewards info file from IPFS for the given interval",
				UsageText: "rocketpool api service download-rewards-file interval",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}

					interval, err := cliutils.ValidateUint("interval", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(downloadRewardsFile(c, interval))
					return nil

				},
			},

			{
				Name:      "is-houston-deployed",
				Aliases:   []string{"ihd"},
				Usage:     "Checks if Houston has been deployed yet.",
				UsageText: "rocketpool api network is-houston-deployed",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(isHoustonDeployed(c))
					return nil

				},
			},

			{
				Name:      "latest-delegate",
				Usage:     "Get the address of the latest minipool delegate contract.",
				UsageText: "rocketpool api network latest-delegate",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(getLatestDelegate(c))
					return nil

				},
			},

			{
				Name:      "can-initialize-voting",
				Aliases:   []string{"civ"},
				Usage:     "Checks if voting can be initialized.",
				UsageText: "rocketpool api network can-initialize-voting",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(canNodeInitializeVoting(c))
					return nil

				},
			},
			{
				Name:      "initialize-voting",
				Aliases:   []string{"iv"},
				Usage:     "Initialize voting.",
				UsageText: "rocketpool api network initialize-voting",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(nodeInitializedVoting(c))
					return nil

				},
			},
			{
				Name:      "estimate-set-voting-delegate-gas",
				Usage:     "Estimate the gas required to set an on-chain voting delegate",
				UsageText: "rocketpool api network estimate-set-voting-delegate-gas address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}

					delegate, err := cliutils.ValidateAddress("delegate", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(estimateSetVotingDelegateGas(c, delegate))
					return nil

				},
			},
			{
				Name:      "set-voting-delegate",
				Usage:     "Set an on-chain voting delegate for the node",
				UsageText: "rocketpool api network set-voting-delegate address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}

					delegate, err := cliutils.ValidateAddress("delegate", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(setVotingDelegate(c, delegate))
					return nil

				},
			},
			{
				Name:      "get-current-voting-delegate",
				Usage:     "Get the current on-chain voting delegate for the node",
				UsageText: "rocketpool api network get-current-voting-delegate",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(getCurrentVotingDelegate(c))
					return nil

				},
			},
		},
	})
}
