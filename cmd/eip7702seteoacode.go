package cmd

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"log"
	"math/big"
)

// eip7702SetEoaCodeCmd represents the setEoaCode command
var eip7702SetEoaCodeCmd = &cobra.Command{
	Use:   "eip7702-set-eoa-code <delegate-to>",
	Short: "Set EOA account code, see EIP-7702. Just use 0x0000000000000000000000000000000000000000 when you want to clear the code.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires an address")
		}
		if !isValidEthAddress(args[0]) {
			return fmt.Errorf("%v is not a valid eth address", args[0])
		}

		if globalOptPrivateKey == "" {
			log.Fatalf("--private-key is required for this command")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		InitGlobalClient(globalOptNodeUrl)

		var delegateTo = common.HexToAddress(args[0])

		// We just create a self transfer transaction, the amount is not important
		var valueInEther = decimal.RequireFromString("0")

		var privateKey *ecdsa.PrivateKey
		if globalOptPrivateKey != "" {
			privateKey = hexToPrivateKey(globalOptPrivateKey)
		}

		if globalOptPrivateKey == "" && buildRawTxSignData == "" {
			log.Fatalf("require --sign-data or --private-key")
		}

		var fromAddress = extractAddressFromPrivateKey(privateKey)
		var toAddress = fromAddress

		currentCode, err := globalClient.EthClient.CodeAt(context.Background(), fromAddress, nil)
		checkErr(err)

		log.Printf("%s current code = %s", fromAddress, hexutil.Encode(currentCode))

		signedTx, err := BuildEip7702SignedTx(globalClient.EthClient, privateKey, &toAddress, valueInEther.BigInt(), common.FromHex(buildRawTxHexData), common.FromHex(buildRawTxSignData), delegateTo)
		checkErr(err)

		signedRawTx, err := GenRawTx(signedTx)
		checkErr(err)

		fmt.Printf("signed raw tx (can be used by rpc eth_sendRawTransaction) = %v\n", signedRawTx)

		if globalOptDryRun {
			// dry run, do not send the transaction
			return
		}
		rpcReturnTx, err := SendRawTransaction(globalClient.RpcClient, signedRawTx)
		if err != nil {
			log.Fatalf("SendRawTransaction fail: %s", err)
		}

		log.Printf("tx %s is broadcasted", rpcReturnTx)
	},
}

// BuildEip7702SignedTx builds signed transaction
func BuildEip7702SignedTx(
	client *ethclient.Client, privateKey *ecdsa.PrivateKey,
	toAddress *common.Address, amount *big.Int, data []byte, sigData []byte, delegateTo common.Address,
) (*types.Transaction, error) {
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("NetworkID fail: %w", err)
	}

	tx, err := BuildEIP7702Tx(client, privateKey, toAddress, amount, data, delegateTo)
	if err != nil {
		return nil, fmt.Errorf("BuildTx fail: %w", err)
	}

	signer := types.NewPragueSigner(chainID)
	preHash := signer.Hash(tx)
	if globalOptShowPreHash {
		fmt.Printf("hash before ecdsa sign (hex) = %x\n", preHash.Bytes())
	}

	// If sigData is not provided, signs the transaction using the private key
	if len(sigData) == 0 {
		sigData, err = crypto.Sign(preHash[:], privateKey)
		if err != nil {
			return nil, err
		}
	}

	// attach sigData to tx
	signedTx, err := tx.WithSignature(signer, sigData)
	if err != nil {
		return nil, fmt.Errorf("WithSignature fail: %w", err)
	}

	return signedTx, nil
}

// BuildEIP7702Tx builds EIP7702 transaction
func BuildEIP7702Tx(client *ethclient.Client, privateKey *ecdsa.PrivateKey,
	toAddress *common.Address, amount *big.Int, data []byte, delegateTo common.Address,
) (*types.Transaction, error) {
	log.Printf("amount = %s", amount.String())

	var nonce uint64
	var err error
	if globalOptNonce < 0 {

		var account = extractAddressFromPrivateKey(privateKey)

		nonce, err = client.PendingNonceAt(context.Background(), account)
		if err != nil {
			return nil, fmt.Errorf("PendingNonceAt fail: %w", err)
		}
	} else {
		nonce = uint64(globalOptNonce)
	}

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("NetworkID fail: %w", err)
	}

	gasLimit := globalOptGasLimit
	if gasLimit == 0 { // if not specified
		estimateGasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{
			To:    toAddress,
			Value: amount,
			Data:  data,
		})
		if err != nil {
			return nil, fmt.Errorf("EstimateGas fail: %w", err)
		}
		gasLimit = estimateGasLimit
	}

	const PerEmptyAccountCost = 25000
	gasLimit = gasLimit + uint64(PerEmptyAccountCost) + 10000 // add some buffer

	var tx *types.Transaction

	var maxFeePerGasEstimate = new(big.Int)
	var maxPriorityFeePerGasEstimate = new(big.Int)
	if globalOptMaxPriorityFeePerGas == "" || globalOptMaxFeePerGas == "" {
		maxFeePerGasEstimate, maxPriorityFeePerGasEstimate, err = getEIP1559GasPriceByFeeHistory(client)
		if err != nil {
			return nil, fmt.Errorf("getEIP1559GasPriceByFeeHistory fail: %w", err)
		}
	}

	var maxPriorityFeePerGas *big.Int
	if globalOptMaxPriorityFeePerGas == "" {
		// Use estimate value
		maxPriorityFeePerGas = maxPriorityFeePerGasEstimate
	} else {
		// Use the value set by the user
		maxPriorityFeePerGasDecimal, _ := decimal.NewFromString(globalOptMaxPriorityFeePerGas)
		// convert from gwei to wei
		maxPriorityFeePerGas = maxPriorityFeePerGasDecimal.Mul(decimal.RequireFromString("1000000000")).BigInt()
	}

	var maxFeePerGas *big.Int
	if globalOptMaxFeePerGas == "" {
		// Use estimate value
		maxFeePerGas = maxFeePerGasEstimate
	} else {
		// Use the value set by the user
		maxFeePerGasDecimal, _ := decimal.NewFromString(globalOptMaxFeePerGas)
		// convert from gwei to wei
		maxFeePerGas = maxFeePerGasDecimal.Mul(decimal.RequireFromString("1000000000")).BigInt()
	}

	auth := types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainID),
		Address: delegateTo,
		// EIP-7702 says the authorization tuple execution time is:
		// At the start of executing the transaction (outer tx), after incrementing the sender’s nonce
		// https://eips.ethereum.org/EIPS/eip-7702
		// So, we need to increment the nonce by 1 if the sender of outer tx is the authority in the authorization tuple
		Nonce: nonce + 1,
	}

	signedAuth, err := types.SignSetCode(privateKey, auth)
	if err != nil {
		return nil, fmt.Errorf("SignSetCode fail: %w", err)
	}

	log.Printf("signed auth = %v", signedAuth)

	tx = types.NewTx(&types.SetCodeTx{
		ChainID:    uint256.MustFromBig(chainID),
		Nonce:      nonce,
		GasTipCap:  uint256.MustFromBig(maxPriorityFeePerGas),
		GasFeeCap:  uint256.MustFromBig(maxFeePerGas),
		Gas:        gasLimit,
		To:         *toAddress,
		Value:      uint256.MustFromBig(amount),
		Data:       data,
		AccessList: nil,
		AuthList:   []types.SetCodeAuthorization{signedAuth},
		V:          nil,
		R:          nil,
		S:          nil,
	})

	return tx, err
}
