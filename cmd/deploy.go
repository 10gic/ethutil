package cmd

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"math/big"
	"strings"
)

var deployABIFile string
var deployBinFile string

func init() {
	deployCmd.Flags().StringVarP(&deployABIFile, "abi-file", "", "", "the path of abi file, if this option specified, 'constructor signature' must not specified")
	deployCmd.Flags().StringVarP(&deployBinFile, "bin-file", "", "", "the path of byte code file of contract")

	deployCmd.MarkFlagRequired("bin-file")
}

var deployCmd = &cobra.Command{
	Use:   "contract-deploy [constructor signature] arg1 arg2 ...",
	Short: "Deploy contract",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := ethclient.Dial(nodeUrlOpt)
		checkErr(err)

		var funcSignature string
		var inputArgData []string

		if deployABIFile == "" { // abi file not provided
			if len(args) > 0 {
				funcSignature = args[0]
				inputArgData = args[1:]
			}
		} else { // abi file provided
			abiContent, err := ioutil.ReadFile(callCmdABIFile)
			if err != nil {
				log.Fatal(err)
			}
			funcSignature, err = extractFuncDefinition(string(abiContent), "constructor")
			checkErr(err)
			// log.Printf("extract func definition from abi: %v", funcSignature)

			inputArgData = args[0:]
		}

		bytecode, err := ioutil.ReadFile(deployBinFile)
		checkErr(err)

		var bytecodeHex = strings.TrimSpace(string(bytecode))
		// remove leading 0x
		if strings.HasPrefix(bytecodeHex, "0x") {
			bytecodeHex = bytecodeHex[2:]
		}
		bytecodeByteArray, err := hex.DecodeString(bytecodeHex)
		if err != nil {
			log.Fatal("--bin-file invalid")
		}

		txData, err := buildTxDataForContractDeploy(funcSignature, inputArgData, bytecodeByteArray)
		checkErr(err)
		// log.Printf("txData=%s", hex.Dump(txData))

		gasPrice, err := client.SuggestGasPrice(context.Background())
		checkErr(err)

		if privateKeyOpt == "" {
			log.Fatalf("--private-key is required for contract-deploy command")
		} else {
			tx, err := Transact(client, buildPrivateKeyFromHex(privateKeyOpt), nil, big.NewInt(0), gasPrice, txData)
			checkErr(err)

			log.Printf("transaction %s finished", tx)
		}

	},
}

func buildTxDataForContractDeploy(funcSignature string, inputArgData []string, bytecode []byte) ([]byte, error) {
	if funcSignature == "" { // no constructor
		return bytecode, nil
	}

	_, funcArgTypes, err := parseFuncSignature(funcSignature)
	if err != nil {
		return nil, err
	}

	if len(funcArgTypes) != len(inputArgData) {
		return nil, fmt.Errorf("invalid input, there are %v args in constructor, but %v args are provided", len(funcArgTypes), len(inputArgData))
	}
	data, err := encodeParameters(funcArgTypes, inputArgData)
	if err != nil {
		return nil, fmt.Errorf("encodeParameters fail: %v", err)
	}
	return append(bytecode, data...), nil
}
