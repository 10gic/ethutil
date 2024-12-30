package cmd

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"log"
	"math/big"
)

// aaDeployCmd represents the aaDeploy command
var aaDeployCmd = &cobra.Command{
	Use:   "deploy <account-owner-address>",
	Short: "Deploy AA (EIP4337) account contract, solidity source contracts/AASimpleAccountFactory.sol",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		var accountOwnerAddress = args[0]

		InitGlobalClient(globalOptNodeUrl)

		if globalOptPrivateKey == "" {
			log.Fatalf("--private-key is required for this command")
		}

		// Check if SingletonFactory (EIP2470) contract deployed
		isSingletonFactoryDeployed, err := isContractAddress(globalClient.EthClient, singletonFactoryAddr)
		checkErr(err)
		if !isSingletonFactoryDeployed {
			log.Fatalf("SingletonFactory (EIP2470) contract not found")
		}

		// Check if AASimpleAccountFactory contract deployed, deploy it when necessary
		isAASimpleAccountFactoryDeployed, err := isContractAddress(globalClient.EthClient, getAASimpleAccountFactoryAddress())
		checkErr(err)
		if !isAASimpleAccountFactoryDeployed {
			log.Printf("AA Simple Account Factory contract not deployed, it will be deployed firstly")

			if err := deployAASimpleAccountFactory(); err != nil {
				log.Fatalf("deploy AA Simple Account Factory failed: %s", err)
			}
		}

		// Check if AASimpleAccount contract deployed, deploy it when necessary
		aaSimpleAccountAddr := getAASimpleAccountAddr(accountOwnerAddress)
		isAASimpleAccountDeployed, err := isContractAddress(globalClient.EthClient, aaSimpleAccountAddr)
		checkErr(err)
		if !isAASimpleAccountDeployed {
			if err := deployAASimpleAccount(accountOwnerAddress); err != nil {
				log.Fatalf("deploy AA Account failed: %s", err)
			}
			log.Printf("AA Account (owner %s) deployed at %s", accountOwnerAddress, aaSimpleAccountAddr)
		} else {
			log.Printf("AA Account (owner %s) already deployed at %s, do nothing",
				accountOwnerAddress,
				aaSimpleAccountAddr)
		}
	},
}

// deployAASimpleAccountFactory deploy AA Simple Account Factory by calling function deploy in SingletonFactory (EIP2470)
func deployAASimpleAccountFactory() error {
	// https://etherscan.io/address/0xce0042B868300000d44A59004Da54A005ffdcf9f#code
	funcSignature := "function deploy(bytes memory _initCode, bytes32 _salt)"
	inputArgData := []string{
		aaSimpleAccountFactoryInitCode,
		aaSimpleAccountFactorySalt,
	}
	txInputData, err := buildTxInputData(funcSignature, inputArgData)
	if err != nil {
		return err
	}

	log.Printf("deploying AA simple account factory contract")
	contract := singletonFactoryAddr
	tx, err := Transact(globalClient.RpcClient, globalClient.EthClient, hexToPrivateKey(globalOptPrivateKey), &contract, big.NewInt(0), nil, txInputData)
	if err != nil {
		return err
	}
	log.Printf("deploying AA simple account factory contract, tx %s", tx)

	return nil
}

// getAASimpleAccountAddr query the address of AA Simple Account by calling function getAddress in SimpleAccountFactory (contracts/AASimpleAccountFactory.sol)
func getAASimpleAccountAddr(accountOwner string) common.Address {
	funcSignature := "function getAddress(bytes32 _salt, address _accountOwner, address _entryPoint) public view returns (address)"
	inputArgData := []string{
		"0x0000000000000000000000000000000000000000000000000000000000000000", // salt
		accountOwner,
		aaEntryPoint.String(),
	}
	txInputData, err := buildTxInputData(funcSignature, inputArgData)

	contract := getAASimpleAccountFactoryAddress()
	output, err := Call(globalClient.RpcClient, contract, txInputData)
	checkErr(err)

	return common.BytesToAddress(output)
}

// deployAASimpleAccount deploy AA Simple Account by calling function createAccount in AASimpleAccountFactory (contracts/AASimpleAccountFactory.sol)
func deployAASimpleAccount(accountOwner string) error {
	// See AASimpleAccountFactory.sol
	funcSignature := "function createAccount(bytes32 _salt, address _accountOwner, address _entryPoint)"
	inputArgData := []string{
		"0x0000000000000000000000000000000000000000000000000000000000000000", // salt
		accountOwner,
		aaEntryPoint.String(),
	}
	txInputData, err := buildTxInputData(funcSignature, inputArgData)
	if err != nil {
		return err
	}

	log.Printf("deploying AA account contract")
	contract := getAASimpleAccountFactoryAddress()
	tx, err := Transact(globalClient.RpcClient, globalClient.EthClient, hexToPrivateKey(globalOptPrivateKey), &contract, big.NewInt(0), nil, txInputData)
	if err != nil {
		return err
	}
	log.Printf("deploying AA account contract, tx %s", tx)

	return nil
}

func init() {
	aaSimpleAccountCmd.AddCommand(aaDeployCmd)
}
