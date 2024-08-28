package cmd

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"io"
	"os"
)

var keccakBinaryModeOpt bool

func init() {
	keccakCmd.Flags().BoolVarP(&keccakBinaryModeOpt, "binary", "b", false, "read in binary mode, default is text mode")
}

var keccakCmd = &cobra.Command{
	Use:   "keccak [flags] <file> ...",
	Short: "Compute keccak hash, read data from file or stdin (file name -)",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// if no file specified, read from stdin
			computeAndOutputKeccak("-")
			return
		}

		for _, fil := range args {
			computeAndOutputKeccak(fil)
		}
	},
}

func computeAndOutputKeccak(f string) {
	var fileContent []byte
	var err error

	if f == "-" {
		fileContent, err = io.ReadAll(os.Stdin)
		checkErr(err)
	} else {
		fileContent, err = os.ReadFile(f)
		checkErr(err)
	}

	// binary mode
	if keccakBinaryModeOpt {
		fileContent, err = hex.DecodeString(string(fileContent))
		checkErr(err)
	}

	fmt.Printf("%v  %v\n", hexutil.Encode(crypto.Keccak256(fileContent))[2:], f)
}
