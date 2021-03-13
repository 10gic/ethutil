package cmd

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"github.com/tyler-smith/go-bip39"
)

var dumpAddrCmd = &cobra.Command{
	Use:     "dump-address private-key-or-mnemonics private-key-or-mnemonics ...",
	Aliases: []string{"dump-addr"},
	Short:   "Dump address from private key or mnemonic",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires private-key-or-mnemonics")
		}
		for _, arg := range args {
			if isValidHexString(arg) {
				continue
			}
			if bip39.IsMnemonicValid(arg) {
				continue
			}
			return fmt.Errorf("invalid private-key-or-mnemonics: %v", arg)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		privateKeyOrMnemonics := args

		for _, dumpAddrPrivateKeyOrMnemonic := range privateKeyOrMnemonics {

			var privateKey *ecdsa.PrivateKey
			var err error
			if isValidHexString(dumpAddrPrivateKeyOrMnemonic) {
				privateKey = buildPrivateKeyFromHex(dumpAddrPrivateKeyOrMnemonic)
			} else { // mnemonic
				privateKey, err = hdWallet(dumpAddrPrivateKeyOrMnemonic)
				checkErr(err)
			}

			privateHexStr := hexutil.Encode(crypto.FromECDSA(privateKey))
			addr := extractAddressFromPrivateKey(privateKey).String()
			if globalOptTerseOutput {
				fmt.Printf("%v %v\n", privateHexStr, addr)
			} else {
				fmt.Printf("private key %v, addr %v\n", privateHexStr, addr)
			}
		}
	},
}

func hdWallet(mnemonic string) (*ecdsa.PrivateKey, error) {
	// Generate a Bip32 HD wallet for the mnemonic and a user supplied password
	seed := bip39.NewSeed(mnemonic, "")
	// Generate a new master node using the seed.
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}
	// This gives the path: m/44H
	acc44H, err := masterKey.Child(hdkeychain.HardenedKeyStart + 44)
	if err != nil {
		return nil, err
	}
	// This gives the path: m/44H/60H
	acc44H60H, err := acc44H.Child(hdkeychain.HardenedKeyStart + 60)
	if err != nil {
		return nil, err
	}
	// This gives the path: m/44H/60H/0H
	acc44H60H0H, err := acc44H60H.Child(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, err
	}
	// This gives the path: m/44H/60H/0H/0
	acc44H60H0H0, err := acc44H60H0H.Child(0)
	if err != nil {
		return nil, err
	}
	// This gives the path: m/44H/60H/0H/0/0
	acc44H60H0H00, err := acc44H60H0H0.Child(0)
	if err != nil {
		return nil, err
	}
	btcecPrivKey, err := acc44H60H0H00.ECPrivKey()
	if err != nil {
		return nil, err
	}
	privateKey := btcecPrivKey.ToECDSA()
	return privateKey, nil
}
