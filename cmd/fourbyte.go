package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

var fourByteCmd = &cobra.Command{
	Use:   "4byte [func-selector]",
	Short: "Get the function signatures for the given selector from https://openchain.xyz/signatures",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var funcHash = args[0]
		if !strings.HasPrefix(funcHash, "0x") {
			log.Fatalf("func-selector must starts with 0x")
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
