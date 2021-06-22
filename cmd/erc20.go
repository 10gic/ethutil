package cmd

import (
	"context"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/spf13/cobra"
)

var erc20FuncSignature = map[string]string{
	"approve":      "function approve(address delegate, uint256 tokens)  public returns (bool)",
	"transfer":     "function transfer(address receiver, uint256 tokens) public returns (bool)",
	"transferFrom": "function transferFrom(address owner, address buyer, uint256 tokens) public returns (bool)",
	"balanceOf":    "function balanceOf(address tokenOwner) public view returns (uint256)",
	"allowance":    "function allowance(address owner, address delegate) public view returns uint256",
	"totalSupply":  "function totalSupply() public view returns (uint256)",
	"name":         "function name() public view returns (string)",
	"symbol":       "function symbol() public view returns (string)",
	"decimals":     "function decimals() public view returns (uint8)",
	"mint":         "function mint(address account, uint256 amount)",
}

// changeContractState return true if erc20FuncName change contract state
func changeContractState(erc20FuncName string) bool {
	switch erc20FuncName {
	case
		"approve",
		"transfer",
		"transferFrom",
		"mint":
		return true
	}
	return false
}

var erc20Cmd = &cobra.Command{
	Use:   "erc20 contract-address approve/transfer/transferFrom/balanceOf/allowance/totalSupply/name/symbol/decimals/mint [args]",
	Short: "Call ERC20 contract, a helper for subcommand call/query",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		InitGlobalClient(globalOptNodeUrl)

		contractAddr := args[0]
		funcName := args[1]
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

		funcSignature, ok := erc20FuncSignature[funcName]
		if !ok {
			log.Fatalf("%v is NOT supported", funcName)
		}

		txInputData, err := buildTxInputData(funcSignature, inputArgData)
		checkErr(err)

		if globalOptShowInputData {
			log.Printf("input data = %v", hexutil.Encode(txInputData))
		}

		if changeContractState(funcName) {
			gasPrice, err := globalClient.EthClient.SuggestGasPrice(context.Background())
			checkErr(err)

			if globalOptPrivateKey == "" {
				log.Fatalf("--private-key is required for this command")
			} else {
				var contract = common.HexToAddress(contractAddr)
				tx, err := Transact(globalClient.RpcClient, globalClient.EthClient, buildPrivateKeyFromHex(globalOptPrivateKey), &contract, big.NewInt(0), gasPrice, txInputData)
				checkErr(err)

				log.Printf("transaction %s finished", tx)
			}
		} else {
			output, err := Call(globalClient.EthClient, common.HexToAddress(contractAddr), txInputData)
			checkErr(err)

			printContractReturnData(funcSignature, output)
		}

	},
}
