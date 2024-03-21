package cmd

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/spf13/cobra"
	"github.com/stackup-wallet/stackup-bundler/pkg/userop"
	"math/big"
	"os"
)

var aaOwnerPrivateKey string
var aaSender string
var aaNonce string
var aaInitCode string
var aaCallGasLimit string
var aaVerificationGasLimit string
var aaPreVerificationGas string
var aaMaxFeePerGas string
var aaMaxPriorityFeePerGas string
var aaPaymasterAndData string

// aaSimpleAccountCmd represents the aa-simple-account command
var aaSimpleAccountCmd = &cobra.Command{
	Use:   "aa-simple-account",
	Short: "AA (EIP4337) simple account, owned by an EOA account",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
		os.Exit(1)
	},
}

func init() {
	aaSimpleAccountCmd.PersistentFlags().StringVarP(&aaOwnerPrivateKey, "owner-private-key", "", "", "The private key of owner of AA simple account contract")
	aaSimpleAccountCmd.PersistentFlags().StringVarP(&aaSender, "aa-sender", "", "", "The field sender in User Operation")
	aaSimpleAccountCmd.PersistentFlags().StringVarP(&aaNonce, "aa-nonce", "", "", "The field nonce in User Operation")
	aaSimpleAccountCmd.PersistentFlags().StringVarP(&aaInitCode, "aa-init-code", "", "", "The field initCode in User Operation")
	aaSimpleAccountCmd.PersistentFlags().StringVarP(&aaCallGasLimit, "aa-call-gas-limit", "", "", "The field callGasLimit in User Operation")
	aaSimpleAccountCmd.PersistentFlags().StringVarP(&aaVerificationGasLimit, "aa-verification-gas-limit", "", "", "The field verificationGasLimit in User Operation")
	aaSimpleAccountCmd.PersistentFlags().StringVarP(&aaPreVerificationGas, "aa-pre-verification-gas", "", "", "The field preVerificationGas in User Operation")
	aaSimpleAccountCmd.PersistentFlags().StringVarP(&aaMaxFeePerGas, "aa-max-fee-per-gas", "", "", "The field maxFeePerGas in User Operation")
	aaSimpleAccountCmd.PersistentFlags().StringVarP(&aaMaxPriorityFeePerGas, "aa-max-priority-fee-per-gas", "", "", "The field maxPriorityFeePerGas in User Operation")
	aaSimpleAccountCmd.PersistentFlags().StringVarP(&aaPaymasterAndData, "aa-paymaster-and-data", "", "", "The field paymasterAndData in User Operation")
}

func buildUserOpForEstimateGas(callData []byte) (userop.UserOperation, error) {
	var err error
	var uo = userop.UserOperation{
		Sender: GetSender(),
		//Nonce:                nil,
		//InitCode:             nil,
		CallData: callData,
		// For EstimateGas, CallGasLimit can be zero
		CallGasLimit: big.NewInt(0),
		// For EstimateGas, VerificationGasLimit can be zero
		VerificationGasLimit: big.NewInt(0),
		// For EstimateGas, PreVerificationGas can be zero
		PreVerificationGas: big.NewInt(0),
		// Set low value in order to avoid error: "AA23 reverted (or OOG)" or "AA51 prefund below actualGasCost"
		MaxFeePerGas: big.NewInt(1),
		// Set low value in order to avoid error: "AA23 reverted (or OOG)" or "AA51 prefund below actualGasCost"
		MaxPriorityFeePerGas: big.NewInt(1),
		//PaymasterAndData:     nil,
		// This is dummySig
		Signature: hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
	}

	uo.Sender = GetSender()
	uo.Nonce, err = GetNonce(uo.Sender.String())
	if err != nil {
		return userop.UserOperation{}, err
	}
	uo.InitCode, err = GetInitCode()
	if err != nil {
		return userop.UserOperation{}, err
	}
	uo.PaymasterAndData, err = GetPaymasterAndData()
	if err != nil {
		return userop.UserOperation{}, err
	}

	return uo, nil
}

func ParseBigInt(input string) (*big.Int, error) {
	if has0xPrefix(input) {
		// Hex string
		return hexutil.DecodeBig(input)
	} else {
		// Decimalist
		n := new(big.Int)
		n, ok := n.SetString(input, 10)
		if !ok {
			return nil, fmt.Errorf("parse big int failed: %s", input)
		}
		return n, nil
	}
}

func GetSender() common.Address {
	if aaSender != "" {
		return common.HexToAddress(aaSender)
	} else {
		panic("--aa-sender is required")
	}
}

// GetNonce get nonce of AA contract
// If specified in the parameter, it is used directly.
// If not specified, it is fetched from function getNonce in EntryPoint
func GetNonce(sender string) (*big.Int, error) {
	if aaNonce != "" {
		// Read it from parameter
		return ParseBigInt(aaNonce)
	}

	// Query nonce from function getNonce in EntryPoint
	funcSignature := "function getNonce(address sender, uint192 key)"
	inputArgData := []string{
		sender,
		"0",
	}
	txInputData, err := buildTxInputData(funcSignature, inputArgData)
	if err != nil {
		return nil, fmt.Errorf("buildTxInputData failed: %w", err)
	}

	contract := aaEntryPoint
	output, err := Call(globalClient.RpcClient, contract, txInputData)
	if err != nil {
		return nil, fmt.Errorf("call failed: %w", err)
	}

	z := new(big.Int)
	z.SetBytes(output)

	return z, nil
}

func GetInitCode() ([]byte, error) {
	if aaInitCode != "" {
		if has0xPrefix(aaInitCode) {
			return hex.DecodeString(aaInitCode[2:])
		} else {
			return hex.DecodeString(aaInitCode)
		}
	} else {
		return nil, nil
	}
}

func GetPaymasterAndData() ([]byte, error) {
	if aaPaymasterAndData != "" {
		if has0xPrefix(aaPaymasterAndData) {
			return hex.DecodeString(aaPaymasterAndData[2:])
		} else {
			return hex.DecodeString(aaPaymasterAndData)
		}
	} else {
		return nil, nil
	}
}
