package cmd

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/spf13/cobra"
	"log"
	"strings"
)

var getCodeCmd = &cobra.Command{
	Use:   "code <address>",
	Short: "Get runtime bytecode of a contract on the blockchain, or EIP-7702 EOA code.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires address")
		}
		if len(args) > 1 {
			return fmt.Errorf("multiple address is not supported")
		}

		if !isValidEthAddress(args[0]) {
			return fmt.Errorf("%v is not a valid eth address", args[0])
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		address := args[0]

		InitGlobalClient(globalOptNodeUrl)

		ctx := context.Background()

		byteCode, err := globalClient.EthClient.CodeAt(ctx, common.HexToAddress(address), nil)
		checkErr(err)

		if len(byteCode) == 0 {
			log.Printf("no runtime bytecode found for %v", address)
			return
		}

		if strings.HasPrefix(hexutil.Encode(byteCode), "0xef0100") { // See EIP-7702
			log.Printf("%s is delegate to %v", address, hexutil.Encode(byteCode[3:23]))
			fmt.Printf("the code of EOA %v is %v\n", address, hexutil.Encode(byteCode))
		} else {
			fmt.Printf("runtime bytecode of contract %v is %v\n", address, hexutil.Encode(byteCode))
		}
	},
}
