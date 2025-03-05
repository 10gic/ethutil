package cmd

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
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

// https://docs.optimism.io/builders/tools/build/oracles#gas-oracle
var l1GasPriceOracle = common.HexToAddress("0x420000000000000000000000000000000000000F")

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

	if globalOptChain == nodeMainnet {
		// in case of mainnet, get gap price from ethgasstation
		gasPrice, err = getGasPriceFromEthGasStation()
		if err != nil {
			log.Fatalf("getGasPrice fail: %s", err)
		}
	}
	log.Printf("gas price %v wei", gasPrice)

	return gasPrice, nil
}

var transferCmd = &cobra.Command{
	Use:   "transfer <target-address> <amount>",
	Short: "Transfer native token",
	Long:  "Transfer AMOUNT of eth to TARGET-ADDRESS, special word `all` is valid amount. unit is ether, can be changed by --unit.",
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
			// Note: We always use EIP155 for transfer all balance
			// Because we need to pay for gas, and we need to know the exact balance excluding gas fee
			globalOptTxType = txTypeEip155

			// transfer all balance (only reserve some gas just pay for this tx) to target address
			fromAddr := extractAddressFromPrivateKey(hexToPrivateKey(globalOptPrivateKey))

			// Get current balance
			balance, err := globalClient.EthClient.BalanceAt(ctx, fromAddr, nil)
			checkErr(err)

			log.Printf("balance of %v is %v wei", fromAddr.String(), balance.String())

			var gasLimit = globalOptGasLimit
			if globalOptGasLimit == 0 {
				// If globalOptGasLimit is not specified, use the default value gasUsedByTransferEth
				toAddr := common.HexToAddress(targetAddress)
				estimateGasLimit, err := globalClient.EthClient.EstimateGas(context.Background(), ethereum.CallMsg{
					To:    &toAddr,
					Value: big.NewInt(0),
					Data:  common.FromHex(transferHexData),
				})
				if err != nil {
					log.Fatalf("EstimateGas fail: %w", err)
				}
				gasLimit = estimateGasLimit
			}
			gasMayUsed := big.NewInt(0).Mul(gasPrice, big.NewInt(int64(gasLimit)))

			if gasMayUsed.Cmp(balance) > 0 {
				log.Fatalf("insufficient balance %v, can not pay for gas %v", balance, gasMayUsed)
			}

			l1GasPriceOracleExisted, err := isContractAddress(globalClient.EthClient, l1GasPriceOracle)
			if err != nil {
				log.Fatalf("isContractAddress failed %v", err)
			}
			// If l1GasPriceOracle is existed in current chain, the current chain is probability L2 chain, and need L1 fee when submit tx
			// We must subtract `L2 fee + L1 fee` if use want transfer 'all' native token
			if l1GasPriceOracleExisted {
				// Estimate L1 Fee
				privateKey := hexToPrivateKey(globalOptPrivateKey)
				fromAddress := extractAddressFromPrivateKey(privateKey)
				toAddr := common.HexToAddress(targetAddress)
				signedTx, err := BuildSignedTx(globalClient.EthClient, privateKey, &fromAddress, &toAddr, big.NewInt(0).Sub(balance, gasMayUsed), gasPrice, common.FromHex(transferHexData), nil)
				if err != nil {
					log.Fatalf("BuildSignedTx fail: %v", err)
				}
				rawTx, err := GenRawTx(signedTx)
				if err != nil {
					log.Fatalf("GenRawTx fail: %v", err)
				}
				l1Fee, err := getL1Fee(globalClient.RpcClient, rawTx)
				if err != nil {
					log.Fatalf("getL1Fee failed %v", err)
				}
				log.Printf("L2 fee %v", gasMayUsed.String())
				log.Printf("L1 fee %v", l1Fee.String())
				// L2 fee + L1 fee
				gasMayUsed.Add(gasMayUsed, l1Fee)
			}

			amountBigInt := big.NewInt(0).Sub(balance, gasMayUsed)
			amount = bigIntToDecimal(amountBigInt)

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
		wei2Other(bigIntToDecimal(amountInWei), unitEther).String(),
		amountInWei.String(),
		extractAddressFromPrivateKey(hexToPrivateKey(privateKeyHex)).String(),
		toAddress)
	var toAddr = common.HexToAddress(toAddress)
	return Transact(rcpClient, client, hexToPrivateKey(privateKeyHex), &toAddr, amountInWei, gasPrice, data)
}

// getL1Fee call contract function getL1Fee to get the L1 fee
func getL1Fee(rcpClient *rpc.Client, rawTx string) (*big.Int, error) {
	txInputData, err := buildTxInputData("getL1Fee(bytes)", []string{rawTx})
	if err != nil {
		return nil, err
	}
	output, err := Call(rcpClient, l1GasPriceOracle, txInputData)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(output), nil
}
