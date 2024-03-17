package cmd

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
	"math"
	"math/big"
	"strings"
)

var dumpAddrCmdDerivationPath string

func init() {
	dumpAddrCmd.Flags().StringVarP(&dumpAddrCmdDerivationPath, "derivation-path", "", ethBip44Path, "the HD derivation path")
}

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
			if isValidHexString(dumpAddrPrivateKeyOrMnemonic) {
				privateKey = buildPrivateKeyFromHex(dumpAddrPrivateKeyOrMnemonic)
			} else { // mnemonic
				var err error
				privateKey, err = MnemonicToPrivateKey(dumpAddrPrivateKeyOrMnemonic, dumpAddrCmdDerivationPath)
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
