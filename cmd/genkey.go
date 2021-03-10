package cmd

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var genkeyNumOpt int

func init() {
	genkeyCmd.Flags().IntVarP(&genkeyNumOpt, "number", "n", 1, "number of private key you want to generate, must greater than 1")
}

var genkeyCmd = &cobra.Command{
	Use:   "gen-private-key",
	Short: "Generate eth private key",
	Run: func(cmd *cobra.Command, args []string) {
		if genkeyNumOpt <= 0 {
			_ = cmd.Help()
			os.Exit(1)
		}

		for i := 1; i <= genkeyNumOpt; i++ {
			privateKey, err := crypto.GenerateKey()
			if err != nil {
				log.Fatal(err)
			}

			addr := extractAddressFromPrivateKey(privateKey).String()

			privateHexStr := hexutil.Encode(crypto.FromECDSA(privateKey))

			if terseOutputOpt {
				fmt.Printf("%v %v\n", privateHexStr, addr)
			} else {
				fmt.Printf("private key %v, addr %v\n", privateHexStr, addr)
			}
		}
	},
}
