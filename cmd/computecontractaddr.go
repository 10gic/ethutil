package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
)

var computeContractAddrSalt string
var computeContractAddrInitCode string

func init() {
	computeContractAddrCmd.Flags().StringVarP(&computeContractAddrSalt, "salt", "", "", "salt, for CREATE2")
	computeContractAddrCmd.Flags().StringVarP(&computeContractAddrInitCode, "init-code", "", "", "init code, for CREATE2")
}

func validationComputeContractAddrCmdOpts() bool {
	if len(computeContractAddrSalt) == 0 && len(computeContractAddrInitCode) != 0 {
		log.Fatalf("--salt must also provided")
		return false
	}
	if len(computeContractAddrSalt) != 0 && len(computeContractAddrInitCode) == 0 {
		log.Fatalf("--init-code must also provided")
		return false
	}

	if !isValidHexString(computeContractAddrSalt) {
		log.Fatalf("--salt must hex string")
		return false
	}

	if !isValidHexString(computeContractAddrInitCode) {
		log.Fatalf("--init-code must hex string")
		return false
	}

	return true
}

var computeContractAddrCmd = &cobra.Command{
	Use:   "compute-contract-addr <deployer-address>",
	Short: "Compute contract address before deployment",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires the address of deployer")
		}
		if len(args) > 1 {
			return fmt.Errorf("you can not specify multiple deployers")
		}
		if !isValidEthAddress(args[0]) {
			return fmt.Errorf("%v is not a valid eth address", args[0])
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !validationComputeContractAddrCmdOpts() {
			_ = cmd.Help()
			os.Exit(1)
		}

		deployerAddr := args[0]

		if len(computeContractAddrSalt) == 0 {
			var nonce uint64
			if globalOptNonce < 0 {
				// get nonce online
				client, err := ethclient.Dial(globalOptNodeUrl)
				checkErr(err)

				nonce, err = client.PendingNonceAt(context.TODO(), common.HexToAddress(deployerAddr))
				checkErr(err)
			} else {
				nonce = uint64(globalOptNonce)
			}
			contractAddr := crypto.CreateAddress(common.HexToAddress(deployerAddr), nonce)
			fmt.Printf("deployer address %v\nnonce %v\ncontract address %v\n",
				deployerAddr,
				globalOptNonce,
				contractAddr.Hex())
		} else {
			var salt32 [32]byte
			copy(salt32[:], common.FromHex(computeContractAddrSalt))
			contractAddr := crypto.CreateAddress2(common.HexToAddress(deployerAddr), salt32, crypto.Keccak256(common.FromHex(computeContractAddrInitCode)))
			fmt.Printf("deployer address %v\nsalt %v\ninit code %v\ncontract address %v\n",
				deployerAddr,
				computeContractAddrSalt,
				computeContractAddrInitCode,
				contractAddr.Hex())
		}
	},
}
