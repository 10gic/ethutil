/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"log"

	"github.com/spf13/cobra"
)

// eip7702SignAuthTupleCmd represents the eip7702SignAuthTuple command
var eip7702SignAuthTupleCmd = &cobra.Command{
	Use:   "eip7702-sign-auth-tuple <chain-id> <delegate-to> <nonce>",
	Short: "Sign EIP-7702 authorization tuple, see EIP-7702.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 3 {
			return fmt.Errorf("requires <chain-id> <delegate-to> <nonce>")
		}
		if !isValidEthAddress(args[1]) {
			return fmt.Errorf("%v is not a valid eth address", args[1])
		}

		if globalOptPrivateKey == "" {
			log.Fatalf("--private-key is required for this command")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		chainId, err := ParseBigInt(args[0])
		checkErr(err)

		var delegateTo = common.HexToAddress(args[1])

		nonce, err := ParseBigInt(args[2])
		checkErr(err)

		auth := types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(chainId),
			Address: delegateTo,
			Nonce:   nonce.Uint64(),
		}

		var preHash = prefixedRlpHash(0x05, []any{
			auth.ChainID,
			auth.Address,
			nonce,
		})

		fmt.Printf("auth pre hash = %x\n", preHash[:])

		var privateKey = hexToPrivateKey(globalOptPrivateKey)
		sig, err := crypto.Sign(preHash[:], privateKey)
		checkErr(err)

		authority := extractAddressFromPrivateKey(privateKey)
		fmt.Printf("authority (i.e. signer) = %s\n", authority.Hex())

		fmt.Printf("sig hex = %x\n", sig)
		fmt.Printf("sig r = %x\n", sig[0:32])
		fmt.Printf("sig s = %x\n", sig[32:64])
		fmt.Printf("sig v = %d\n", sig[64])
	},
}
