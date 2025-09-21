package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spf13/cobra"
)

type ChainData struct {
	Name      string        `json:"name"`
	Chain     string        `json:"chain"`
	ChainId   uint64        `json:"chainId"`
	Rpc       []RpcEndpoint `json:"rpc"`
	ChainSlug string        `json:"chainSlug"`
}

type RpcEndpoint struct {
	Url          string `json:"url"`
	Tracking     string `json:"tracking"`
	IsOpenSource bool   `json:"isOpenSource,omitempty"`
}

type RpcStatus struct {
	Url           string
	BlockHeight   *big.Int
	ClientVersion string
	Error         string
	ResponseTime  time.Duration
}

var publicRpcCmd = &cobra.Command{
	Use:   "public-rpc <chain-id>",
	Short: "Show public RPC endpoints for a chain",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("requires exactly one chain ID argument")
		}

		if _, err := strconv.ParseUint(args[0], 10, 64); err != nil {
			return fmt.Errorf("chain ID must be a valid number: %v", args[0])
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		chainId := args[0]

		log.Printf("Fetching RPC data for chain ID %s", chainId)

		// Fetch chain data from chainlist.org
		resp, err := http.Get("https://chainlist.org/rpcs.json")
		if err != nil {
			log.Printf("Failed to fetch RPC data: %v", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response body: %v", err)
			os.Exit(1)
		}

		var chains []ChainData
		err = json.Unmarshal(body, &chains)
		if err != nil {
			log.Printf("Failed to parse JSON: %v", err)
			os.Exit(1)
		}

		// Find the chain with matching chain ID
		var targetChain *ChainData
		for i, chain := range chains {
			if strconv.FormatUint(chain.ChainId, 10) == chainId {
				targetChain = &chains[i]
				break
			}
		}

		if targetChain == nil {
			log.Printf("Chain ID %s not found", chainId)
			os.Exit(1)
		}

		log.Printf("Found chain: %s (%s)", targetChain.Name, targetChain.Chain)

		if len(targetChain.Rpc) == 0 {
			fmt.Printf("No RPC endpoints found for chain %s\n", chainId)
			return
		}

		fmt.Printf("Chain: %s (ID: %s)\n", targetChain.Name, chainId)
		fmt.Printf("RPC Endpoints:\n\n")

		// Test each RPC endpoint and get block height
		for _, endpoint := range targetChain.Rpc {
			// Skip endpoints with template variables
			if strings.Contains(endpoint.Url, "${") {
				continue
			}

			status := testRpcEndpoint(endpoint.Url)

			if status.Error != "" {
				fmt.Printf("❌ %s\n   Error: %s\n\n", endpoint.Url, status.Error)
			} else {
				fmt.Printf("✅ %s\n   Block Height: %s\n   Client Version: %s\n   Response Time: %v\n   Tracking: %s\n\n",
					endpoint.Url,
					status.BlockHeight.String(),
					status.ClientVersion,
					status.ResponseTime,
					endpoint.Tracking)
			}
		}
	},
}

func testRpcEndpoint(rpcUrl string) RpcStatus {
	status := RpcStatus{
		Url: rpcUrl,
	}

	start := time.Now()

	// Create RPC client with timeout
	client, err := rpc.DialContext(context.Background(), rpcUrl)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		if len(errMsg) > 200 {
			errMsg = errMsg[:200] + "..."
		}
		status.Error = fmt.Sprintf("Connection failed: %s", errMsg)
		return status
	}
	defer client.Close()

	// Test with eth_blockNumber
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result string
	err = client.CallContext(ctx, &result, "eth_blockNumber")
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		if len(errMsg) > 200 {
			errMsg = errMsg[:200] + "..."
		}
		status.Error = fmt.Sprintf("eth_blockNumber failed: %s", errMsg)
		return status
	}

	status.ResponseTime = time.Since(start)

	// Convert hex block number to big.Int
	if strings.HasPrefix(result, "0x") {
		blockNum, success := new(big.Int).SetString(result[2:], 16)
		if !success {
			status.Error = fmt.Sprintf("Failed to parse block number: %s", result)
			return status
		}
		status.BlockHeight = blockNum
	} else {
		status.Error = fmt.Sprintf("Invalid block number format: %s", result)
		return status
	}

	// Get client version
	var clientVersion string
	err = client.CallContext(ctx, &clientVersion, "web3_clientVersion")
	if err != nil {
		status.ClientVersion = "Unknown"
	} else {
		status.ClientVersion = clientVersion
	}

	return status
}
