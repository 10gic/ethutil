package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
)

var fourByteCmd = &cobra.Command{
	Use:   "4byte <func-selector>",
	Short: "Get the function signatures for the given selector from https://openchain.xyz/signatures or https://www.4byte.directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var funcHash = args[0]
		if !isValidHexString(funcHash) {
			log.Fatalf("func-selector must be hex string")
		}
		// Add 0x prefix if not present, for example, change 8c905368 to 0x8c905368
		if len(funcHash) == 8 && funcHash[0:2] != "0x" {
			funcHash = "0x" + funcHash
		}

		funcSig, err := GetFuncSig(funcHash)
		if err != nil {
			log.Printf("getFuncSig failed %v", err)
		}
		for _, data := range funcSig {
			fmt.Printf("%s\n", data)
		}
		if len(funcSig) == 0 {
			fmt.Printf("Not found\n")
		}
	},
}
