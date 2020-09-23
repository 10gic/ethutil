package cmd

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

var queryCmdABIFile string

func init() {
	queryCmd.Flags().StringVarP(&queryCmdABIFile, "abi-file", "", "", "the path of abi file, if this option specified, 'function definition' can be just function name")
}

var queryCmd = &cobra.Command{
	Use:   "contract-query contract_address 'function definition' arg1 arg2 ...",
	Short: "Invokes the (constant) contract method",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if !validationQueryCmdOpts(args) {
			cmd.Help()
			os.Exit(1)
		}

		client, err := ethclient.Dial(nodeUrlOpt)
		checkErr(err)

		contractAddr := args[0]
		funcDefinition := args[1]
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

		if queryCmdABIFile != "" {
			abiContent, err := ioutil.ReadFile(queryCmdABIFile)
			if err != nil {
				log.Fatal(err)
			}
			funcName := funcDefinition
			funcDefinition, err = extractFuncDefinition(string(abiContent), extractFuncName(funcName))
			checkErr(err)
			// log.Printf("extract func definition from abi: %v", funcDefinition)
		}

		txData, err := buildTxData(funcDefinition, inputArgData)
		checkErr(err)

		output, err := Call(client, common.HexToAddress(contractAddr), txData)
		checkErr(err)

		var v = make(map[string]interface{})
		returnArgs, err := buildReturnArgs(funcDefinition)
		checkErr(err)

		// Unpack hex data into v
		err = returnArgs.UnpackIntoMap(v, output)
		checkErr(err)

		for _, returnArg := range returnArgs {
			// fmt.Printf("type of v: %v\n", reflect.TypeOf(v[returnArg.Name]))
			if returnArg.Type.T == abi.AddressTy {
				fmt.Printf("%v = %v\n", returnArg.Name, v[returnArg.Name].(common.Address).Hex())
			} else if returnArg.Type.T == abi.SliceTy {
				if returnArg.Type.Elem.T == abi.AddressTy { // element is address
					addresses := v[returnArg.Name].([]common.Address)

					fmt.Printf("%v = [", returnArg.Name)
					for index, address := range addresses {
						fmt.Printf("%v", address.Hex())
						if index < len(addresses) - 1 {
							fmt.Printf(" ") // separator
						}
					}
					fmt.Printf("]")
				} else {
					fmt.Printf("%v = %v\n", returnArg.Name, v[returnArg.Name])
				}
			} else {
				fmt.Printf("%v = %v\n", returnArg.Name, v[returnArg.Name])
			}
		}

		// fmt.Printf("raw output:\n%s\n", hex.Dump(output))
	},
}

func validationQueryCmdOpts(args []string) bool {
	if !isValidEthAddress(args[0]) {
		log.Printf("%s is NOT a valid eth address", args[0])
		return false
	}
	return true
}

// funcDefinition example: "function balanceOf(address _owner) public constant returns (uint balance)"
func buildReturnArgs(funcDefinition string) (abi.Arguments, error) {
	returnsLoc := strings.Index(funcDefinition, "returns")
	if returnsLoc < 0 {
		// return immediately if keyword `returns` no found in input
		return nil, nil
	}
	partAfterReturns := funcDefinition[returnsLoc:]

	leftParenthesisLoc := strings.Index(partAfterReturns, "(")
	if leftParenthesisLoc < 0 {
		return nil, fmt.Errorf("char ) is not found after keyword returns")
	}
	rightParenthesisLoc := strings.Index(partAfterReturns, ")")
	if rightParenthesisLoc < 0 {
		return nil, fmt.Errorf("char ) is not found after keyword returns")
	}

	var theReturnTypes abi.Arguments

	returnPart := partAfterReturns[leftParenthesisLoc+1 : rightParenthesisLoc]
	returnList := strings.Split(returnPart, ",")
	for index, returnElem := range returnList {
		fields := strings.Fields(returnElem)
		if len(fields) == 0 {
			return nil, fmt.Errorf("func definition `%v` invalid, type missing in returns", funcDefinition)
		}

		typ, err := abi.NewType(typeNormalize(fields[0]), "", nil)
		if err != nil {
			return nil, fmt.Errorf("abi.NewType fail: %w", err)
		}

		theReturnName := "ret"+ strconv.FormatInt(int64(index),10) // default name ret0, ret1, etc
		if len(fields) > 1 {
			if fields[1] == "memory" || fields[1] == "calldata" {
				// skip keyword "memory" and "calldata"
				if len(fields) > 2 {
					theReturnName = fields[2]
				}
			} else {
				theReturnName = fields[1]
			}
		}
		theReturnTypes = append(theReturnTypes, abi.Argument{Type: typ, Name: theReturnName})
	}

	return theReturnTypes, nil
}
