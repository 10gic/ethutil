package cmd

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

var buildRawTxFrom string
var buildRawTxSignData string
var buildRawTxHexValueInWei string

func init() {
	buildRawTxCmd.Flags().StringVarP(&buildRawTxFrom, "from", "", "", "sender address. Required with --sign-data; ignored with --private-key (derived from the key).")
	buildRawTxCmd.Flags().StringVarP(&buildRawTxSignData, "sign-data", "", "", "65 bytes signature in [R || S || V] format where V is 0 or 1. Required if --private-key is not set; the two are mutually exclusive.")
	buildRawTxCmd.Flags().StringVarP(&buildRawTxHexValueInWei, "hex-value-in-wei", "", "", "tx value in wei, hex-encoded with 0x prefix (e.g. 0xde0b6b3a7640000 for 1 ether). Defaults to 0 if omitted.")
}

// buildRawTxCmd represents the encode-raw-tx command
var buildRawTxCmd = &cobra.Command{
	Use:   "build-raw-tx <to-address> <hex-data>",
	Short: "Build raw transaction (for eth_sendRawTransaction). Requires exactly one of --private-key or --sign-data.",
	Long: `Build raw transaction (for eth_sendRawTransaction). Requires exactly one of --private-key or --sign-data.

Arguments:
  <to-address>   recipient address (0x-prefixed).
  <hex-data>     calldata, 0x-prefixed hex. Pass 0x if no calldata is needed (e.g. a plain transfer).`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		InitGlobalClient(globalOptNodeUrl)

		if globalOptPrivateKey == "" && buildRawTxSignData == "" {
			log.Fatalf("exactly one of --private-key or --sign-data is required")
		}
		if globalOptPrivateKey != "" && buildRawTxSignData != "" {
			log.Fatalf("--private-key and --sign-data are mutually exclusive, provide only one")
		}

		var privateKey *ecdsa.PrivateKey
		var fromAddress common.Address
		if globalOptPrivateKey != "" {
			privateKey = hexToPrivateKey(globalOptPrivateKey)
			fromAddress = extractAddressFromPrivateKey(privateKey)
		} else {
			if buildRawTxFrom == "" {
				log.Fatalf("--from is required when --sign-data is used")
			}
			fromAddress = common.HexToAddress(buildRawTxFrom)
		}

		if !isValidEthAddress(args[0]) {
			log.Fatalf("invalid <to-address>: %s", args[0])
		}
		var toAddress = common.HexToAddress(args[0])

		var hexData = args[1]
		if !strings.HasPrefix(hexData, "0x") {
			log.Fatalf("<hex-data> must start with 0x, got: %s (use 0x for empty data)", hexData)
		}
		if !isValidHexString(hexData) {
			log.Fatalf("invalid <hex-data>: %s", hexData)
		}

		value := new(big.Int) // default 0
		if buildRawTxHexValueInWei != "" {
			if !strings.HasPrefix(buildRawTxHexValueInWei, "0x") {
				log.Fatalf("--hex-value-in-wei must start with 0x, got: %s", buildRawTxHexValueInWei)
			}
			parsed, ok := new(big.Int).SetString(buildRawTxHexValueInWei[2:], 16)
			if !ok {
				log.Fatalf("invalid --hex-value-in-wei: %s", buildRawTxHexValueInWei)
			}
			value = parsed
		}

		signedTx, err := BuildSignedTx(globalClient.EthClient, privateKey, &fromAddress, &toAddress, value, nil, common.FromHex(hexData), common.FromHex(buildRawTxSignData))
		checkErr(err)

		rawTx, err := GenRawTx(signedTx)
		checkErr(err)

		fmt.Printf("signed raw tx (can be used by rpc eth_sendRawTransaction) = %v\n", rawTx)
	},
}
