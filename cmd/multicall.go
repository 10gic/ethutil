package cmd

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"strings"
)

const MulticallContractAddr = "0xcA11bde05977b3631167028862bE2a173976CA11" // See https://github.com/mds1/multicall
const MulticallFuncSignGetEthBalance = "4d2301cc" // 4 bytes func signature of `getEthBalance(address)`
const MulticallFuncSignAggregate = "252dba42" // 4 bytes func signature of `aggregate((address,bytes)[])`

func isMulticallDeployed(client *ethclient.Client) bool {
	deployed, err := isContractAddress(client, common.HexToAddress(MulticallContractAddr))
	if err != nil {
		return false
	}

	return deployed
}

func queryEthBalancesByMulticall(addresses []string) ([]*big.Int, error) {
	contractAddress := common.HexToAddress(MulticallContractAddr)

	funcSignGetEthBalance, err := hex.DecodeString(MulticallFuncSignGetEthBalance)
	if err != nil {
		return nil, err
	}

	funcSignAggregate, err := hex.DecodeString(MulticallFuncSignAggregate)
	if err != nil {
		return nil, err
	}

	// build tx input data
	var callDataForAggregate []string
	for _, address := range addresses {
		parameter, err := encodeParameters([]string{"address"}, []string{address})
		if err != nil {
			return nil, err
		}

		// prepare call data for function getEthBalance(address)
		callDataForGetEthBalance := append(funcSignGetEthBalance, parameter...)

		callDataForAggregate = append(callDataForAggregate,
			"("+contractAddress.String()+",0x"+hex.EncodeToString(callDataForGetEthBalance)+")")
	}
	txInputData, err := encodeParameters([]string{"(address,bytes)[]"},
		[]string{"[" + strings.Join(callDataForAggregate, ",") + "]"})
	// fmt.Printf("txInputData = %x\n", txInputData)

	// call multicall contract function aggregate:
	// function aggregate((address,bytes)[]) public payable returns (uint256 blockNumber, bytes[] memory returnData)
	output, err := Call(globalClient.EthClient, contractAddress, append(funcSignAggregate, txInputData...))
	checkErr(err)
	// fmt.Printf("output = %x\n", output)
	//
	// Here is an output example of query balances for two addresses:
	// 0000000000000000000000000000000000000000000000000000000000865cd5000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000074bfbaccf3734130a00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000020116afda686cbe3c
	//[0]:  0x0000000000000000000000000000000000000000000000000000000000865cd5  // blockNumber
	//[1]:  0x0000000000000000000000000000000000000000000000000000000000000040  // offset of returnData
	//[2]:  0x0000000000000000000000000000000000000000000000000000000000000002  // length of returnData
	//[3]:  0x0000000000000000000000000000000000000000000000000000000000000040  // offset of first element
	//[4]:  0x0000000000000000000000000000000000000000000000000000000000000080  // offset of second element
	//[5]:  0x0000000000000000000000000000000000000000000000000000000000000020  // length of first element
	//[6]:  0x0000000000000000000000000000000000000000000000074bfbaccf3734130a  // first element (first address balance)   <- result 0
	//[7]:  0x0000000000000000000000000000000000000000000000000000000000000020  // length of second element
	//[8]:  0x0000000000000000000000000000000000000000000000020116afda686cbe3c  // second element (second address balance)  <- result 1

	balancePart := output[len(output)-(64*len(addresses)):] // take last 64*len(addresses) bytes
	// fmt.Printf("balancePart = %x\n", balancePart)

	rv := make([]*big.Int, len(addresses))
	for index := range addresses {
		result := balancePart[64*index+32 : 64*index+64]
		balance := new(big.Int)
		balance.SetString(hex.EncodeToString(result), 16)
		rv[index] = balance
	}

	return rv, nil
}
