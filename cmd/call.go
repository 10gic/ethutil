package cmd

import (
	"context"
	"io/ioutil"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

var callCmdABIFile string
var callCmdTransferUnit string
var callCmdTransferAmt string

func init() {
	callCmd.Flags().StringVarP(&callCmdABIFile, "abi-file", "", "", "the path of abi file, if this option specified, 'function signature' can be just function name")
	callCmd.Flags().StringVarP(&callCmdTransferUnit, "unit", "u", "ether", "wei | gwei | ether, unit of amount")
	callCmd.Flags().StringVarP(&callCmdTransferAmt, "value", "", "0", "the amount you want to transfer when call contract, unit is ether and can be changed by --unit")
}

var callCmd = &cobra.Command{
	Use:   "call contract-address 'function signature' arg1 arg2 ...",
	Short: "Invokes the (paid) contract method",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if !validationCallCmdOpts(args) {
			_ = cmd.Help()
			os.Exit(1)
		}

		InitGlobalClient(globalOptNodeUrl)

		contractAddr := args[0]
		funcSignature := args[1]
		inputArgData := args[2:]

		if !globalOptDryRun {
			// don't check contract address if --dry-run specified
			isContract, err := isContractAddress(globalClient.EthClient, common.HexToAddress(contractAddr))
			if err != nil {
				panic(err)
			}
			if !isContract {
				log.Fatalf("%v is NOT a contract address, can not find it from blockchain", contractAddr)
			}
		}

		if callCmdABIFile != "" {
			abiContent, err := ioutil.ReadFile(callCmdABIFile)
			if err != nil {
				log.Fatal(err)
			}
			funcName := funcSignature
			funcSignature, err = extractFuncDefinition(string(abiContent), extractFuncName(funcName))
			checkErr(err)
			// log.Printf("extract func definition from abi: %v", funcSignature)
		}

		txInputData, err := buildTxInputData(funcSignature, inputArgData)
		checkErr(err)

		if globalOptShowInputData {
			log.Printf("input data = %v", hexutil.Encode(txInputData))
		}

		gasPrice, err := globalClient.EthClient.SuggestGasPrice(context.Background())
		checkErr(err)

		if globalOptPrivateKey == "" {
			log.Fatalf("--private-key is required for call command")
		} else {
			var value = decimal.RequireFromString(callCmdTransferAmt)
			var valueInWei = unify2Wei(value, callCmdTransferUnit)

			var contract = common.HexToAddress(contractAddr)
			tx, err := Transact(globalClient.RpcClient, globalClient.EthClient, buildPrivateKeyFromHex(globalOptPrivateKey), &contract, valueInWei.BigInt(), gasPrice, txInputData)
			checkErr(err)

			log.Printf("transaction %s finished", tx)
		}

	},
}

func validationCallCmdOpts(args []string) bool {
	if !isValidEthAddress(args[0]) {
		log.Printf("%s is NOT a valid eth address", args[0])
		return false
	}
	return true
}
