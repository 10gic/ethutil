package cmd

import (
	"log"
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

var (
	globalOptNodeUrl     string
	globalOptNode        string
	globalOptGasPrice    string
	globalOptGasLimit    uint64
	globalOptPrivateKey  string
	globalOptTerseOutput bool
	globalOptDryRun      bool
	globalOptShowRawTx   bool
	rootCmd              = &cobra.Command{
		Use:   "ethutil",
		Short: "An Ethereum util, can transfer eth, check balance, drop pending tx, etc",
	}

	globalClient *Client
)

type Client struct {
	EthClient *ethclient.Client
	RpcClient *rpc.Client
}

// InitGlobalClient initializes a client that uses the given RPC client.
func InitGlobalClient(nodeUrl string) {
	rpcClient, err := rpc.Dial(nodeUrl)
	checkErr(err)

	globalClient = &Client{
		EthClient: ethclient.NewClient(rpcClient),
		RpcClient: rpcClient,
	}
}

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

var nodeTxExplorerUrlMap = map[string]string{
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
	cobra.EnableCommandSorting = false
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&globalOptNodeUrl, "node-url", "", "", "the target connection node url, if this option specified, the --node option is ignored")
	rootCmd.PersistentFlags().StringVarP(&globalOptNode, "node", "", "kovan", "mainnet | ropsten | kovan | rinkeby | sokol, the node type")
	rootCmd.PersistentFlags().StringVarP(&globalOptGasPrice, "gas-price", "", "", "the gas price, unit is gwei.")
	rootCmd.PersistentFlags().Uint64VarP(&globalOptGasLimit, "gas-limit", "", 0, "the gas limit")
	rootCmd.PersistentFlags().StringVarP(&globalOptPrivateKey, "private-key", "k", "", "the private key, eth would be send from this account")
	rootCmd.PersistentFlags().BoolVarP(&globalOptTerseOutput, "terse", "", false, "produce terse output")
	rootCmd.PersistentFlags().BoolVarP(&globalOptDryRun, "dry-run", "", false, "do not broadcast tx")
	rootCmd.PersistentFlags().BoolVarP(&globalOptShowRawTx, "show-raw-tx", "", false, "print raw signed tx")

	rootCmd.AddCommand(balanceCmd)
	rootCmd.AddCommand(transferCmd)
	rootCmd.AddCommand(callCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(dropTxCmd)
	rootCmd.AddCommand(genkeyCmd)
	rootCmd.AddCommand(dumpAddrCmd)
	rootCmd.AddCommand(computeContractAddrCmd)
	rootCmd.AddCommand(decodeTxCmd)
	rootCmd.AddCommand(getCodeCmd)
}

func initConfig() {
	var err error
	globalOptNodeUrl, err = rootCmd.Flags().GetString("node-url")
	checkErr(err)
	globalOptNode, err = rootCmd.Flags().GetString("node")
	checkErr(err)
	globalOptGasPrice, err = rootCmd.Flags().GetString("gas-price")
	checkErr(err)
	globalOptTerseOutput, err = rootCmd.Flags().GetBool("terse")
	checkErr(err)

	// validation
	if !contains([]string{nodeMainnet, nodeRopsten, nodeKovan, nodeRinkeby, nodeGoerli, nodeSokol,
		nodeHeco, nodeHecoTest}, globalOptNode) {
		log.Printf("invalid option for --node: %v", globalOptNode)
		_ = rootCmd.Help()
		os.Exit(1)
	}

	if globalOptNodeUrl == "" {
		globalOptNodeUrl = nodeUrlMap[globalOptNode]
	}

	if globalOptGasPrice != "" {
		if _, err = decimal.NewFromString(globalOptGasPrice); err != nil {
			log.Printf("invalid option for --gas-price: %v", globalOptGasPrice)
			_ = rootCmd.Help()
			os.Exit(1)
		}
	}
}
