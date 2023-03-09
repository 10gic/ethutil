package cmd

import (
	"encoding/hex"
	"github.com/spf13/cobra"
	"log"
	"math/big"
	"os"
)

var deployErc20Cmd = &cobra.Command{
	Use:   "deploy-erc20 [total-supply [name [symbol [decimals]]]]",
	Short: "Deploy an ERC20 token",
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("Current network is %v", globalOptNode)

		// Some default value
		var totalSupply = "10000000000000000000000000"
		var name = "A Simple ERC20"
		var symbol = "TEST"
		var decimals = "18"

		switch len(args) {
		case 0:
			break
		case 1:
			totalSupply = args[0]
		case 2:
			totalSupply = args[0]
			name = args[1]
		case 3:
			totalSupply = args[0]
			name = args[1]
			symbol = args[2]
		case 4:
			totalSupply = args[0]
			name = args[1]
			symbol = args[2]
			decimals = args[3]
		default:
			_ = cmd.Help()
			os.Exit(1)
		}

		InitGlobalClient(globalOptNodeUrl)

		var funcSignature = "constructor(uint256, string, string, uint8)"
		var inputArgData = []string{totalSupply, name, symbol, decimals}

		var bytecodeHex = "60806040523480156200001157600080fd5b50604051620013df380380620013df833981810160405281019062000037919062000235565b83600081905550826001908051906020019062000056929190620000d9565b5081600290805190602001906200006f929190620000d9565b5080600360006101000a81548160ff021916908360ff16021790555083600460003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555050505050620004b4565b828054620000e79062000391565b90600052602060002090601f0160209004810192826200010b576000855562000157565b82601f106200012657805160ff191683800117855562000157565b8280016001018555821562000157579182015b828111156200015657825182559160200191906001019062000139565b5b5090506200016691906200016a565b5090565b5b80821115620001855760008160009055506001016200016b565b5090565b6000620001a06200019a846200030e565b620002e5565b905082815260208101848484011115620001bf57620001be62000460565b5b620001cc8482856200035b565b509392505050565b600082601f830112620001ec57620001eb6200045b565b5b8151620001fe84826020860162000189565b91505092915050565b600081519050620002188162000480565b92915050565b6000815190506200022f816200049a565b92915050565b600080600080608085870312156200025257620002516200046a565b5b6000620002628782880162000207565b945050602085015167ffffffffffffffff81111562000286576200028562000465565b5b6200029487828801620001d4565b935050604085015167ffffffffffffffff811115620002b857620002b762000465565b5b620002c687828801620001d4565b9250506060620002d9878288016200021e565b91505092959194509250565b6000620002f162000304565b9050620002ff8282620003c7565b919050565b6000604051905090565b600067ffffffffffffffff8211156200032c576200032b6200042c565b5b62000337826200046f565b9050602081019050919050565b6000819050919050565b600060ff82169050919050565b60005b838110156200037b5780820151818401526020810190506200035e565b838111156200038b576000848401525b50505050565b60006002820490506001821680620003aa57607f821691505b60208210811415620003c157620003c0620003fd565b5b50919050565b620003d2826200046f565b810181811067ffffffffffffffff82111715620003f457620003f36200042c565b5b80604052505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600080fd5b600080fd5b600080fd5b600080fd5b6000601f19601f8301169050919050565b6200048b8162000344565b81146200049757600080fd5b50565b620004a5816200034e565b8114620004b157600080fd5b50565b610f1b80620004c46000396000f3fe608060405234801561001057600080fd5b50600436106100935760003560e01c8063313ce56711610066578063313ce5671461013457806370a082311461015257806395d89b4114610182578063a9059cbb146101a0578063dd62ed3e146101d057610093565b806306fdde0314610098578063095ea7b3146100b657806318160ddd146100e657806323b872dd14610104575b600080fd5b6100a0610200565b6040516100ad9190610c5c565b60405180910390f35b6100d060048036038101906100cb9190610b9b565b61028e565b6040516100dd9190610c41565b60405180910390f35b6100ee610380565b6040516100fb9190610c7e565b60405180910390f35b61011e60048036038101906101199190610b48565b610386565b60405161012b9190610c41565b60405180910390f35b61013c610706565b6040516101499190610c99565b60405180910390f35b61016c60048036038101906101679190610adb565b610719565b6040516101799190610c7e565b60405180910390f35b61018a610762565b6040516101979190610c5c565b60405180910390f35b6101ba60048036038101906101b59190610b9b565b6107f0565b6040516101c79190610c41565b60405180910390f35b6101ea60048036038101906101e59190610b08565b6109d7565b6040516101f79190610c7e565b60405180910390f35b6001805461020d90610de2565b80601f016020809104026020016040519081016040528092919081815260200182805461023990610de2565b80156102865780601f1061025b57610100808354040283529160200191610286565b820191906000526020600020905b81548152906001019060200180831161026957829003601f168201915b505050505081565b600081600560003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b9258460405161036e9190610c7e565b60405180910390a36001905092915050565b60005481565b6000600460008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020548211156103d457600080fd5b600560008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205482111561045d57600080fd5b6104af82600460008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610a5e90919063ffffffff16565b600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555061058182600560008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610a5e90919063ffffffff16565b600560008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555061065382600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610a8590919063ffffffff16565b600460008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040516106f39190610c7e565b60405180910390a3600190509392505050565b600360009054906101000a900460ff1681565b6000600460008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b6002805461076f90610de2565b80601f016020809104026020016040519081016040528092919081815260200182805461079b90610de2565b80156107e85780601f106107bd576101008083540402835291602001916107e8565b820191906000526020600020905b8154815290600101906020018083116107cb57829003601f168201915b505050505081565b6000600460003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205482111561083e57600080fd5b61089082600460003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610a5e90919063ffffffff16565b600460003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555061092582600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610a8590919063ffffffff16565b600460008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040516109c59190610c7e565b60405180910390a36001905092915050565b6000600560008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905092915050565b600082821115610a7157610a70610e14565b5b8183610a7d9190610d26565b905092915050565b6000808284610a949190610cd0565b905083811015610aa757610aa6610e14565b5b8091505092915050565b600081359050610ac081610eb7565b92915050565b600081359050610ad581610ece565b92915050565b600060208284031215610af157610af0610ea1565b5b6000610aff84828501610ab1565b91505092915050565b60008060408385031215610b1f57610b1e610ea1565b5b6000610b2d85828601610ab1565b9250506020610b3e85828601610ab1565b9150509250929050565b600080600060608486031215610b6157610b60610ea1565b5b6000610b6f86828701610ab1565b9350506020610b8086828701610ab1565b9250506040610b9186828701610ac6565b9150509250925092565b60008060408385031215610bb257610bb1610ea1565b5b6000610bc085828601610ab1565b9250506020610bd185828601610ac6565b9150509250929050565b610be481610d6c565b82525050565b6000610bf582610cb4565b610bff8185610cbf565b9350610c0f818560208601610daf565b610c1881610ea6565b840191505092915050565b610c2c81610d98565b82525050565b610c3b81610da2565b82525050565b6000602082019050610c566000830184610bdb565b92915050565b60006020820190508181036000830152610c768184610bea565b905092915050565b6000602082019050610c936000830184610c23565b92915050565b6000602082019050610cae6000830184610c32565b92915050565b600081519050919050565b600082825260208201905092915050565b6000610cdb82610d98565b9150610ce683610d98565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff03821115610d1b57610d1a610e43565b5b828201905092915050565b6000610d3182610d98565b9150610d3c83610d98565b925082821015610d4f57610d4e610e43565b5b828203905092915050565b6000610d6582610d78565b9050919050565b60008115159050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000819050919050565b600060ff82169050919050565b60005b83811015610dcd578082015181840152602081019050610db2565b83811115610ddc576000848401525b50505050565b60006002820490506001821680610dfa57607f821691505b60208210811415610e0e57610e0d610e72565b5b50919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052600160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b600080fd5b6000601f19601f8301169050919050565b610ec081610d5a565b8114610ecb57600080fd5b50565b610ed781610d98565b8114610ee257600080fd5b5056fea2646970667358221220cdd2a727f1b9a1481bdaaceeff8008d3588177639091ba216243d9402fe89d8664736f6c63430008060033"
		bytecodeByteArray, err := hex.DecodeString(bytecodeHex)
		if err != nil {
			log.Fatal("--bin-file invalid")
		}

		txData, err := buildTxDataForContractDeploy(funcSignature, inputArgData, bytecodeByteArray)
		checkErr(err)
		// log.Printf("txData=%s", hex.Dump(txData))

		if globalOptPrivateKey == "" {
			log.Fatalf("--private-key is required for deploy command")
		}

		tx, err := Transact(globalClient.RpcClient, globalClient.EthClient, buildPrivateKeyFromHex(globalOptPrivateKey), nil, big.NewInt(0), nil, txData)
		checkErr(err)

		log.Printf("transaction %s finished", tx)
	},
}
