package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

// broadcastTxCmd represents the broadcastTx command
var broadcastTxCmd = &cobra.Command{
	Use:   "broadcast-tx <signed-raw-tx>",
	Short: "Broadcast tx by rpc eth_sendRawTransaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		InitGlobalClient(globalOptNodeUrl)

		var signedTxHexStr = args[0]
		rpcReturnTx, err := SendRawTransaction(globalClient.RpcClient, signedTxHexStr)
		if err != nil {
			log.Fatalf("SendRawTransaction fail: %s", err)
		}

		log.Printf("tx %s is broadcasted", rpcReturnTx)

		for k, v := range nodeUrlMap {
			if v == globalOptNodeUrl {
				log.Printf(nodeTxExplorerUrlMap[k] + rpcReturnTx.String())
				break
			}
		}
	},
}
