package cmd

import (
	"log"
	"os"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

var (
	nodeUrlOpt     string
	nodeOpt        string
	gasPriceOpt    string
	gasLimitOpt    uint64
	privateKeyOpt  string
	terseOutputOpt bool
	rootCmd        = &cobra.Command{
		Use:   "ethutil",
		Short: "An Ethereum util, can transfer eth, check balance, drop pending tx, etc",
	}
)

const nodeMainnet = "mainnet"
const nodeRopsten = "ropsten"
const nodeKovan = "kovan"
const nodeRinkeby = "rinkeby"
const nodeGoerli = "goerli"
const nodeSokol = "sokol"
const nodeHeco = "heco"
const nodeHecoTest = "hecotest"

var nodeUrlMap = map[string]string{
	nodeMainnet:  "wss://mainnet.infura.io/ws/v3/21a9f5ba4bce425795cac796a66d7472", // please replace it
	nodeRopsten:  "wss://ropsten.infura.io/ws/v3/21a9f5ba4bce425795cac796a66d7472", // please replace it
	nodeKovan:    "wss://kovan.infura.io/ws/v3/21a9f5ba4bce425795cac796a66d7472",   // please replace it
	nodeRinkeby:  "wss://rinkeby.infura.io/ws/v3/21a9f5ba4bce425795cac796a66d7472", // please replace it
	nodeGoerli:   "wss://goerli.infura.io/ws/v3/21a9f5ba4bce425795cac796a66d7472",  // please replace it
	nodeSokol:    "https://sokol.poa.network",
	nodeHeco:     "wss://ws-mainnet-node.huobichain.com",
	nodeHecoTest: "wss://ws-testnet.hecochain.com",
}

var nodeTxLinkMap = map[string]string{
	nodeMainnet:  "https://etherscan.io/tx/",
	nodeRopsten:  "https://ropsten.etherscan.io/tx/",
	nodeKovan:    "https://kovan.etherscan.io/tx/",
	nodeRinkeby:  "https://rinkeby.etherscan.io/tx/",
	nodeGoerli:   "https://goerli.etherscan.io/tx/",
	nodeSokol:    "https://blockscout.com/poa/sokol/tx/",
	nodeHeco:     "https://scan.hecochain.com/tx/",
	nodeHecoTest: "https://scan-testnet.hecochain.com/tx/",
}

// Execute cobra root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&nodeUrlOpt, "node-url", "", "", "the target connection node url, if this option specified, the --node option is ignored")
	rootCmd.PersistentFlags().StringVarP(&nodeOpt, "node", "", "kovan", "mainnet | ropsten | kovan | rinkeby | sokol, the node type")
	rootCmd.PersistentFlags().StringVarP(&gasPriceOpt, "gas-price", "", "", "the gas price, unit is gwei.")
	rootCmd.PersistentFlags().Uint64VarP(&gasLimitOpt, "gas-limit", "", 0, "the gas limit")
	rootCmd.PersistentFlags().StringVarP(&privateKeyOpt, "private-key", "k", "", "the private key, eth would be send from this account")
	rootCmd.PersistentFlags().BoolVarP(&terseOutputOpt, "terse", "", false, "produce terse output")

	rootCmd.AddCommand(balanceCmd)
	rootCmd.AddCommand(transferCmd)
	rootCmd.AddCommand(decodeRawTxCmd)
	rootCmd.AddCommand(dropPendingTxCmd)
	rootCmd.AddCommand(genkeyCmd)
	rootCmd.AddCommand(dumpAddrCmd)
	rootCmd.AddCommand(callCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(computeContractAddrCmd)
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
	if !contains([]string{nodeMainnet, nodeRopsten, nodeKovan, nodeRinkeby, nodeGoerli, nodeSokol,
		nodeHeco, nodeHecoTest}, nodeOpt) {
		log.Printf("invalid option for --node: %v", nodeOpt)
		rootCmd.Help()
		os.Exit(1)
	}

	if nodeUrlOpt == "" {
		nodeUrlOpt = nodeUrlMap[nodeOpt]
	}

	if gasPriceOpt != "" {
		if _, err = decimal.NewFromString(gasPriceOpt); err != nil {
			log.Printf("invalid option for --gas-price: %v", gasPriceOpt)
			rootCmd.Help()
			os.Exit(1)
		}
	}
}
