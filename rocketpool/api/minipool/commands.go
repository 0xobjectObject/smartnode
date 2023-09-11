package minipool

import (
	"github.com/urfave/cli"

	types "github.com/rocket-pool/smartnode/shared/types/api"
	"github.com/rocket-pool/smartnode/shared/utils/api"
	cliutils "github.com/rocket-pool/smartnode/shared/utils/cli"
)

// Register subcommands
func RegisterSubcommands(command *cli.Command, name string, aliases []string) {
	command.Subcommands = append(command.Subcommands, cli.Command{
		Name:    name,
		Aliases: aliases,
		Usage:   "Manage the node's minipools",
		Subcommands: []cli.Command{

			{
				Name:      "status",
				Aliases:   []string{"s"},
				Usage:     "Get a list of the node's minipools",
				UsageText: "rocketpool api minipool status",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(getStatus(c))
					return nil

				},
			},

			{
				Name:      "can-stake",
				Usage:     "Check whether the minipool is ready to be staked, moving from prelaunch to staking status",
				UsageText: "rocketpool api minipool can-stake minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(canStakeMinipool(c, minipoolAddress))
					return nil

				},
			},
			{
				Name:      "stake",
				Aliases:   []string{"t"},
				Usage:     "Stake the minipool, moving it from prelaunch to staking status",
				UsageText: "rocketpool api minipool stake minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(stakeMinipool(c, minipoolAddress))
					return nil

				},
			},

			{
				Name:      "can-promote",
				Usage:     "Check whether a vacant minipool is ready to be promoted",
				UsageText: "rocketpool api minipool can-promote minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(canPromoteMinipool(c, minipoolAddress))
					return nil

				},
			},
			{
				Name:      "promote",
				Usage:     "Promote a vacant minipool",
				UsageText: "rocketpool api minipool promote minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(promoteMinipool(c, minipoolAddress))
					return nil

				},
			},

			{
				Name:      "can-refund",
				Usage:     "Check whether the node can refund ETH from the minipool",
				UsageText: "rocketpool api minipool can-refund minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(canRefundMinipool(c, minipoolAddress))
					return nil

				},
			},
			{
				Name:      "refund",
				Aliases:   []string{"r"},
				Usage:     "Refund ETH belonging to the node from a minipool",
				UsageText: "rocketpool api minipool refund minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(refundMinipool(c, minipoolAddress))
					return nil

				},
			},

			{
				Name:      "get-minipool-dissolve-details-for-node",
				Usage:     "Get all of the details for dissolve eligibility of each node's minipools",
				UsageText: "rocketpool api minipool get-minipool-dissolve-details-for-node",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(getMinipoolDissolveDetailsForNode(c))
					return nil

				},
			},
			{
				Name:      "dissolve",
				Aliases:   []string{"d"},
				Usage:     "Dissolve an initialized or prelaunch minipool",
				UsageText: "rocketpool api minipool dissolve minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(dissolveMinipool(c, minipoolAddress))
					return nil

				},
			},

			{
				Name:      "can-exit",
				Usage:     "Check whether the minipool can be exited from the beacon chain",
				UsageText: "rocketpool api minipool can-exit minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(canExitMinipool(c, minipoolAddress))
					return nil

				},
			},
			{
				Name:      "exit",
				Aliases:   []string{"e"},
				Usage:     "Exit a staking minipool from the beacon chain",
				UsageText: "rocketpool api minipool exit minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(exitMinipool(c, minipoolAddress))
					return nil

				},
			},

			{
				Name:      "get-minipool-close-details-for-node",
				Usage:     "Check all of the node's minipools for closure eligibility, and return the details of the closeable ones",
				UsageText: "rocketpool api minipool get-minipool-close-details-for-node",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(getMinipoolCloseDetailsForNode(c))
					return nil

				},
			},
			{
				Name:      "close",
				Aliases:   []string{"c"},
				Usage:     "Withdraw balance from a dissolved minipool and close it",
				UsageText: "rocketpool api minipool close minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(closeMinipool(c, minipoolAddress))
					return nil

				},
			},

			{
				Name:      "get-minipool-delegate-details-for-node",
				Usage:     "Get delegate information for all minipools belonging to the node",
				UsageText: "rocketpool api minipool get-minipool-delegate-details-for-node",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(getMinipoolDelegateDetailsForNode(c))
					return nil

				},
			},
			{
				Name:      "delegate-upgrade",
				Usage:     "Upgrade this minipool to the latest network delegate contract",
				UsageText: "rocketpool api minipool delegate-upgrade minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(upgradeDelegates(c, minipoolAddress))
					return nil

				},
			},

			{
				Name:      "delegate-rollback",
				Usage:     "Rollback the minipool to the previous delegate contract",
				UsageText: "rocketpool api minipool delegate-rollback minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 2); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(rollbackDelegates(c, minipoolAddress))
					return nil

				},
			},

			{
				Name:      "set-use-latest-delegate",
				Usage:     "Set whether or not to ignore the minipool's current delegate and always use the latest delegate instead",
				UsageText: "rocketpool api minipool set-use-latest-delegate minipool-address setting",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 2); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}
					setting, err := cliutils.ValidateBool("setting", c.Args().Get(1))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(setUseLatestDelegates(c, minipoolAddress, setting))
					return nil

				},
			},

			{
				Name:      "get-vanity-artifacts",
				Aliases:   []string{"v"},
				Usage:     "Gets the data necessary to search for vanity minipool addresses",
				UsageText: "rocketpool api minipool get-vanity-artifacts deposit node-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 2); err != nil {
						return err
					}
					depositAmount, err := cliutils.ValidatePositiveWeiAmount("deposit amount", c.Args().Get(0))
					if err != nil {
						return err
					}
					nodeAddressStr := c.Args().Get(1)

					// Run
					api.PrintResponse(getVanityArtifacts(c, depositAmount, nodeAddressStr))
					return nil

				},
			},

			{
				Name:      "get-minipool-begin-reduce-bond-details-for-node",
				Usage:     "Check whether any of the minipools belonging to the node can begin the bond reduction process",
				UsageText: "rocketpool api minipool get-minipool-begin-reduce-bond-details-for-node new-bond-amount-wei",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					newBondAmountWei, err := cliutils.ValidateWeiAmount("new bond amount", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					response, err := runMinipoolQuery[types.GetMinipoolBeginReduceBondDetailsForNodeResponse](c, &minipoolBeginReduceBondManager{
						newBondAmountWei: newBondAmountWei,
					})
					api.PrintResponse(response, err)
					return nil

				},
			},
			{
				Name:      "begin-reduce-bond-amount",
				Usage:     "Begin the bond reduction process for a minipool",
				UsageText: "rocketpool api minipool begin-reduce-bond-amount minipool-address new-bond-amount-wei",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 2); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}
					newBondAmountWei, err := cliutils.ValidateWeiAmount("new bond amount", c.Args().Get(1))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(beginReduceBondAmount(c, minipoolAddress, newBondAmountWei))
					return nil

				},
			},

			{
				Name:      "can-reduce-bond-amount",
				Usage:     "Check if a minipool's bond can be reduced",
				UsageText: "rocketpool api minipool can-reduce-bond-amount minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(canReduceBondAmount(c, minipoolAddress))
					return nil

				},
			},
			{
				Name:      "reduce-bond-amount",
				Usage:     "Reduce a minipool's bond",
				UsageText: "rocketpool api minipool reduce-bond-amount minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(reduceBondAmount(c, minipoolAddress))
					return nil

				},
			},

			{
				Name:      "get-distribute-balance-details-for-node",
				Usage:     "Get the balance distribution details for all of the node's minipools",
				UsageText: "rocketpool api minipool get-distribute-balance-details-for-node",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(getDistributeBalanceDetailsForNode(c))
					return nil

				},
			},
			{
				Name:      "distribute-balance",
				Usage:     "Distribute a minipool's ETH balance",
				UsageText: "rocketpool api minipool distribute-balance minipool-address",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(distributeBalance(c, minipoolAddress))
					return nil

				},
			},

			{
				Name:      "import-key",
				Usage:     "Import a validator private key for a vacant minipool",
				UsageText: "rocketpool api minipool import-key minipool-address mnemonic",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 2); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}
					mnemonic, err := cliutils.ValidateWalletMnemonic("mnemonic", c.Args().Get(1))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(importKey(c, minipoolAddress, mnemonic))
					return nil

				},
			},

			{
				Name:      "can-change-withdrawal-creds",
				Usage:     "Check whether a solo validator's withdrawal credentials can be changed to a minipool address",
				UsageText: "rocketpool api minipool can-change-withdrawal-creds minipool-address mnemonic",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 2); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}
					mnemonic, err := cliutils.ValidateWalletMnemonic("mnemonic", c.Args().Get(1))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(canChangeWithdrawalCreds(c, minipoolAddress, mnemonic))
					return nil

				},
			},
			{
				Name:      "change-withdrawal-creds",
				Usage:     "Change a solo validator's withdrawal credentials to a minipool address",
				UsageText: "rocketpool api minipool change-withdrawal-creds minipool-address mnemonic",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 2); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}
					mnemonic, err := cliutils.ValidateWalletMnemonic("mnemonic", c.Args().Get(1))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(changeWithdrawalCreds(c, minipoolAddress, mnemonic))
					return nil

				},
			},

			{
				Name:      "get-rescue-dissolved-details-for-node",
				Usage:     "Check all of the node's minipools for rescue eligibility, and return the details of the rescuable ones",
				UsageText: "rocketpool api minipool get-rescue-dissolved-details-for-node",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					api.PrintResponse(getMinipoolRescueDissolvedDetailsForNode(c))
					return nil

				},
			},

			{
				Name:      "rescue-dissolved",
				Usage:     "Rescue a dissolved minipool by depositing ETH for it to the Beacon deposit contract",
				UsageText: "rocketpool api minipool rescue-dissolved minipool-address deposit-amount",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 2); err != nil {
						return err
					}
					minipoolAddress, err := cliutils.ValidateAddress("minipool address", c.Args().Get(0))
					if err != nil {
						return err
					}
					depositAmount, err := cliutils.ValidateBigInt("deposit amount", c.Args().Get(1))
					if err != nil {
						return err
					}

					// Run
					api.PrintResponse(rescueDissolvedMinipool(c, minipoolAddress, depositAmount))
					return nil

				},
			},
		},
	})
}
