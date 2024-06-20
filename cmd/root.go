package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

var (
	globalOptNodeUrl              string
	globalOptChain                string
	globalOptGasPrice             string
	globalOptMaxPriorityFeePerGas string
	globalOptMaxFeePerGas         string
	globalOptGasLimit             uint64
	globalOptNonce                int64
	globalOptPrivateKey           string
	globalOptTerseOutput          bool
	globalOptDryRun               bool
	globalOptShowPreHash          bool
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
const nodeSepolia = "sepolia"
const nodeSokol = "sokol"
const nodeBsc = "bsc"

var nodeUrlMap = map[string]string{
	nodeMainnet: "wss://mainnet.infura.io/ws/v3/21a9f5ba4bce425795cac796a66d7472", // please replace infura project id
	nodeSepolia: "wss://sepolia.infura.io/ws/v3/21a9f5ba4bce425795cac796a66d7472", // please replace infura project id
	nodeSokol:   "https://sokol.poa.network",
	nodeBsc:     "https://bsc-dataseed1.binance.org",
}

var nodeTxExplorerUrlMap = map[string]string{
	nodeMainnet: "https://etherscan.io/tx/",
	nodeSepolia: "https://sepolia.etherscan.io/tx/",
	nodeSokol:   "https://blockscout.com/poa/sokol/tx/",
	nodeBsc:     "https://bscscan.com/tx/",
}

var nodeApiUrlMap = map[string]string{
	nodeMainnet: "https://api.etherscan.io/api?module=contract&action=getsourcecode&address=%s",
	nodeSepolia: "https://api-sepolia.etherscan.io/api?module=contract&action=getsourcecode&address=%s",
	nodeSokol:   "https://blockscout.com/poa/sokol/api?module=contract&action=getsourcecode&address=%s",
	nodeBsc:     "https://api.bscscan.com/api?module=contract&action=getsourcecode&address=%s",
}

// Execute cobra root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.EnableCommandSorting = false
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&globalOptNodeUrl, "node-url", "", "", "the target connection node url, if this option specified, the --chain option is ignored")
	rootCmd.PersistentFlags().StringVarP(&globalOptChain, "chain", "", "sepolia", "mainnet | sepolia | sokol | bsc. This parameter can be set as the chain ID, in this case the rpc comes from https://chainid.network/chains_mini.json")
	rootCmd.PersistentFlags().StringVarP(&globalOptGasPrice, "gas-price", "", "", "the gas price, unit is gwei.")
	rootCmd.PersistentFlags().StringVarP(&globalOptMaxPriorityFeePerGas, "max-priority-fee-per-gas", "", "", "maximum fee per gas they are willing to give to miners, unit is gwei. see eip1559")
	rootCmd.PersistentFlags().StringVarP(&globalOptMaxFeePerGas, "max-fee-per-gas", "", "", "maximum fee per gas they are willing to pay total, unit is gwei. see eip1559")
	rootCmd.PersistentFlags().Uint64VarP(&globalOptGasLimit, "gas-limit", "", 0, "the gas limit")
	rootCmd.PersistentFlags().Int64VarP(&globalOptNonce, "nonce", "", -1, "the nonce, -1 means check online")
	rootCmd.PersistentFlags().StringVarP(&globalOptPrivateKey, "private-key", "k", "", "the private key, eth would be send from this account")
	rootCmd.PersistentFlags().BoolVarP(&globalOptTerseOutput, "terse", "", false, "produce terse output")
	rootCmd.PersistentFlags().BoolVarP(&globalOptDryRun, "dry-run", "", false, "do not broadcast tx")
	rootCmd.PersistentFlags().BoolVarP(&globalOptShowPreHash, "show-pre-hash", "", false, "print pre hash, the input of ecdsa sign")
	rootCmd.PersistentFlags().BoolVarP(&globalOptShowRawTx, "show-raw-tx", "", false, "print raw signed tx")
	rootCmd.PersistentFlags().BoolVarP(&globalOptShowInputData, "show-input-data", "", false, "print input data of tx")
	rootCmd.PersistentFlags().BoolVarP(&globalOptShowEstimateGas, "show-estimate-gas", "", false, "print estimate gas of tx")
	rootCmd.PersistentFlags().StringVarP(&globalOptTxType, "tx-type", "", "eip1559", "eip155 | eip1559, the type of tx your want to send")

	rootCmd.AddCommand(balanceCmd)
	rootCmd.AddCommand(transferCmd)
	rootCmd.AddCommand(callCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(deployErc20Cmd)
	rootCmd.AddCommand(dropTxCmd)
	rootCmd.AddCommand(fourByteCmd)
	rootCmd.AddCommand(encodeParamCmd)
	rootCmd.AddCommand(genkeyCmd)
	rootCmd.AddCommand(dumpAddrCmd)
	rootCmd.AddCommand(computeContractAddrCmd)
	rootCmd.AddCommand(buildRawTxCmd)
	rootCmd.AddCommand(broadcastTxCmd)
	rootCmd.AddCommand(decodeTxCmd)
	rootCmd.AddCommand(getCodeCmd)
	rootCmd.AddCommand(erc20Cmd)
	rootCmd.AddCommand(keccakCmd)
	rootCmd.AddCommand(personalSignCmd)
	rootCmd.AddCommand(eip712SignCmd)
	rootCmd.AddCommand(aaSimpleAccountCmd)
	rootCmd.AddCommand(downloadSrcCmd)
}

func initConfig() {
	var err error

	if !contains([]string{nodeMainnet, nodeSepolia, nodeSokol, nodeBsc}, globalOptChain) {
		var chainId = globalOptChain
		log.Printf("Query node rpc for chain %s", chainId)

		// Get rpc from https://chainid.network/chains_mini.json
		resp, err := http.Get("https://chainid.network/chains_mini.json")
		checkErr(err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		checkErr(err)

		type respItem struct {
			Name           string `json:"name"`
			ChainId        uint64 `json:"chainId"`
			ShortName      string `json:"shortName"`
			NativeCurrency struct {
				Symbol   string `json:"symbol"`
				Decimals uint64 `json:"decimals"`
			} `json:"nativeCurrency"`
			Rpc     []string `json:"rpc"`
			InfoUrl string   `json:"infoURL"`
		}

		var data []respItem
		err = json.Unmarshal(body, &data)
		checkErr(err)

		var finalRpc = ""
		for _, item := range data {
			if strconv.Itoa(int(item.ChainId)) == chainId {
				if item.NativeCurrency.Decimals != 18 {
					panic(fmt.Sprintf("Only support chain with decimals 18, but %s is %d", globalOptChain, item.NativeCurrency.Decimals))
				}

				// filter out rpc contains '${', for example, https://mainnet.infura.io/v3/${INFURA_API_KEY}
				for _, nodeRpc := range item.Rpc {
					if !strings.Contains(nodeRpc, "${") {
						finalRpc = nodeRpc
					}
				}
				break
			}
		}

		if finalRpc == "" {
			log.Printf("Chain %v is not supported", globalOptChain)
			os.Exit(1)
		}

		log.Printf("Use node rpc %s", finalRpc)
		globalOptNodeUrl = finalRpc
	}

	if globalOptNodeUrl == "" {
		globalOptNodeUrl = nodeUrlMap[globalOptChain]
	} else {
		// Clear globalOptChain if globalOptNodeUrl is provided
		globalOptChain = ""
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
