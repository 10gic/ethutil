package cmd

import (
	"log"
	"math/big"

	"github.com/spf13/cobra"
)

var dropTxCmd = &cobra.Command{
	Use:   "drop-tx",
	Short: "Drop pending tx for address",
	Run: func(cmd *cobra.Command, args []string) {
		if globalOptPrivateKey == "" {
			log.Fatalf("--private-key is required for drop-tx command")
		}
		log.Printf("Current chain is %v", globalOptChain)

		InitGlobalClient(globalOptNodeUrl)

		// send 0 eth to itself
		// see https://medium.com/fidcom/how-to-remove-ethereum-pending-transaction-f876f211896d

		gasPrice, err := getGasPrice(globalClient.EthClient)
		checkErr(err)

		gasPrice.Add(gasPrice, big.NewInt(10*1000000000)) // plus 10 gwei
		log.Printf("gas price change to %v wei", gasPrice)

		addr := extractAddressFromPrivateKey(hexToPrivateKey(globalOptPrivateKey)).String()
		if tx, err := TransferHelper(globalClient.RpcClient, globalClient.EthClient, globalOptPrivateKey, addr, big.NewInt(0), gasPrice, nil); err != nil {
			log.Fatalf("transfer 0 wei to self fail: %v", err)
		} else {
			log.Printf("transfer 0 wei to self finished, tx = %v", tx)
		}
	},
}
