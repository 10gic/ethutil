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

var keccakHexModeOpt bool

func init() {
	keccakCmd.Flags().BoolVarP(&keccakHexModeOpt, "hex", "", false, "read in hex string mode, default is text mode")
}

var keccakCmd = &cobra.Command{
	Use:   "keccak [flags] <data> ...",
	Short: "Compute keccak hash of data. If data is a existing file, compute the hash of the file content",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		finalData := getFinalData(args[0])
		fmt.Printf("%v\n", hexutil.Encode(crypto.Keccak256(finalData))[2:])
	},
}

func getFinalData(data string) []byte {
	var content []byte
	var err error

	if fileExists(data) {
		content, err = os.ReadFile(data)
		checkErr(err)
	} else if data == "-" {
		content, err = io.ReadAll(os.Stdin)
		checkErr(err)
	} else {
		content = []byte(data)
	}

	// hex string mode
	if keccakHexModeOpt {
		if len(content) > 0 && content[0] == '0' && content[1] == 'x' {
			content = content[2:]
		}
		content, err = hex.DecodeString(string(content))
		checkErr(err)
	}

	return content
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}
