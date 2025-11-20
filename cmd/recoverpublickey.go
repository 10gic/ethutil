package cmd

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

// recoverPublicKeyCmd represents the recover-public-key command
var recoverPublicKeyCmd = &cobra.Command{
	Use:   "recover-public-key <message-hash> <signature>",
	Short: "Recover public key and address from message hash and signature",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		messageHashHex := args[0]
		signatureHex := args[1]

		// Parse message hash (must be 32 bytes)
		// Support both with and without 0x prefix
		messageHash, err := hex.DecodeString(remove0xPrefix(messageHashHex))
		checkErr(err)
		if len(messageHash) != 32 {
			log.Fatalf("message hash must be 32 bytes, got %d bytes", len(messageHash))
		}

		// Parse signature (must be 65 bytes)
		// Support both with and without 0x prefix
		signature, err := hex.DecodeString(remove0xPrefix(signatureHex))
		checkErr(err)
		if len(signature) != 65 {
			log.Fatalf("signature must be 65 bytes, got %d bytes", len(signature))
		}

		// Extract r, s, v from signature
		// Signature format: 65 bytes, first 64 bytes are r||s, last byte is recovery ID
		r := new(big.Int).SetBytes(signature[0:32])
		s := new(big.Int).SetBytes(signature[32:64])
		recoveryID := signature[64]

		// Convert recovery ID to v
		// Recovery ID can be 0/1 (EIP-2718) or 27/28 (pre-EIP-155)
		var v *big.Int
		if recoveryID == 0 || recoveryID == 1 || recoveryID == 27 || recoveryID == 28 {
			v = big.NewInt(int64(recoveryID))
		} else {
			log.Fatalf("invalid recovery ID: %d (must be 0, 1, 27, or 28)", recoveryID)
		}

		// Recover public key
		pubkeyBytes, err := RecoverPubkey(v, r, s, messageHash)
		checkErr(err)

		// Convert uncompressed public key to ecdsa.PublicKey
		pubkey, err := crypto.UnmarshalPubkey(pubkeyBytes)
		checkErr(err)

		// Extract address from ecdsa.PublicKey
		addr := crypto.PubkeyToAddress(*pubkey)

		// Output
		fmt.Printf("uncompressed public key (hex) = %s\n", hexutil.Encode(pubkeyBytes))
		fmt.Printf("address = %s\n", addr.String())
	},
}
