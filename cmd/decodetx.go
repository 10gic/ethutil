package cmd

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/spf13/cobra"
)

var decodeTxCmd = &cobra.Command{
	Use:   "decode-tx tx-data",
	Short: "Decode raw transaction",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires tx-data")
		}
		if len(args) > 1 {
			return fmt.Errorf("multiple tx-data is not supported")
		}

		if !isValidHexString(args[0]) {
			return fmt.Errorf("tx-data must hex string")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		rawTxHexData := args[0]

		if strings.HasPrefix(rawTxHexData, "0x") {
			rawTxHexData = rawTxHexData[2:] // remove leading 0x
		}

		var firstHex = rawTxHexData[0:2]
		transactionType, err := strconv.ParseInt(firstHex, 16, 64)
		checkErr(err)

		if transactionType > 0x7f { // EIP-155
			decodeEip155(rawTxHexData)
		} else { // EIP-2718
			decodeEip2718(int(transactionType), rawTxHexData[2:])
		}
	},
}

func decodeEip155(rawTxHexData string) {
	var tx *types.Transaction
	rawTxBytes, _ := hex.DecodeString(rawTxHexData)
	err := rlp.DecodeBytes(rawTxBytes, &tx)
	if err != nil {
		panic("rlp decode failed, may not a valid eth raw transaction")
	}

	fmt.Printf("basic info (eip155):\n")
	fmt.Printf("nonce = %d\n", tx.Nonce())
	fmt.Printf("gasPrice = %s, i.e. %s Gwei\n", tx.GasPrice().String(), wei2Other(bigInt2Decimal(tx.GasPrice()), unitGwei).String())
	fmt.Printf("gasLimit = %d\n", tx.Gas())
	fmt.Printf("to = %s\n", tx.To().String())
	fmt.Printf("value = %s, i.e. %s Ether\n", tx.Value().String(), wei2Other(bigInt2Decimal(tx.Value()), unitEther).String())
	fmt.Printf("data (hex) = %x\n", tx.Data())

	if tx.ChainId().Int64() > 0 { // chain id is not available before eip155
		fmt.Printf("chainId = %s\n", tx.ChainId().String())
	}

	v, r, s := tx.RawSignatureValues()
	fmt.Printf("v = %s\n", v.String())
	fmt.Printf("r (hex) = %x\n", r)
	fmt.Printf("s (hex) = %x\n", s)

	fmt.Printf("\n")
	fmt.Printf("derived info:\n")
	fmt.Printf("txid (hex) = %x\n", tx.Hash().Bytes())
	var chainId = tx.ChainId()

	// build msg (hash of data) before sign
	singer := types.NewLondonSigner(chainId)
	hash := singer.Hash(tx)
	fmt.Printf("hash before ecdsa sign (hex) = %x\n", hash.Bytes())

	fmt.Printf("ecdsa recovery id = %d\n", getRecoveryId(v))

	pubkeyBytes, err := RecoverPubkey(v, r, s, hash.Bytes())
	checkErr(err)
	fmt.Printf("uncompressed 65 bytes public key of sender (hex) = %x\n", pubkeyBytes)

	// convert uncompressed public key to ecdsa.PublicKey
	pubkey, err := crypto.UnmarshalPubkey(pubkeyBytes)
	checkErr(err)

	// extract address from ecdsa.PublicKey
	addr := crypto.PubkeyToAddress(*pubkey)
	fmt.Printf("sender = %s\n", addr.Hex())
}

func decodeEip2718(transactionType int, transactionPayload string) {
	switch transactionType {
	case 1:
		// EIP-2930
		decodeEip2930(transactionType, transactionPayload)
	case 2:
		// EIP-1559
		decodeEip1559(transactionType, transactionPayload)
	default:
		panic("not implemented for this transaction type")
	}
}

func decodeEip2930(transactionType int, transactionPayload string) {
	var accessListTx *types.AccessListTx
	rawTxBytes, _ := hex.DecodeString(transactionPayload)
	err := rlp.DecodeBytes(rawTxBytes, &accessListTx)
	if err != nil {
		panic("rlp decode failed, may not a valid eth raw transaction")
	}

	fmt.Printf("basic info (eip2930):\n")
	fmt.Printf("transactionType = %v\n", transactionType)
	fmt.Printf("chainId = %s\n", accessListTx.ChainID.String())
	fmt.Printf("nonce = %d\n", accessListTx.Nonce)
	fmt.Printf("gasPrice = %s, i.e. %s Gwei\n", accessListTx.GasPrice.String(), wei2Other(bigInt2Decimal(accessListTx.GasPrice), unitGwei).String())
	fmt.Printf("gasLimit = %d\n", accessListTx.Gas)
	fmt.Printf("to = %s\n", accessListTx.To.String())
	fmt.Printf("value = %s, i.e. %s Ether\n", accessListTx.Value.String(), wei2Other(bigInt2Decimal(accessListTx.Value), unitEther).String())
	fmt.Printf("data (hex) = %x\n", accessListTx.Data)
	fmt.Printf("accessList = %v\n", accessListTx.AccessList)
	fmt.Printf("yParity (ecdsa recovery id) = %s\n", accessListTx.V)
	fmt.Printf("r (hex) = %x\n", accessListTx.R)
	fmt.Printf("s (hex) = %x\n", accessListTx.S)

	fmt.Printf("\n")
	fmt.Printf("derived info:\n")

	tx := types.NewTx(&types.AccessListTx{
		Nonce:    accessListTx.Nonce,
		To:       accessListTx.To,
		Value:    accessListTx.Value,
		Gas:      accessListTx.Gas,
		GasPrice: accessListTx.GasPrice,
		Data:     accessListTx.Data,
	})

	fmt.Printf("txid (hex) = %x\n", tx.Hash().Bytes())

	// build msg (hash of data) before sign
	singer := types.NewLondonSigner(accessListTx.ChainID)
	hash := singer.Hash(tx)
	fmt.Printf("hash before ecdsa sign (hex) = %x\n", hash.Bytes())

	pubkeyBytes, err := RecoverPubkey(accessListTx.V, accessListTx.R, accessListTx.S, hash.Bytes())
	checkErr(err)
	fmt.Printf("uncompressed 65 bytes public key of sender (hex) = %x\n", pubkeyBytes)

	// convert uncompressed public key to ecdsa.PublicKey
	pubkey, err := crypto.UnmarshalPubkey(pubkeyBytes)
	checkErr(err)

	// extract address from ecdsa.PublicKey
	addr := crypto.PubkeyToAddress(*pubkey)
	fmt.Printf("sender = %s\n", addr.Hex())
}

func decodeEip1559(transactionType int, transactionPayload string) {
	var dynamicFeeTx *types.DynamicFeeTx
	rawTxBytes, _ := hex.DecodeString(transactionPayload)
	err := rlp.DecodeBytes(rawTxBytes, &dynamicFeeTx)
	if err != nil {
		panic("rlp decode failed, may not a valid eth raw transaction")
	}

	fmt.Printf("basic info (eip1559):\n")
	fmt.Printf("transactionType = %v\n", transactionType)
	fmt.Printf("chainId = %s\n", dynamicFeeTx.ChainID.String())
	fmt.Printf("nonce = %d\n", dynamicFeeTx.Nonce)
	fmt.Printf("maxPriorityFeePerGas = %s, i.e. %s Gwei\n", dynamicFeeTx.GasTipCap.String(), wei2Other(bigInt2Decimal(dynamicFeeTx.GasTipCap), unitGwei).String())
	fmt.Printf("maxFeePerGas = %s, i.e. %s Gwei\n", dynamicFeeTx.GasFeeCap.String(), wei2Other(bigInt2Decimal(dynamicFeeTx.GasFeeCap), unitGwei).String())
	fmt.Printf("gasLimit = %d\n", dynamicFeeTx.Gas)
	fmt.Printf("to = %s\n", dynamicFeeTx.To.String())
	fmt.Printf("value = %s, i.e. %s Ether\n", dynamicFeeTx.Value.String(), wei2Other(bigInt2Decimal(dynamicFeeTx.Value), unitEther).String())
	fmt.Printf("data (hex) = %x\n", dynamicFeeTx.Data)
	fmt.Printf("accessList = %v\n", dynamicFeeTx.AccessList)
	fmt.Printf("yParity (ecdsa recovery id) = %s\n", dynamicFeeTx.V)
	fmt.Printf("r (hex) = %x\n", dynamicFeeTx.R)
	fmt.Printf("s (hex) = %x\n", dynamicFeeTx.S)

	fmt.Printf("\n")
	fmt.Printf("derived info:\n")

	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     dynamicFeeTx.Nonce,
		To:        dynamicFeeTx.To,
		Value:     dynamicFeeTx.Value,
		Gas:       dynamicFeeTx.Gas,
		GasFeeCap: dynamicFeeTx.GasFeeCap,
		GasTipCap: dynamicFeeTx.GasTipCap,
		Data:      dynamicFeeTx.Data,
	})

	fmt.Printf("txid (hex) = %x\n", tx.Hash().Bytes())

	// build msg (hash of data) before sign
	singer := types.NewLondonSigner(dynamicFeeTx.ChainID)
	hash := singer.Hash(tx)
	fmt.Printf("hash before ecdsa sign (hex) = %x\n", hash.Bytes())

	pubkeyBytes, err := RecoverPubkey(dynamicFeeTx.V, dynamicFeeTx.R, dynamicFeeTx.S, hash.Bytes())
	checkErr(err)
	fmt.Printf("uncompressed 65 bytes public key of sender (hex) = %x\n", pubkeyBytes)

	// convert uncompressed public key to ecdsa.PublicKey
	pubkey, err := crypto.UnmarshalPubkey(pubkeyBytes)
	checkErr(err)

	// extract address from ecdsa.PublicKey
	addr := crypto.PubkeyToAddress(*pubkey)
	fmt.Printf("sender = %s\n", addr.Hex())
}
