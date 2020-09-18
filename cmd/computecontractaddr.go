package cmd

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var computeContractAddrDeployerAddr string
var computeContractAddrNonce int64
var computeContractAddrSalt string
var computeContractAddrInitCode string

func init() {
	computeContractAddrCmd.Flags().StringVarP(&computeContractAddrDeployerAddr, "deployer-addr", "", "", "the address of deployer")
	computeContractAddrCmd.Flags().Int64VarP(&computeContractAddrNonce, "nonce", "", -1, "the nonce, -1 means check online")
	computeContractAddrCmd.Flags().StringVarP(&computeContractAddrSalt, "salt", "", "", "salt, for CREATE2")
	computeContractAddrCmd.Flags().StringVarP(&computeContractAddrInitCode, "init-code", "", "", "init code, for CREATE2")

	computeContractAddrCmd.MarkFlagRequired("deployer-addr")
}

func validationComputeContractAddrCmdOpts() bool {
	// validation
	if !isValidEthAddress(computeContractAddrDeployerAddr) {
		log.Fatalf("%v is not a valid eth address", transferTargetAddr)
		return false
	}

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
	Use:   "compute-contract-addr",
	Short: "Compute contract address before deployment",
	Run: func(cmd *cobra.Command, args []string) {
		if !validationComputeContractAddrCmdOpts() {
			cmd.Help()
			os.Exit(1)
		}

		if len(computeContractAddrSalt) == 0 {
			var nonce uint64
			if computeContractAddrNonce < 0 {
				// get nonce online
				client, err := ethclient.Dial(nodeUrlOpt)
				checkErr(err)

				nonce , err = client.PendingNonceAt(context.TODO(), common.HexToAddress(computeContractAddrDeployerAddr))
				checkErr(err)
			} else {
				nonce = uint64(computeContractAddrNonce)
			}
			contractAddr := crypto.CreateAddress(common.HexToAddress(computeContractAddrDeployerAddr), nonce)
			fmt.Printf("deployer address %v\nnonce %v\ncontract address %v\n",
				computeContractAddrDeployerAddr,
				computeContractAddrNonce,
				contractAddr.Hex())
		} else {
			var salt32 [32]byte
			copy(salt32[:], common.FromHex(computeContractAddrSalt))
			contractAddr := crypto.CreateAddress2(common.HexToAddress(computeContractAddrDeployerAddr), salt32, crypto.Keccak256(common.FromHex(computeContractAddrInitCode)))
			fmt.Printf("deployer address %v\nsalt %v\ninit code %v\ncontract address %v\n",
				computeContractAddrDeployerAddr,
				computeContractAddrSalt,
				computeContractAddrInitCode,
				contractAddr.Hex())
		}
	},
}
