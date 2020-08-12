package cmd

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"log"
	"math/big"
	"os"
)

var transferPrivateKeyHex string
var transferTargetAddr string
var transferUnit string
var transferAmt string
var transferHexData []byte

func init() {
	transferCmd.Flags().StringVarP(&transferPrivateKeyHex, "private-key", "k", "", "the private key, eth would be send from this account")
	transferCmd.Flags().StringVarP(&transferTargetAddr, "to-addr", "t", "", "the target address you want to transfer eth")
	transferCmd.Flags().StringVarP(&transferUnit, "unit", "u", "ether", "wei | gwei | ether, unit of amount")
	transferCmd.Flags().StringVarP(&transferAmt, "amount", "n", "", "the amount you want to transfer, special word \"all\" means all balance would transfer to target address, unit is specified by --unit")
	transferCmd.Flags().BytesHexVarP(&transferHexData, "hex-data", "", nil, "the payload hex data when transfer, please remove the leading 0x")

	transferCmd.MarkFlagRequired("private-key")
	transferCmd.MarkFlagRequired("to-addr")
	transferCmd.MarkFlagRequired("amount")
}

func validationTransferCmdOpts() bool {
	// validation
	if !contains([]string{unitWei, unitGwei, unitEther}, transferUnit) {
		log.Fatalf("invalid option for --unit: %v", transferUnit)
		return false
	}

	if ! isValidEthAddress(transferTargetAddr) {
		log.Fatalf("%v is not a valid eth address", transferTargetAddr)
		return false
	}

	return true
}

const gasUsedByTransferEth = 21000 // The gas used by any transfer is always 21000

func getGasPrice(client *ethclient.Client) (*big.Int, error) {
	var gasPrice *big.Int
	if gasPriceOpt != "" {
		gasPriceDecimal, err := decimal.NewFromString(gasPriceOpt)
		if err != nil {
			return nil, err
		}
		// convert from gwei to wei
		return gasPriceDecimal.Mul(decimal.RequireFromString("1000000000")).BigInt(), nil
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())

	if nodeOpt == nodeMainnet {
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
	Use:   "transfer",
	Short: "Transfer eth to another address",
	Run: func(cmd *cobra.Command, args []string) {
		if ! validationTransferCmdOpts() {
			cmd.Help()
			os.Exit(1)
		}

		ctx := context.Background()

		client, err := ethclient.Dial(nodeUrlOpt)
		checkErr(err)

		gasPrice, err := getGasPrice(client)
		checkErr(err)

		var amount decimal.Decimal
		var amountInWei decimal.Decimal
		if transferAmt == "all" {
			// transfer all balance (only reserve some gas just pay for this tx) to target address
			fromAddr := extractAddressFromPrivateKey(buildPrivateKeyFromHex(transferPrivateKeyHex))
			balance, err := client.BalanceAt(ctx, fromAddr, nil)
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

		if tx, err := TransferHelper(client, transferPrivateKeyHex, transferTargetAddr, amountInWei.BigInt(), gasPrice, transferHexData); err != nil {
			log.Fatalf("transfer fail: %v", err)
		} else {
			log.Printf("transfer finished, tx = %v", tx)
		}
	},
}


func Transfer(client *ethclient.Client, privateKey *ecdsa.PrivateKey, toAddress common.Address, amount *big.Int, gasPrice *big.Int, data []byte) (string, error) {
	fromAddress := extractAddressFromPrivateKey(privateKey)

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", fmt.Errorf("PendingNonceAt fail: %w", err)
	}

	isContract, err := isContractAddress(client, toAddress)
	if err != nil {
		return "", fmt.Errorf("isContractAddress fail: %w", err)
	}

	gasLimit := uint64(gasUsedByTransferEth)
	if isContract { // gasUsedByTransferEth may be not enough if send to contract
		gasLimit = 900000
	}
	if len(data) > 0 { // gasUsedByTransferEth may be not enough if with payload data
		gasLimit = 900000
	}

	tx := types.NewTransaction(nonce, toAddress, amount, gasLimit, gasPrice, data)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return "", fmt.Errorf("NetworkID fail: %w", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("SignTx fail: %w", err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("SendTransaction fail: %w", err)
	}

	//log.Printf("tx sent: %s", signedTx.Hash().Hex())

	rp, err := getReceipt(client, signedTx.Hash(), 0)
	if err != nil {
		return "", fmt.Errorf("getReceipt fail: %w", err)
	}

	if rp.Status != types.ReceiptStatusSuccessful {
		return "", fmt.Errorf("tx (%v) fail", signedTx.Hash().String())
	}

	return signedTx.Hash().String(), nil
}

func TransferHelper(client *ethclient.Client, privateKeyHex string, toAddress string, amountInWei *big.Int, gasPrice *big.Int, data []byte) (string, error) {
	log.Printf("transfer %v ether (%v wei) from %v to %v",
		wei2Other(bigInt2Decimal(amountInWei), unitEther).String(),
		amountInWei.String(),
		extractAddressFromPrivateKey(buildPrivateKeyFromHex(privateKeyHex)).String(),
		toAddress)
	return Transfer(client, buildPrivateKeyFromHex(privateKeyHex), common.HexToAddress(toAddress), amountInWei, gasPrice, data)
}
