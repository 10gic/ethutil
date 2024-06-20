package cmd

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var deployABIFile string
var deployBinFile string
var deploySrcFile string
var deployContractName string
var deployValueUnit string
var deployValue string

func init() {
	deployCmd.Flags().StringVarP(&deployABIFile, "abi-file", "", "", "the path of abi file, if 'constructor signature' is specified, this option cannot be specified")
	deployCmd.Flags().StringVarP(&deployBinFile, "bin-file", "", "", "the path of byte code file of contract")
	deployCmd.Flags().StringVarP(&deploySrcFile, "src-file", "", "", "the path of source file of contract, launch tool solcjs to compile it. If this option is specified, --bin-file, --abi-file, 'constructor signature' cannot be specified")
	deployCmd.Flags().StringVarP(&deployContractName, "contract-name", "", "", "the contract in source file you want to deploy, if it's not specified, auto find the LAST contract in source file")
	deployCmd.Flags().StringVarP(&deployValueUnit, "unit", "u", "ether", "wei | gwei | ether, unit of amount")
	deployCmd.Flags().StringVarP(&deployValue, "value", "", "0", "the amount you want to transfer when deploy contract, unit is ether and can be changed by --unit")

}

var deployCmd = &cobra.Command{
	Use:   "deploy [constructor signature] arg1 arg2 ...",
	Short: "Deploy contract",
	Run: func(cmd *cobra.Command, args []string) {
		if deployBinFile == "" && deploySrcFile == "" {
			log.Fatalf("must specify --bin-file or --src-file")
		}
		log.Printf("Current chain is %v", globalOptChain)

		InitGlobalClient(globalOptNodeUrl)

		var funcSignature string
		var inputArgData []string
		var bytecodeByteArray []byte

		if deploySrcFile != "" { // source file provided
			// compile source using solcjs, set deployABIFile, deployBinFile

			dir, err := os.MkdirTemp("", "ethutil-deploy")
			if err != nil {
				log.Fatal(err)
			}
			defer os.RemoveAll(dir)

			//log.Printf("solcjs output to dir %v", dir)

			cmd := exec.Command("solcjs", "--base-path", ".", "--include-path", "./node_modules", "--bin", "--abi", "--output-dir", dir, deploySrcFile)
			log.Printf("executing command %v", cmd.String())
			var out bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &stderr
			err = cmd.Run()
			if err != nil {
				log.Fatal(fmt.Sprint(err) + ": " + stderr.String())
			}

			if deployContractName == "" {
				log.Printf("option --contract-name is not specified, use last contract in source file")
				// If contract name is not specified, use last contract in source file
				deployContractName = findContractName(deploySrcFile)
				if len(deployContractName) == 0 {
					log.Fatalf("no contract name found in file %v", deploySrcFile)
				}
			}

			log.Printf("deploying contract %v", deployContractName)

			abiMatches, err := filepath.Glob(fmt.Sprintf("%v/*_%v.abi", dir, deployContractName))
			checkErr(err)
			if len(abiMatches) > 1 {
				log.Fatalf("more than one abi files found: %v", abiMatches)
			}
			deployABIFile = abiMatches[0]

			binMatches, err := filepath.Glob(fmt.Sprintf("%v/*_%v.bin", dir, deployContractName))
			checkErr(err)
			if len(binMatches) > 1 {
				log.Fatalf("more than one bin files found: %v", binMatches)
			}
			deployBinFile = binMatches[0]
		}

		if deployABIFile == "" { // abi file not provided
			if len(args) > 0 {
				funcSignature = args[0]
				inputArgData = args[1:]
			}
		} else { // abi file provided
			abiContent, err := os.ReadFile(deployABIFile)
			checkErr(err)

			funcSignature, err = extractFuncDefinition(string(abiContent), "constructor")
			checkErr(err)
			// log.Printf("extract func definition from abi: %v", funcSignature)

			inputArgData = args[0:]
		}

		bytecode, err := os.ReadFile(deployBinFile)
		checkErr(err)

		var bytecodeHex = strings.TrimSpace(string(bytecode))
		// remove leading 0x
		if strings.HasPrefix(bytecodeHex, "0x") {
			bytecodeHex = bytecodeHex[2:]
		}
		bytecodeByteArray, err = hex.DecodeString(bytecodeHex)
		if err != nil {
			log.Fatal("--bin-file invalid")
		}

		txData, err := buildTxDataForContractDeploy(funcSignature, inputArgData, bytecodeByteArray)
		checkErr(err)
		// log.Printf("txData=%s", hex.Dump(txData))

		if globalOptPrivateKey == "" {
			log.Fatalf("--private-key is required for deploy command")
		}

		var value = decimal.RequireFromString(deployValue)
		var valueInWei = unify2Wei(value, deployValueUnit)

		tx, err := Transact(globalClient.RpcClient, globalClient.EthClient, buildPrivateKeyFromHex(globalOptPrivateKey), nil, valueInWei.BigInt(), nil, txData)
		checkErr(err)

		log.Printf("transaction %s finished", tx)

	},
}

// findContractName find last contract name in source file
func findContractName(deploySrcFile string) string {
	srcContent, err := os.ReadFile(deploySrcFile)
	checkErr(err)

	re := regexp.MustCompile(`(?m)^[ ]*contract[ ]+[a-zA-Z0-9_]+ `)
	matches := re.FindAllString(string(srcContent), -1)
	if len(matches) == 0 {
		return ""
	}

	lastContract := matches[len(matches)-1]
	re = regexp.MustCompile(`^[ ]*contract[ ]+(?P<Name>[a-zA-Z0-9_]+) `)
	matches = re.FindStringSubmatch(lastContract)
	nameIndex := re.SubexpIndex("Name")
	lastContractName := matches[nameIndex]

	return lastContractName
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
