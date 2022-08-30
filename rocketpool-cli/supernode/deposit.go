package supernode

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rocket-pool/rocketpool-go/utils/eth"
	"github.com/urfave/cli"

	"github.com/rocket-pool/smartnode/shared/services/gas"
	"github.com/rocket-pool/smartnode/shared/services/rocketpool"
	cliutils "github.com/rocket-pool/smartnode/shared/utils/cli"
	"github.com/rocket-pool/smartnode/shared/utils/math"
)

func nodeDeposit(c *cli.Context) error {

	// Get RP client
	rp, err := rocketpool.NewClientFromCtx(c)
	if err != nil {
		return err
	}
	defer rp.Close()

	// Check and assign the EC status
	err = cliutils.CheckClientStatus(rp)
	if err != nil {
		return err
	}

	// Make sure ETH2 is on the correct chain
	depositContractInfo, err := rp.DepositContractInfo()
	if err != nil {
		return err
	}
	if depositContractInfo.RPNetwork != depositContractInfo.BeaconNetwork ||
		depositContractInfo.RPDepositContract != depositContractInfo.BeaconDepositContract {
		cliutils.PrintDepositMismatchError(
			depositContractInfo.RPNetwork,
			depositContractInfo.BeaconNetwork,
			depositContractInfo.RPDepositContract,
			depositContractInfo.BeaconDepositContract)
		return nil
	}

	fmt.Println("Your eth2 client is on the correct network.\n")

	// Force 16 ETH minipools as the only option after much community discussion
	amountWei := eth.EthToWei(16.0)

	// Get the supernode address
	var supernodeAddress common.Address
	supernodeAddressString := c.String("supernode-address")
	if supernodeAddressString != "" {
		supernodeAddress, err = cliutils.ValidateAddress("supernode-address", supernodeAddressString)
		if err != nil {
			return fmt.Errorf("Error parsing supernode-address [%s]: %w", supernodeAddressString, err)
		}
	} else {
		supernodeAddressString = cliutils.Prompt("What is the address of the supernode you want to create a minipool for?", "^0x[0-9a-fA-F]{40}$", "Invalid address")
		supernodeAddress, err = cliutils.ValidateAddress("supernode-address", supernodeAddressString)
		if err != nil {
			return fmt.Errorf("Error parsing supernode address [%s]: %w", supernodeAddressString, err)
		}
	}

	// Get minipool salt
	var salt *big.Int
	if c.String("salt") != "" {
		var success bool
		salt, success = big.NewInt(0).SetString(c.String("salt"), 0)
		if !success {
			return fmt.Errorf("Invalid minipool salt: %s", c.String("salt"))
		}
	} else {
		buffer := make([]byte, 32)
		_, err = rand.Read(buffer)
		if err != nil {
			return fmt.Errorf("Error generating random salt: %w", err)
		}
		salt = big.NewInt(0).SetBytes(buffer)
	}

	// Check deposit can be made
	canDeposit, err := rp.CanSupernodeDeposit(amountWei, supernodeAddress, salt)
	if err != nil {
		return err
	}
	if !canDeposit.CanDeposit {
		fmt.Println("Cannot make node deposit:")
		if canDeposit.InsufficientBalance {
			fmt.Println("The node's ETH balance is insufficient.")
		}
		if canDeposit.InsufficientRplStake {
			fmt.Println("The node has not staked enough RPL to collateralize a new minipool.")
		}
		if canDeposit.InvalidAmount {
			fmt.Println("The deposit amount is invalid.")
		}
		if canDeposit.UnbondedMinipoolsAtMax {
			fmt.Println("The node cannot create any more unbonded minipools.")
		}
		if canDeposit.DepositDisabled {
			fmt.Println("Node deposits are currently disabled.")
		}
		if !canDeposit.InConsensus {
			fmt.Println("The RPL price and total effective staked RPL of the network are still being voted on by the Oracle DAO.\nPlease try again in a few minutes.")
		}
		return nil
	}

	if c.String("salt") != "" {
		fmt.Printf("Using custom salt %s, your minipool address will be %s.\n\n", c.String("salt"), canDeposit.MinipoolAddress.Hex())
	}

	// Check to see if eth2 is synced
	colorReset := "\033[0m"
	colorRed := "\033[31m"
	colorYellow := "\033[33m"
	syncResponse, err := rp.NodeSync()
	if err != nil {
		fmt.Printf("%s**WARNING**: Can't verify the sync status of your consensus client.\nYOU WILL LOSE ETH if your minipool is activated before it is fully synced.\n"+
			"Reason: %s\n%s", colorRed, err, colorReset)
	} else {
		if syncResponse.BcStatus.PrimaryClientStatus.IsSynced {
			fmt.Printf("Your consensus client is synced, you may safely create a minipool.\n")
		} else if syncResponse.BcStatus.FallbackEnabled {
			if syncResponse.BcStatus.FallbackClientStatus.IsSynced {
				fmt.Printf("Your fallback consensus client is synced, you may safely create a minipool.\n")
			} else {
				fmt.Printf("%s**WARNING**: neither your primary nor fallback consensus clients are fully synced.\nYOU WILL LOSE ETH if your minipool is activated before they are fully synced.\n%s", colorRed, colorReset)
			}
		} else {
			fmt.Printf("%s**WARNING**: your primary consensus client is either not fully synced or offline and you do not have a fallback client configured.\nYOU WILL LOSE ETH if your minipool is activated before it is fully synced.\n%s", colorRed, colorReset)
		}
	}

	// Assign max fees
	err = gas.AssignMaxFeeAndLimit(canDeposit.GasInfo, rp, c.Bool("yes"))
	if err != nil {
		return err
	}

	// Prompt for confirmation
	if !(c.Bool("yes") || cliutils.Confirm(fmt.Sprintf(
		"You are about to deposit %.6f ETH to create a minipool under supernode %s.\n"+
			"%sARE YOU SURE YOU WANT TO DO THIS? Running a minipool is a long-term commitment, and this action cannot be undone!%s",
		math.RoundDown(eth.WeiToEth(amountWei), 6),
		supernodeAddress.Hex(),
		colorYellow,
		colorReset))) {
		fmt.Println("Cancelled.")
		return nil
	}

	// Make deposit
	response, err := rp.SupernodeDeposit(amountWei, supernodeAddress, salt)
	if err != nil {
		return err
	}

	// Log and wait for the minipool address
	fmt.Printf("Creating minipool...\n")
	cliutils.PrintTransactionHash(rp, response.TxHash)
	_, err = rp.WaitForTransaction(response.TxHash)
	if err != nil {
		return err
	}

	// Log & return
	fmt.Printf("The node deposit of %.6f ETH was made successfully!\n", math.RoundDown(eth.WeiToEth(amountWei), 6))
	fmt.Printf("Your new minipool's address is: %s\n", response.MinipoolAddress)
	fmt.Printf("The validator pubkey is: %s\n\n", response.ValidatorPubkey.Hex())

	fmt.Println("Your minipool is now in Initialized status.")
	fmt.Println("Once the 16 ETH deposit has been matched by the staking pool, it will move to Prelaunch status.")
	fmt.Printf("After that, it will move to Staking status once %s have passed.\n", response.ScrubPeriod)
	fmt.Println("You can watch its progress using `rocketpool service logs node`.")

	return nil

}
