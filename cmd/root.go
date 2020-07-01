package cmd

import (
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var (
	nodeUrlOpt     string
	nodeOpt        string
	gasPriceOpt    string
	terseOutputOpt bool
	rootCmd        = &cobra.Command{
		Use:   "ethutil",
		Short: "An Ethereum util, can transfer eth, check balance, drop pending tx, etc",
	}
)

const nodeUrlMainnet = "wss://mainnet.infura.io/ws/v3/21a9f5ba4bce425795cac796a66d7472" // please replace it
const nodeUrlKovan = "wss://kovan.infura.io/ws/v3/21a9f5ba4bce425795cac796a66d7472" // please replace it

const nodeMainnet = "mainnet"
const nodeKovan = "kovan"

// Execute cobra root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&nodeUrlOpt, "node-url", "", "", "the target connection node url, if this option specified, the --node option is ignored")
	rootCmd.PersistentFlags().StringVarP(&nodeOpt, "node", "x", "mainnet", "mainnet | kovan, the node type")
	rootCmd.PersistentFlags().StringVarP(&gasPriceOpt, "gas-price", "", "", "the gas price, unit is gwei.")
	rootCmd.PersistentFlags().BoolVarP(&terseOutputOpt, "terse", "t", false, "produce terse output")

	rootCmd.AddCommand(balanceCmd)
	rootCmd.AddCommand(transferCmd)
	rootCmd.AddCommand(dropPendingTxCmd)
	rootCmd.AddCommand(genkeyCmd)
	rootCmd.AddCommand(dumpAddrCmd)
}

func initConfig() {
	var err error
	nodeUrlOpt, err = rootCmd.Flags().GetString("node-url")
	checkErr(err)
	nodeOpt, err = rootCmd.Flags().GetString("node")
	checkErr(err)
	gasPriceOpt, err = rootCmd.Flags().GetString("gas-price")
	checkErr(err)
	terseOutputOpt, err = rootCmd.Flags().GetBool("terse")
	checkErr(err)

	// validation
	if !contains([]string{nodeMainnet, nodeKovan}, nodeOpt) {
		log.Printf("invalid option for --node: %v", nodeOpt)
		rootCmd.Help()
		os.Exit(1)
	}

	if gasPriceOpt != "" {
		if _, err = decimal.NewFromString(gasPriceOpt); err != nil {
			log.Printf("invalid option for --gas-price: %v", gasPriceOpt)
			rootCmd.Help()
			os.Exit(1)
		}
	}

	if nodeUrlOpt == "" {
		if nodeOpt == nodeMainnet {
			nodeUrlOpt = nodeUrlMainnet
		} else if nodeOpt == nodeKovan {
			nodeUrlOpt = nodeUrlKovan
		}
	}
}
