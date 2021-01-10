package cmd

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/spf13/cobra"
)

var rawTxHexData string

func init() {
	decodeRawTxCmd.Flags().StringVarP(&rawTxHexData, "hex-data", "", "", "the hex data (leading 0x is optional) of raw tx")
	decodeRawTxCmd.MarkFlagRequired("hex-data")
}

var decodeRawTxCmd = &cobra.Command{
	Use:   "decode-raw-tx",
	Short: "Decode raw transaction",
	Run: func(cmd *cobra.Command, args []string) {
		if len(rawTxHexData) == 0 {
			cmd.Help()
			os.Exit(1)
		}
		if !isValidHexString(rawTxHexData) {
			log.Fatalf("--hex-data must hex string")
		}

		if strings.HasPrefix(rawTxHexData, "0x") {
			rawTxHexData = rawTxHexData[2:] // remove leading 0x
		}

		var tx *types.Transaction
		rawTxBytes, _ := hex.DecodeString(rawTxHexData)
		rlp.DecodeBytes(rawTxBytes, &tx)

		fmt.Printf("basic info (see bip155):\n")
		fmt.Printf("nonce = %d\n", tx.Nonce())
		fmt.Printf("gas price = %s, i.e. %s Gwei\n", tx.GasPrice().String(), wei2Other(bigInt2Decimal(tx.GasPrice()), unitGwei).String())
		fmt.Printf("gas limit = %d\n", tx.Gas())
		fmt.Printf("to = %s\n", tx.To().String())
		fmt.Printf("value = %s, i.e. %s Ether\n", tx.Value().String(), wei2Other(bigInt2Decimal(tx.Value()), unitEther).String())
		fmt.Printf("input data (hex) = %x\n", tx.Data())

		if tx.ChainId().Int64() > 0 { // chain id is not available before bip155
			fmt.Printf("chain id = %s\n", tx.ChainId().String())
		}

		v, r, s := tx.RawSignatureValues()
		fmt.Printf("v = %s\n", v.String())
		fmt.Printf("r (hex) = %x\n", r)
		fmt.Printf("s (hex) = %x\n", s)

		fmt.Printf("\nderived info:\n")
		fmt.Printf("txid (hex) = %x\n", tx.Hash().Bytes())
		var chainId = tx.ChainId()

		// build msg (hash of data) before sign
		singer := types.NewEIP155Signer(chainId)
		hash := singer.Hash(tx)
		fmt.Printf("hash before ecdsa sign (hex) = %x\n", hash.Bytes())

		var recoveryId = getRecoveryId(v)
		fmt.Printf("ecdsa recovery id = %d\n", recoveryId)

		pubkeyBytes, err := RecoverPubkey(v, r, s, hash.Bytes())
		checkErr(err)
		fmt.Printf("uncompressed 65 bytes public key of sender (hex) = %x\n", pubkeyBytes)

		// convert uncompressed public key to ecdsa.PublicKey
		pubkey, err := crypto.UnmarshalPubkey(pubkeyBytes)
		checkErr(err)

		// extract address from ecdsa.PublicKey
		addr := crypto.PubkeyToAddress(*pubkey)
		fmt.Printf("address of sender = %s\n", addr.Hex())
	},
}
