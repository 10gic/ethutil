package cmd

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/spf13/cobra"
	"log"
)

var getCodeCmd = &cobra.Command{
	Use:   "code contract-address",
	Short: "Get runtime bytecode of a contract on the blockchain",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires contract-address")
		}
		if len(args) > 1 {
			return fmt.Errorf("multiple contract-address is not supported")
		}

		if !isValidEthAddress(args[0]) {
			return fmt.Errorf("%v is not a valid eth address", args[0])
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		contractAddress := args[0]
		log.Printf("Current chain is %v", globalOptChain)

		InitGlobalClient(globalOptNodeUrl)

		ctx := context.Background()

		byteCode, err := globalClient.EthClient.CodeAt(ctx, common.HexToAddress(contractAddress), nil)
		checkErr(err)

		if len(byteCode) == 0 {
			log.Printf("no runtime bytecode found for %v, it is not a deployed contract", contractAddress)
			return
		}

		fmt.Printf("runtime bytecode of contract %v is %v\n", contractAddress, hexutil.Encode(byteCode))
	},
}
