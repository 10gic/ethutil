package cmd

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

var dumpAddrCmdDerivationPath string

func init() {
	dumpAddrCmd.Flags().StringVarP(&dumpAddrCmdDerivationPath, "derivation-path", "", ethBip44Path, "the HD derivation path")
}

var dumpAddrCmd = &cobra.Command{
	Use:     "dump-address <mnemonics-or-private-key-or-public-key> <mnemonics-or-private-key-or-public-key> ...",
	Aliases: []string{"dump-addr"},
	Short:   "Dump address from mnemonics or private key or public key",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires mnemonics-or-private-key-or-public-key")
		}
		for _, arg := range args {
			if isValidHexString(arg) {
				continue
			}
			if bip39.IsMnemonicValid(arg) {
				continue
			}
			return fmt.Errorf("invalid mnemonics-or-private-key-or-public-key: %v", arg)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		privateKeyOrMnemonics := args
		var err error

		for _, dumpAddrPrivateKeyOrMnemonic := range privateKeyOrMnemonics {

			var privateKey *ecdsa.PrivateKey
			var publicKey *ecdsa.PublicKey
			if isValidHexString(dumpAddrPrivateKeyOrMnemonic) {
				var hexLen = len(remove0xPrefix(dumpAddrPrivateKeyOrMnemonic))
				if hexLen == 64 { // private key
					privateKey = hexToPrivateKey(dumpAddrPrivateKeyOrMnemonic)
					publicKey = &privateKey.PublicKey
				} else if hexLen == 66 || hexLen == 130 { // public key
					publicKey, err = hexToPublicKey(dumpAddrPrivateKeyOrMnemonic)
					checkErr(err)
				} else {
					fmt.Printf("invalid key length %v\n", hexLen)
					return
				}
			} else { // mnemonic
				privateKey, err = MnemonicToPrivateKey(dumpAddrPrivateKeyOrMnemonic, dumpAddrCmdDerivationPath)
				publicKey = &privateKey.PublicKey
				checkErr(err)
			}

			if privateKey != nil {
				privateHexStr := hexutil.Encode(crypto.FromECDSA(privateKey))
				fmt.Printf("private key: %v\n", privateHexStr)
			}

			publicKeyHexStr := hexutil.Encode(crypto.FromECDSAPub(publicKey))
			fmt.Printf("public key: %v\n", publicKeyHexStr)

			addr := crypto.PubkeyToAddress(*publicKey).String()
			fmt.Printf("addr: %v\n", addr)
		}
	},
}

func parseDerivationPath(derivationPath string) ([]uint32, error) {
	components := strings.Split(derivationPath, "/")
	if len(components) == 0 {
		return nil, errors.New("empty derivation path")
	}

	if strings.TrimSpace(components[0]) != "m" {
		return nil, errors.New("use 'm/' prefix for path")
	}

	components = components[1:]

	// All remaining components are relative, append one by one
	if len(components) == 0 {
		return nil, errors.New("empty derivation path") // Empty relative paths
	}

	var result []uint32
	for _, component := range components {
		// Ignore any user added whitespace
		component = strings.TrimSpace(component)
		var value uint32

		// Handle hardened paths
		if strings.HasSuffix(component, "'") {
			value = bip32.FirstHardenedChild
			component = strings.TrimSpace(strings.TrimSuffix(component, "'"))
		}
		// Handle the non hardened component
		bigval, ok := new(big.Int).SetString(component, 0)
		if !ok {
			return nil, fmt.Errorf("invalid component: %s", component)
		}
		max := math.MaxUint32 - value
		if bigval.Sign() < 0 || bigval.Cmp(big.NewInt(int64(max))) > 0 {
			if value == 0 {
				return nil, fmt.Errorf("component %v out of allowed range [0, %d]", bigval, max)
			}
			return nil, fmt.Errorf("component %v out of allowed hardened range [0, %d]", bigval, max)
		}
		value += uint32(bigval.Uint64())

		// Append and repeat
		result = append(result, value)
	}

	return result, nil
}
