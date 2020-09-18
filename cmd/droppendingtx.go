package cmd

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"log"
	"math/big"
)

var dropPendingTxCmd = &cobra.Command{
	Use:   "drop-pending-tx",
	Short: "Drop pending tx for address",
	Run: func(cmd *cobra.Command, args []string) {
		if privateKeyOpt == "" {
			log.Fatalf("--private-key is required for drop-pending-tx command")
		}

		// send 0 eth to itself
		// see https://medium.com/fidcom/how-to-remove-ethereum-pending-transaction-f876f211896d

		client, err := ethclient.Dial(nodeUrlOpt)
		checkErr(err)

		gasPrice, err := getGasPrice(client)
		checkErr(err)

		gasPrice.Add(gasPrice, big.NewInt(10 * 1000000000)) // plus 10 gwei
		log.Printf("gas price change to %v wei", gasPrice)

		addr := extractAddressFromPrivateKey(buildPrivateKeyFromHex(privateKeyOpt)).String()
		if tx, err := TransferHelper(client, privateKeyOpt, addr, big.NewInt(0), gasPrice, nil); err != nil {
			log.Fatalf("transfer 0 wei to self fail: %v", err)
		} else {
			log.Printf("transfer 0 wei to self finished, tx = %v", tx)
		}
	},
}
