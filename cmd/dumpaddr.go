package cmd

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

var dumpAddrPrivateKeys []string

func init() {
	dumpAddrCmd.Flags().StringSliceVarP(&dumpAddrPrivateKeys, "private-key", "k", []string{}, "the private key your want to dump address, multiple private keys can separate by comma, the option can be also specified multiple times")
	dumpAddrCmd.MarkFlagRequired("private-key")
}

var dumpAddrCmd = &cobra.Command{
	Use:   "dump-address",
	Aliases: []string{"dump-addr"},
	Short: "Dump address from private key",
	Run: func(cmd *cobra.Command, args []string) {

		for _, dumpAddrPrivateKeyHexStr := range dumpAddrPrivateKeys {
			privateKey := buildPrivateKeyFromHex(dumpAddrPrivateKeyHexStr)
			privateHexStr := hexutil.Encode(crypto.FromECDSA(privateKey))
			addr := extractAddressFromPrivateKey(privateKey).String()

			if terseOutputOpt {
				fmt.Printf("%v %v\n", privateHexStr, addr)
			} else {
				fmt.Printf("private key %v, addr %v\n", privateHexStr, addr)
			}
		}
	},
}
