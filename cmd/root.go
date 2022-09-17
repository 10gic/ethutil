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
	globalOptNodeUrl              string
	globalOptNode                 string
	globalOptGasPrice             string
	globalOptMaxPriorityFeePerGas string
	globalOptMaxFeePerGas         string
	globalOptGasLimit             uint64
	globalOptNonce                int64
	globalOptPrivateKey           string
	globalOptTerseOutput          bool
	globalOptDryRun               bool
	globalOptShowRawTx            bool
	globalOptShowInputData        bool
	globalOptShowEstimateGas      bool
	globalOptTxType               string
	rootCmd                       = &cobra.Command{
		Use:   "ethutil",
		Short: "An Ethereum util, can transfer eth, check balance, call any contract function etc",
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

const txTypeEip155 = "eip155"
const txTypeEip1559 = "eip1559"

const nodeMainnet = "mainnet"
const nodeGoerli = "goerli"
const nodeSepolia = "sepolia"
const nodeSokol = "sokol"
const nodeBsc = "bsc"
const nodeHeco = "heco"

var nodeUrlMap = map[string]string{
	nodeMainnet: "wss://mainnet.infura.io/ws/v3/21a9f5ba4bce425795cac796a66d7472", // please replace infura project id
	nodeGoerli:  "wss://goerli.infura.io/ws/v3/21a9f5ba4bce425795cac796a66d7472",  // please replace infura project id
	nodeSepolia: "wss://sepolia.infura.io/v3/21a9f5ba4bce425795cac796a66d7472",    // please replace infura project id
	nodeSokol:   "https://sokol.poa.network",
	nodeBsc:     "https://bsc-dataseed1.binance.org",
	nodeHeco:    "wss://ws-mainnet-node.huobichain.com",
}

var nodeTxExplorerUrlMap = map[string]string{
	nodeMainnet: "https://etherscan.io/tx/",
	nodeGoerli:  "https://goerli.etherscan.io/tx/",
	nodeSepolia: "https://sepolia.etherscan.io/tx/",
	nodeSokol:   "https://blockscout.com/poa/sokol/tx/",
	nodeBsc:     "https://bscscan.com/tx/",
	nodeHeco:    "https://scan.hecochain.com/tx/",
}

var nodeApiUrlMap = map[string]string{
	nodeMainnet: "https://api.etherscan.io/api?module=contract&action=getsourcecode&address=%s",
	nodeGoerli:  "https://api-goerli.etherscan.io/api?module=contract&action=getsourcecode&address=%s",
	nodeSepolia: "https://api-sepolia.etherscan.io/api?module=contract&action=getsourcecode&address=%s",
	nodeSokol:   "https://blockscout.com/poa/sokol/api?module=contract&action=getsourcecode&address=%s",
	nodeBsc:     "https://api.bscscan.com/api?module=contract&action=getsourcecode&address=%s",
	nodeHeco:    "https://api.hecoinfo.com/api?module=contract&action=getsourcecode&address=%s",
}

// Execute cobra root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.EnableCommandSorting = false
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&globalOptNodeUrl, "node-url", "", "", "the target connection node url, if this option specified, the --node option is ignored")
	rootCmd.PersistentFlags().StringVarP(&globalOptNode, "node", "", "goerli", "mainnet | goerli | sepolia |sokol | bsc | heco, the node type")
	rootCmd.PersistentFlags().StringVarP(&globalOptGasPrice, "gas-price", "", "", "the gas price, unit is gwei.")
	rootCmd.PersistentFlags().StringVarP(&globalOptMaxPriorityFeePerGas, "max-priority-fee-per-gas", "", "", "maximum fee per gas they are willing to give to miners, unit is gwei. see eip1559")
	rootCmd.PersistentFlags().StringVarP(&globalOptMaxFeePerGas, "max-fee-per-gas", "", "", "maximum fee per gas they are willing to pay total, unit is gwei. see eip1559")
	rootCmd.PersistentFlags().Uint64VarP(&globalOptGasLimit, "gas-limit", "", 0, "the gas limit")
	rootCmd.PersistentFlags().Int64VarP(&globalOptNonce, "nonce", "", -1, "the nonce, -1 means check online")
	rootCmd.PersistentFlags().StringVarP(&globalOptPrivateKey, "private-key", "k", "", "the private key, eth would be send from this account")
	rootCmd.PersistentFlags().BoolVarP(&globalOptTerseOutput, "terse", "", false, "produce terse output")
	rootCmd.PersistentFlags().BoolVarP(&globalOptDryRun, "dry-run", "", false, "do not broadcast tx")
	rootCmd.PersistentFlags().BoolVarP(&globalOptShowRawTx, "show-raw-tx", "", false, "print raw signed tx")
	rootCmd.PersistentFlags().BoolVarP(&globalOptShowInputData, "show-input-data", "", false, "print input data of tx")
	rootCmd.PersistentFlags().BoolVarP(&globalOptShowEstimateGas, "show-estimate-gas", "", false, "print estimate gas of tx")
	rootCmd.PersistentFlags().StringVarP(&globalOptTxType, "tx-type", "", "eip155", "eip155 | eip1559, the type of tx your want to send")

	rootCmd.AddCommand(balanceCmd)
	rootCmd.AddCommand(transferCmd)
	rootCmd.AddCommand(callCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(deployErc20Cmd)
	rootCmd.AddCommand(dropTxCmd)
	rootCmd.AddCommand(encodeParamCmd)
	rootCmd.AddCommand(genkeyCmd)
	rootCmd.AddCommand(dumpAddrCmd)
	rootCmd.AddCommand(computeContractAddrCmd)
	rootCmd.AddCommand(decodeTxCmd)
	rootCmd.AddCommand(getCodeCmd)
	rootCmd.AddCommand(erc20Cmd)
	rootCmd.AddCommand(keccakCmd)
	rootCmd.AddCommand(downloadSrcCmd)
}

func initConfig() {
	var err error

	// validation
	if !contains([]string{nodeMainnet, nodeGoerli, nodeSepolia, nodeSokol,
		nodeBsc, nodeHeco}, globalOptNode) {
		log.Printf("invalid option for --node: %v", globalOptNode)
		_ = rootCmd.Help()
		os.Exit(1)
	}

	if globalOptNodeUrl == "" {
		globalOptNodeUrl = nodeUrlMap[globalOptNode]

		// Print current network if --node-url is not specified
		log.Printf("Current network is %v", globalOptNode)
	}

	if globalOptGasPrice != "" {
		if _, err = decimal.NewFromString(globalOptGasPrice); err != nil {
			log.Printf("invalid option for --gas-price: %v", globalOptGasPrice)
			_ = rootCmd.Help()
			os.Exit(1)
		}
	}

	if globalOptMaxPriorityFeePerGas != "" {
		if _, err = decimal.NewFromString(globalOptMaxPriorityFeePerGas); err != nil {
			log.Printf("invalid option for --max-priority-fee-per-gas: %v", globalOptMaxPriorityFeePerGas)
			_ = rootCmd.Help()
			os.Exit(1)
		}
	}

	if globalOptMaxFeePerGas != "" {
		if _, err = decimal.NewFromString(globalOptMaxFeePerGas); err != nil {
			log.Printf("invalid option for --max-fee-per-gas: %v", globalOptMaxFeePerGas)
			_ = rootCmd.Help()
			os.Exit(1)
		}
	}

	if !contains([]string{txTypeEip155, txTypeEip1559}, globalOptTxType) {
		log.Printf("invalid option for --tx-type: %v", globalOptTxType)
		_ = rootCmd.Help()
		os.Exit(1)
	}
}
