package cmd

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

var transferUnit string
var transferNotCheck bool
var transferHexData string

func init() {
	transferCmd.Flags().StringVarP(&transferUnit, "unit", "u", "ether", "wei | gwei | ether, unit of amount")
	transferCmd.Flags().BoolVarP(&transferNotCheck, "not-check", "", false, "don't check result, return immediately after send transaction")
	transferCmd.Flags().StringVarP(&transferHexData, "hex-data", "", "", "the payload hex data when transfer")
}

func validationTransferCmdOpts() bool {
	// validation
	if !contains([]string{unitWei, unitGwei, unitEther}, transferUnit) {
		log.Fatalf("invalid option for --unit: %v", transferUnit)
		return false
	}

	if globalOptPrivateKey == "" {
		log.Fatalf("--private-key is required for transfer command")
		return false
	}

	if !isValidHexString(transferHexData) {
		log.Fatalf("--hex-data must hex string")
		return false
	}

	return true
}

const gasUsedByTransferEth = 21000 // The gas used by any transfer is always 21000

func getGasPrice(client *ethclient.Client) (*big.Int, error) {
	var gasPrice *big.Int
	if globalOptGasPrice != "" {
		gasPriceDecimal, err := decimal.NewFromString(globalOptGasPrice)
		if err != nil {
			return nil, err
		}
		// convert from gwei to wei
		return gasPriceDecimal.Mul(decimal.RequireFromString("1000000000")).BigInt(), nil
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())

	if globalOptNode == nodeMainnet {
		// in case of mainnet, get gap price from ethgasstation
		gasPrice, err = getGasPriceFromEthgasstation()
		if err != nil {
			log.Fatalf("getGasPrice fail: %s", err)
		}
	}
	log.Printf("gas price %v wei", gasPrice)

	return gasPrice, nil
}

var transferCmd = &cobra.Command{
	Use:   "transfer target-address amount",
	Short: "Transfer amount of eth to target-address",
	Long: "Transfer amount of eth to target-address, special word `all` is valid amount. unit is ether, can be changed by --unit.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires target-address and amount")
		}
		if len(args) == 1 {
			return fmt.Errorf("requires amount")
		}
		if len(args) > 2 {
			return fmt.Errorf("too many args")
		}

		targetAddress := args[0]
		transferAmt := args[1]

		if !isValidEthAddress(targetAddress) {
			return fmt.Errorf("%v is not a valid eth address", targetAddress)
		}

		if transferAmt == "all" {
			return nil
		} else {
			_, err := decimal.NewFromString(transferAmt)
			if err != nil {
				return fmt.Errorf("%v is not a valid amount", transferAmt)
			}
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !validationTransferCmdOpts() {
			_ = cmd.Help()
			os.Exit(1)
		}

		targetAddress := args[0]
		transferAmt := args[1]

		InitGlobalClient(globalOptNodeUrl)

		ctx := context.Background()

		gasPrice, err := getGasPrice(globalClient.EthClient)
		checkErr(err)

		var amount decimal.Decimal
		var amountInWei decimal.Decimal
		if transferAmt == "all" {
			// transfer all balance (only reserve some gas just pay for this tx) to target address
			fromAddr := extractAddressFromPrivateKey(buildPrivateKeyFromHex(globalOptPrivateKey))
			balance, err := globalClient.EthClient.BalanceAt(ctx, fromAddr, nil)
			checkErr(err)

			log.Printf("balance of %v is %v wei", fromAddr.String(), balance.String())

			gasMayUsed := big.NewInt(0).Mul(gasPrice, big.NewInt(gasUsedByTransferEth))

			if gasMayUsed.Cmp(balance) > 0 {
				log.Fatalf("insufficient balance %v, can not pay for gas %v", balance, gasMayUsed)
			}

			amountBigInt := big.NewInt(0).Sub(balance, gasMayUsed)
			amount = bigInt2Decimal(amountBigInt)

			amountInWei = amount
		} else {
			amount = decimal.RequireFromString(transferAmt)
			amountInWei = unify2Wei(amount, transferUnit)
		}

		if tx, err := TransferHelper(globalClient.RpcClient, globalClient.EthClient, globalOptPrivateKey, targetAddress, amountInWei.BigInt(), gasPrice, common.FromHex(transferHexData)); err != nil {
			log.Fatalf("transfer fail: %v", err)
		} else {
			log.Printf("transfer finished, tx = %v", tx)
		}
	},
}

func TransferHelper(rcpClient *rpc.Client, client *ethclient.Client, privateKeyHex string, toAddress string, amountInWei *big.Int, gasPrice *big.Int, data []byte) (string, error) {
	log.Printf("transfer %v ether (%v wei) from %v to %v",
		wei2Other(bigInt2Decimal(amountInWei), unitEther).String(),
		amountInWei.String(),
		extractAddressFromPrivateKey(buildPrivateKeyFromHex(privateKeyHex)).String(),
		toAddress)
	var toAddr = common.HexToAddress(toAddress)
	return Transact(rcpClient, client, buildPrivateKeyFromHex(privateKeyHex), &toAddr, amountInWei, gasPrice, data)
}
