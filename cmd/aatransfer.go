package cmd

import (
	"context"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"log"
	"math/big"
)

var aaTransferUnit string

func init() {
	aaSimpleAccountCmd.AddCommand(aaTransferCmd)

	aaTransferCmd.Flags().StringVarP(&aaTransferUnit, "unit", "u", "ether", "wei | gwei | ether, unit of amount")
}

// aaTransferCmd represents the AA Simple Account transfer command
var aaTransferCmd = &cobra.Command{
	Use:   "transfer TARGET-ADDRESS AMOUNT",
	Short: "Transfer AMOUNT of eth from AA-ACCOUNT-CONTRACT to TARGET-ADDRESS",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if aaOwnerPrivateKey == "" {
			log.Fatalf("--owner-private-key is required for this command")
		}
		ownerPrivateKey := buildPrivateKeyFromHex(aaOwnerPrivateKey)

		targetAddress := args[0]
		transferAmt := args[1]
		if !isValidEthAddress(targetAddress) {
			log.Fatalf("%v is not a valid eth address", targetAddress)
		}
		amount := decimal.RequireFromString(transferAmt)
		amountInWei := unify2Wei(amount, aaTransferUnit)

		InitGlobalClient(globalOptNodeUrl)

		funcSignature := "function execute(address dest, uint256 value, bytes calldata func)"
		inputArgData := []string{
			targetAddress,
			amountInWei.String(),
			"0x",
		}
		callData, err := buildTxInputData(funcSignature, inputArgData)
		checkErr(err)

		uo, err := buildUserOpForEstimateGas(callData)
		checkErr(err)

		// Estimate PreVerificationGas/VerificationGasLimit/CallGasLimit
		preVerificationGas, verificationGas, callGasLimit, err := estimateUserOperationGas(uo, aaEntryPoint)
		checkErr(err)
		// Add 5000 buffer to avoid error: "preVerificationGas: below expected gas of 44068"
		uo.PreVerificationGas = new(big.Int).Add(preVerificationGas, big.NewInt(5000))
		// Add 20000 buffer to avoid error: "AA40 over verificationGasLimit"
		uo.VerificationGasLimit = new(big.Int).Add(verificationGas, big.NewInt(20000))
		uo.CallGasLimit = callGasLimit

		if aaPreVerificationGas != "" {
			uo.PreVerificationGas, err = ParseBigInt(aaPreVerificationGas)
			checkErr(err)
		}
		if aaVerificationGasLimit != "" {
			uo.VerificationGasLimit, err = ParseBigInt(aaVerificationGasLimit)
			checkErr(err)
		}
		if aaCallGasLimit != "" {
			uo.CallGasLimit, err = ParseBigInt(aaCallGasLimit)
			checkErr(err)
		}

		// Estimate MaxFeePerGas and MaxPriorityFeePerGas
		maxFeePerGasEstimate, maxPriorityFeePerGasEstimate, err := getEIP1559GasPrice(globalClient.EthClient)
		checkErr(err)
		uo.MaxFeePerGas = maxFeePerGasEstimate
		uo.MaxPriorityFeePerGas = maxPriorityFeePerGasEstimate

		if aaMaxFeePerGas != "" {
			uo.MaxFeePerGas, err = ParseBigInt(aaMaxFeePerGas)
			checkErr(err)
		}
		if aaMaxPriorityFeePerGas != "" {
			uo.MaxPriorityFeePerGas, err = ParseBigInt(aaMaxPriorityFeePerGas)
			checkErr(err)
		}

		chainID, err := globalClient.EthClient.NetworkID(context.Background())
		checkErr(err)

		userOpHash := uo.GetUserOpHash(aaEntryPoint, chainID)
		log.Printf("userOpHash: %s", userOpHash)

		sig, err := personalSign(userOpHash.Bytes(), ownerPrivateKey)
		checkErr(err)
		uo.Signature = sig

		err = sendUserOperation(uo, aaEntryPoint)
		checkErr(err)

		log.Printf("https://www.jiffyscan.xyz/userOpHash/%s", userOpHash)
	},
}
