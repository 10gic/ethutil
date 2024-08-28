package cmd

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"log"
)

var buildRawTxHexData string
var buildRawTxSignData string

func init() {
	buildRawTxCmd.Flags().StringVarP(&buildRawTxHexData, "hex-data", "", "", "the payload hex data when encoding raw tx")
	buildRawTxCmd.Flags().StringVarP(&buildRawTxSignData, "sign-data", "", "", "65 bytes signature, it needs to be in the [R || S || V] format where V is 0 or 1.")
}

// buildRawTxCmd represents the encode-raw-tx command
var buildRawTxCmd = &cobra.Command{
	Use:   "build-raw-tx <from-address> <to-address> <value-in-ether>",
	Short: "Build raw transaction, the output can be used by rpc eth_sendRawTransaction",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		InitGlobalClient(globalOptNodeUrl)

		var fromAddress = common.HexToAddress(args[0])
		var toAddress = common.HexToAddress(args[1])
		var valueInEther = decimal.RequireFromString(args[2])

		var privateKey *ecdsa.PrivateKey
		if globalOptPrivateKey != "" {
			privateKey = hexToPrivateKey(globalOptPrivateKey)
		}

		if globalOptPrivateKey == "" && buildRawTxSignData == "" {
			log.Fatalf("require --sign-data or --private-key")
		}

		signedTx, err := BuildSignedTx(globalClient.EthClient, privateKey, &fromAddress, &toAddress, valueInEther.BigInt(), nil, common.FromHex(buildRawTxHexData), common.FromHex(buildRawTxSignData))
		checkErr(err)

		rawTx, err := GenRawTx(signedTx)
		checkErr(err)

		fmt.Printf("signed raw tx (can be used by rpc eth_sendRawTransaction) = %v\n", rawTx)
	},
}
