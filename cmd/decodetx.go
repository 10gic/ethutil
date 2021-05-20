package cmd

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/spf13/cobra"
)

var decodeTxCmd = &cobra.Command{
	Use:   "decode-tx tx-data",
	Short: "Decode raw transaction",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires tx-data")
		}
		if len(args) > 1 {
			return fmt.Errorf("multiple tx-data is not supported")
		}

		if !isValidHexString(args[0]) {
			return fmt.Errorf("tx-data must hex string")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		rawTxHexData := args[0]

		if strings.HasPrefix(rawTxHexData, "0x") {
			rawTxHexData = rawTxHexData[2:] // remove leading 0x
		}

		var tx *types.Transaction
		rawTxBytes, _ := hex.DecodeString(rawTxHexData)
		err := rlp.DecodeBytes(rawTxBytes, &tx)
		if err != nil {
			panic("rlp decode failed, may not a valid eth raw transaction")
		}

		fmt.Printf("basic info (see eip155):\n")
		fmt.Printf("nonce = %d\n", tx.Nonce())
		fmt.Printf("gas price = %s, i.e. %s Gwei\n", tx.GasPrice().String(), wei2Other(bigInt2Decimal(tx.GasPrice()), unitGwei).String())
		fmt.Printf("gas limit = %d\n", tx.Gas())
		fmt.Printf("to = %s\n", tx.To().String())
		fmt.Printf("value = %s, i.e. %s Ether\n", tx.Value().String(), wei2Other(bigInt2Decimal(tx.Value()), unitEther).String())
		fmt.Printf("input data (hex) = %x\n", tx.Data())

		if tx.ChainId().Int64() > 0 { // chain id is not available before eip155
			fmt.Printf("chain id = %s\n", tx.ChainId().String())
		}

		v, r, s := tx.RawSignatureValues()
		fmt.Printf("v = %s\n", v.String())
		fmt.Printf("r (hex) = %x\n", r)
		fmt.Printf("s (hex) = %x\n", s)

		fmt.Printf("\n")
		fmt.Printf("derived info:\n")
		fmt.Printf("txid (hex) = %x\n", tx.Hash().Bytes())
		var chainId = tx.ChainId()

		// build msg (hash of data) before sign
		singer := types.NewEIP2930Signer(chainId)
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
