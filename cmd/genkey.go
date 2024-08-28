package cmd

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"github.com/tyler-smith/go-bip39"
)

var genkeyNumOpt int
var genkeyMnemonicLengthOpt int

func init() {
	genkeyCmd.Flags().IntVarP(&genkeyNumOpt, "number", "n", 1, "number of private key you want to generate, must greater than 1")
	genkeyCmd.Flags().IntVarP(&genkeyMnemonicLengthOpt, "mnemonic-len", "", 12, "number of mnemonic words, can be 12/15/18/21/24")
}

var genkeyCmd = &cobra.Command{
	Use:     "gen-key",
	Aliases: []string{"gen-private-key"},
	Short:   "Generate eth mnemonic words, private key, and its address",
	Run: func(cmd *cobra.Command, args []string) {
		if genkeyNumOpt <= 0 {
			_ = cmd.Help()
			os.Exit(1)
		}

		var entropyBitSize int
		if genkeyMnemonicLengthOpt == 12 {
			entropyBitSize = 128
		} else if genkeyMnemonicLengthOpt == 15 {
			entropyBitSize = 160
		} else if genkeyMnemonicLengthOpt == 18 {
			entropyBitSize = 192
		} else if genkeyMnemonicLengthOpt == 21 {
			entropyBitSize = 224
		} else if genkeyMnemonicLengthOpt == 24 {
			entropyBitSize = 256
		} else {
			panic(fmt.Sprintf("invalid mnemonic-len %v", genkeyMnemonicLengthOpt))
		}

		for i := 1; i <= genkeyNumOpt; i++ {
			entropy, err := bip39.NewEntropy(entropyBitSize)
			checkErr(err)
			mnemonic, err := bip39.NewMnemonic(entropy)
			checkErr(err)
			privateKey, err := MnemonicToPrivateKey(mnemonic, ethBip44Path)
			checkErr(err)

			addr := extractAddressFromPrivateKey(privateKey).String()

			privateHexStr := hexutil.Encode(crypto.FromECDSA(privateKey))

			publicKeyHexStr := hexutil.Encode(crypto.FromECDSAPub(&privateKey.PublicKey))

			if globalOptTerseOutput {
				fmt.Printf("%v %v\n", privateHexStr, addr)
			} else {
				fmt.Printf("mnemonic: %v\nprivate key: %v\npublic key: %v\naddr: %v\n", mnemonic, privateHexStr, publicKeyHexStr, addr)
			}
		}
	},
}
