package cmd

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
)

var callCmd = &cobra.Command{
	Use:   "contract-call contract_address 'function signature' arg1 arg2 ...",
	Short: "Invokes the (paid) contract method",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if !validationCallCmdOpts(args) {
			cmd.Help()
			os.Exit(1)
		}

		client, err := ethclient.Dial(nodeUrlOpt)
		checkErr(err)

		contractAddr := args[0]
		funcSignature := args[1]
		inputArgData := args[2:]

		isContract, err := isContractAddress(client, common.HexToAddress(contractAddr))
		if err != nil {
			panic(err)
		}
		if !isContract {
			log.Printf("%v is NOT a contract address", contractAddr)
			cmd.Help()
			os.Exit(1)
		}

		txData, err := buildTxData(funcSignature, inputArgData)
		checkErr(err)
		// log.Printf("txData=%s", hex.Dump(txData))

		gasPrice, err := client.SuggestGasPrice(context.Background())
		checkErr(err)

		if privateKeyOpt == "" {
			log.Fatalf("--private-key is required for contract-call command")

			output, err := Call(client, common.HexToAddress(contractAddr), txData)
			checkErr(err)

			fmt.Printf("output:\n%s\n", hex.Dump(output))
		} else {
			tx, err := Transact(client, buildPrivateKeyFromHex(privateKeyOpt), common.HexToAddress(contractAddr), big.NewInt(0), gasPrice, txData)
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

func buildTxData(funcSignature string, inputArgData []string) ([]byte, error) {
	funcName, funcArgTypes, err := parseFuncSignature(funcSignature)
	if err != nil {
		return nil, err
	}
	funcSign := funcName + "(" + strings.Join(funcArgTypes, ",") + ")"
	functionSelector := crypto.Keccak256([]byte(funcSign))[0:4]

	if len(funcArgTypes) != len(inputArgData) {
		return nil, fmt.Errorf("invalid input, there are %v args in signature, but %v args are provided", len(funcArgTypes), len(inputArgData))
	}
	data, err := encodeParameters(funcArgTypes, inputArgData)
	if err != nil {
		return nil, fmt.Errorf("encodeParameters fail: %v", err)
	}
	return append(functionSelector, data...), nil
}

// input example: "function add(uint256   xx, address xx, bool xx)"
// output "add, [uint256 address bool]"
func parseFuncSignature(input string) (string, []string, error) {
	if strings.HasPrefix(input, "function ") {
		input = input[len("function "):] // remove leading string "function "
	}
	input = strings.TrimLeft(input, " ")

	leftParenthesisLoc := strings.Index(input, "(")
	if leftParenthesisLoc < 0 {
		return "", nil, fmt.Errorf("char ) is not found in function signature")
	}
	funcName := input[:leftParenthesisLoc] // remove all characters from char '('
	funcName = strings.TrimSpace(funcName)

	rightParenthesisLoc := strings.Index(input, ")")
	if rightParenthesisLoc < 0 {
		return "", nil, fmt.Errorf("char ) is not found in function signature")
	}
	argsPart := input[leftParenthesisLoc+1 : rightParenthesisLoc]
	if strings.TrimSpace(argsPart) == "" {
		return funcName, nil, nil
	}
	args := strings.Split(argsPart, ",")
	for index, arg := range args {
		fields := strings.Fields(arg)
		if len(fields) == 0 {
			return "", nil, fmt.Errorf("signature `%v` invalid, type missing in args", input)
		}
		args[index] = typeNormalize(fields[0]) // first field is type. for example, "uint256 xx", first field is uint256
	}

	return funcName, args, nil
}

func encodeParameters(inputArgTypes, inputArgData []string) ([]byte, error) {
	var theTypes abi.Arguments

	var theArgData []interface{}
	for index, inputType := range inputArgTypes {
		typ, err := abi.NewType(typeNormalize(inputType), "", nil)
		if err != nil {
			return nil, err
		}
		theTypes = append(theTypes, abi.Argument{Type: typ})

		if strings.Contains(inputType, "[") { // array type
			return nil, fmt.Errorf("type %v not handled currently", inputType)
		} else if inputType == "string" {
			theArgData = append(theArgData, inputArgData[index])
		} else if inputType == "int8" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, int8(i))
		} else if inputType == "int16" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, int16(i))
		} else if inputType == "int32" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, int32(i))
		} else if inputType == "int64" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, int64(i))
		} else if inputType == "uint8" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, uint8(i))
		} else if inputType == "uint16" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, uint16(i))
		} else if inputType == "uint32" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, uint32(i))
		} else if inputType == "uint64" {
			i, err := strconv.ParseUint(inputArgData[index], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, uint64(i))
		} else if strings.Contains(inputType, "int") { // 其它包含 int 的情况，如 int24, int256, uint48, uint256, etc
			n := new(big.Int)
			n, ok := n.SetString(inputArgData[index], 10)
			if !ok {
				return nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, n)

		} else if inputType == "bool" {
			if strings.EqualFold(inputArgData[index], "true") {
				theArgData = append(theArgData, true)
			} else if strings.EqualFold(inputArgData[index], "false") {
				theArgData = append(theArgData, false)
			} else {
				return nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
		} else if inputType == "address" {
			theArgData = append(theArgData, common.HexToAddress(inputArgData[index]))
		} else if inputType == "bytes" {
			var inputHex = inputArgData[index]
			if strings.HasPrefix(inputArgData[index], "0x") {
				inputHex = inputArgData[index][2:]
			}
			decoded, err := hex.DecodeString(inputHex)
			if err != nil {
				return nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, decoded)
		} else {
			return nil, fmt.Errorf("type %v not handled currently", inputType)
		}
	}

	bytes, err := theTypes.Pack(theArgData...)
	if err != nil {
		return nil, fmt.Errorf("pack fail: %s", err)
	}
	return bytes, nil
}

func typeNormalize(input string) string {
	if input == "uint" {
		return "uint256"
	} else if input == "int" {
		return "int256"
	} else {
		return input
	}
}
