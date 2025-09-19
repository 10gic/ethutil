package cmd

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"
	"strconv"
	"strings"
)

var decodeTxCmd = &cobra.Command{
	Use:   "decode-tx <tx-data>",
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

		if len(rawTxHexData) == 64 {
			// If only tx hash is provided, try to get raw tx data from tx hash
			InitGlobalClient(globalOptNodeUrl)
			rawTx, err := GetRawTx(globalClient.RpcClient, "0x"+rawTxHexData)
			if err != nil {
				fmt.Printf("get raw tx failed: %v\n", err)
				return
			}

			fmt.Printf("rax tx = %s\n", rawTx)
			rawTxHexData = rawTx[2:] // remove leading 0x
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

	fmt.Printf("basic info:\n")
	fmt.Printf("type = eip155, i.e. legacy transaction\n")
	if tx.ChainId().Int64() > 0 { // chain id is not available before eip155
		fmt.Printf("chainId = %s (0x%s)\n", tx.ChainId().String(), hex.EncodeToString(tx.ChainId().Bytes()))
	}
	fmt.Printf("nonce = %d (0x%x)\n", tx.Nonce(), tx.Nonce())
	fmt.Printf("gasPrice = %s (0x%s), i.e. %s Gwei\n", tx.GasPrice().String(), hex.EncodeToString(tx.GasPrice().Bytes()), wei2Other(bigIntToDecimal(tx.GasPrice()), unitGwei).String())
	fmt.Printf("gasLimit = %d (0x%x)\n", tx.Gas(), tx.Gas())
	if tx.To() == nil {
		fmt.Printf("to = nil (nil means contract creation)\n")
	} else {
		fmt.Printf("to = %s\n", tx.To().String())
	}
	fmt.Printf("value = %s (0x%s), i.e. %s Ether\n", tx.Value().String(), hex.EncodeToString(tx.Value().Bytes()), wei2Other(bigIntToDecimal(tx.Value()), unitEther).String())
	fmt.Printf("data (hex) = %x\n", tx.Data())

	v, r, s := tx.RawSignatureValues()
	fmt.Printf("v = %s (0x%s)\n", v.String(), hex.EncodeToString(v.Bytes()))
	fmt.Printf("r (hex) = %064x\n", r)
	fmt.Printf("s (hex) = %064x\n", s)

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
	fmt.Printf("uncompressed public key of sender (hex) = %x\n", pubkeyBytes)

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
	case 4:
		decodeEip7702(transactionType, transactionPayload)
	default:
		panic(fmt.Sprintf("not implemented for this transaction type %v", transactionType))
	}
}

func decodeEip2930(transactionType int, transactionPayload string) {
	var accessListTx *types.AccessListTx
	rawTxBytes, _ := hex.DecodeString(transactionPayload)
	err := rlp.DecodeBytes(rawTxBytes, &accessListTx)
	if err != nil {
		panic("rlp decode failed, may not a valid eth raw transaction")
	}

	fmt.Printf("basic info:\n")
	fmt.Printf("type = eip2930, i.e. TxnType = %v\n", transactionType)
	fmt.Printf("chainId = %s (0x%s)\n", accessListTx.ChainID.String(), hex.EncodeToString(accessListTx.ChainID.Bytes()))
	fmt.Printf("nonce = %d (0x%x)\n", accessListTx.Nonce, accessListTx.Nonce)
	fmt.Printf("gasPrice = %s (0x%s), i.e. %s Gwei\n", accessListTx.GasPrice.String(), hex.EncodeToString(accessListTx.GasPrice.Bytes()), wei2Other(bigIntToDecimal(accessListTx.GasPrice), unitGwei).String())
	fmt.Printf("gasLimit = %d (0x%x)\n", accessListTx.Gas, accessListTx.Gas)
	if accessListTx.To == nil {
		fmt.Printf("to = nil (nil means contract creation)\n")
	} else {
		fmt.Printf("to = %s\n", accessListTx.To.String())
	}
	fmt.Printf("value = %s (0x%s), i.e. %s Ether\n", accessListTx.Value.String(), hex.EncodeToString(accessListTx.Value.Bytes()), wei2Other(bigIntToDecimal(accessListTx.Value), unitEther).String())
	fmt.Printf("data (hex) = %x\n", accessListTx.Data)
	fmt.Printf("accessList = %v\n", accessListTx.AccessList)
	fmt.Printf("yParity (ecdsa recovery id) = %s (0x%s)\n", accessListTx.V, hex.EncodeToString(accessListTx.V.Bytes()))
	fmt.Printf("r (hex) = %064x\n", accessListTx.R)
	fmt.Printf("s (hex) = %064x\n", accessListTx.S)

	fmt.Printf("\n")
	fmt.Printf("derived info:\n")

	tx := types.NewTx(&types.AccessListTx{
		ChainID:  accessListTx.ChainID,
		Nonce:    accessListTx.Nonce,
		To:       accessListTx.To,
		Value:    accessListTx.Value,
		Gas:      accessListTx.Gas,
		GasPrice: accessListTx.GasPrice,
		Data:     accessListTx.Data,
		V:        accessListTx.V,
		R:        accessListTx.R,
		S:        accessListTx.S,
	})

	fmt.Printf("txid (hex) = %x\n", tx.Hash().Bytes())

	// build msg (hash of data) before sign
	singer := types.NewLondonSigner(accessListTx.ChainID)
	hash := singer.Hash(tx)
	fmt.Printf("hash before ecdsa sign (hex) = %x\n", hash.Bytes())

	pubkeyBytes, err := RecoverPubkey(accessListTx.V, accessListTx.R, accessListTx.S, hash.Bytes())
	checkErr(err)
	fmt.Printf("uncompressed public key of sender (hex) = %x\n", pubkeyBytes)

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

	fmt.Printf("basic info:\n")
	fmt.Printf("type = eip1559, i.e. TxnType = %v\n", transactionType)
	fmt.Printf("chainId = %s (0x%s)\n", dynamicFeeTx.ChainID.String(), hex.EncodeToString(dynamicFeeTx.ChainID.Bytes()))
	fmt.Printf("nonce = %d (0x%x)\n", dynamicFeeTx.Nonce, dynamicFeeTx.Nonce)
	fmt.Printf("maxPriorityFeePerGas = %s (0x%s), i.e. %s Gwei\n", dynamicFeeTx.GasTipCap.String(), hex.EncodeToString(dynamicFeeTx.GasTipCap.Bytes()), wei2Other(bigIntToDecimal(dynamicFeeTx.GasTipCap), unitGwei).String())
	fmt.Printf("maxFeePerGas = %s (0x%s), i.e. %s Gwei\n", dynamicFeeTx.GasFeeCap.String(), hex.EncodeToString(dynamicFeeTx.GasFeeCap.Bytes()), wei2Other(bigIntToDecimal(dynamicFeeTx.GasFeeCap), unitGwei).String())
	fmt.Printf("gasLimit = %d (0x%x)\n", dynamicFeeTx.Gas, dynamicFeeTx.Gas)
	if dynamicFeeTx.To == nil {
		fmt.Printf("to = nil (nil means contract creation)\n")
	} else {
		fmt.Printf("to = %s\n", dynamicFeeTx.To.String())
	}
	fmt.Printf("value = %s (0x%s), i.e. %s Ether\n", dynamicFeeTx.Value.String(), hex.EncodeToString(dynamicFeeTx.Value.Bytes()), wei2Other(bigIntToDecimal(dynamicFeeTx.Value), unitEther).String())
	fmt.Printf("data (hex) = %x\n", dynamicFeeTx.Data)
	fmt.Printf("accessList = %v\n", dynamicFeeTx.AccessList)
	fmt.Printf("yParity (ecdsa recovery id) = %s (0x%s)\n", dynamicFeeTx.V, hex.EncodeToString(dynamicFeeTx.V.Bytes()))
	fmt.Printf("r (hex) = %064x\n", dynamicFeeTx.R)
	fmt.Printf("s (hex) = %064x\n", dynamicFeeTx.S)

	fmt.Printf("\n")
	fmt.Printf("derived info:\n")

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   dynamicFeeTx.ChainID,
		Nonce:     dynamicFeeTx.Nonce,
		To:        dynamicFeeTx.To,
		Value:     dynamicFeeTx.Value,
		Gas:       dynamicFeeTx.Gas,
		GasFeeCap: dynamicFeeTx.GasFeeCap,
		GasTipCap: dynamicFeeTx.GasTipCap,
		Data:      dynamicFeeTx.Data,
		V:         dynamicFeeTx.V,
		R:         dynamicFeeTx.R,
		S:         dynamicFeeTx.S,
	})

	fmt.Printf("txid (hex) = %x\n", tx.Hash().Bytes())

	// build msg (hash of data) before sign
	singer := types.NewLondonSigner(dynamicFeeTx.ChainID)
	hash := singer.Hash(tx)
	fmt.Printf("hash before ecdsa sign (hex) = %x\n", hash.Bytes())

	pubkeyBytes, err := RecoverPubkey(dynamicFeeTx.V, dynamicFeeTx.R, dynamicFeeTx.S, hash.Bytes())
	checkErr(err)
	fmt.Printf("uncompressed public key of sender (hex) = %x\n", pubkeyBytes)

	// convert uncompressed public key to ecdsa.PublicKey
	pubkey, err := crypto.UnmarshalPubkey(pubkeyBytes)
	checkErr(err)

	// extract address from ecdsa.PublicKey
	addr := crypto.PubkeyToAddress(*pubkey)
	fmt.Printf("sender = %s\n", addr.Hex())
}

func decodeEip7702(transactionType int, transactionPayload string) {
	var setCodeTx *types.SetCodeTx
	rawTxBytes, _ := hex.DecodeString(transactionPayload)
	err := rlp.DecodeBytes(rawTxBytes, &setCodeTx)
	if err != nil {
		panic("rlp decode failed, may not a valid eth raw transaction")
	}

	fmt.Printf("basic info:\n")
	fmt.Printf("type = eip7702, i.e. TxnType = %v\n", transactionType)
	fmt.Printf("chainId = %s (0x%s)\n", setCodeTx.ChainID.String(), hex.EncodeToString(setCodeTx.ChainID.Bytes()))
	fmt.Printf("nonce = %d (0x%x)\n", setCodeTx.Nonce, setCodeTx.Nonce)
	fmt.Printf("maxPriorityFeePerGas = %s (0x%s), i.e. %s Gwei\n", setCodeTx.GasTipCap.String(), hex.EncodeToString(setCodeTx.GasTipCap.Bytes()), wei2Other(bigIntToDecimal(setCodeTx.GasTipCap.ToBig()), unitGwei).String())
	fmt.Printf("maxFeePerGas = %s (0x%s), i.e. %s Gwei\n", setCodeTx.GasFeeCap.String(), hex.EncodeToString(setCodeTx.GasFeeCap.Bytes()), wei2Other(bigIntToDecimal(setCodeTx.GasFeeCap.ToBig()), unitGwei).String())
	fmt.Printf("gasLimit = %d (0x%x)\n", setCodeTx.Gas, setCodeTx.Gas)
	fmt.Printf("to = %s\n", setCodeTx.To.String())
	fmt.Printf("value = %s (0x%s)\n", setCodeTx.Value.String(), hex.EncodeToString(setCodeTx.Value.Bytes()))
	fmt.Printf("data (hex) = %x\n", setCodeTx.Data)
	printEip7702AuthList(setCodeTx.AuthList)
	fmt.Printf("accessList = %v\n", setCodeTx.AccessList)
	fmt.Printf("yParity (ecdsa recovery id) = %s (0x%s)\n", setCodeTx.V, hex.EncodeToString(setCodeTx.V.Bytes()))
	fmt.Printf("r (hex) = %064x\n", setCodeTx.R)
	fmt.Printf("s (hex) = %064x\n", setCodeTx.S)

	fmt.Printf("\n")
	fmt.Printf("derived info:\n")
	// fmt.Printf("hash before ecdsa sign (hex) = %x\n", setCodeTx.)

	tx := types.NewTx(setCodeTx)
	fmt.Printf("txid (hex) = %x\n", tx.Hash().Bytes())

	// build msg (hash of data) before sign
	singer := types.NewPragueSigner(setCodeTx.ChainID.ToBig())
	hash := singer.Hash(tx)
	fmt.Printf("hash before ecdsa sign (hex) = %x\n", hash.Bytes())

	pubkeyBytes, err := RecoverPubkey(setCodeTx.V.ToBig(), setCodeTx.R.ToBig(), setCodeTx.S.ToBig(), hash.Bytes())
	checkErr(err)
	fmt.Printf("uncompressed public key of sender (hex) = %x\n", pubkeyBytes)

	// convert uncompressed public key to ecdsa.PublicKey
	pubkey, err := crypto.UnmarshalPubkey(pubkeyBytes)
	checkErr(err)

	// extract address from ecdsa.PublicKey
	addr := crypto.PubkeyToAddress(*pubkey)
	fmt.Printf("sender = %s\n", addr.Hex())
}

func printEip7702AuthList(authList []types.SetCodeAuthorization) {
	fmt.Printf("======================================== EIP7702 authList Begin ========================================\n")
	for _, auth := range authList {
		fmt.Printf("chainId = %s (0x%s)\n", auth.ChainID.String(), hex.EncodeToString(auth.ChainID.Bytes()))
		fmt.Printf("address (delegation designator) = %s\n", auth.Address.String())
		fmt.Printf("nonce = %d (0x%x)\n", auth.Nonce, auth.Nonce)
		fmt.Printf("yParity (ecdsa recovery id) = %d (0x%x)\n", auth.V, auth.V)
		fmt.Printf("r (hex) = %064x\n", &auth.R)
		fmt.Printf("s (hex) = %064x\n", &auth.S)
		fmt.Printf("authorityAddress (derived from signature) = %s\n", recoverAuthority(auth).Hex())
	}
	fmt.Printf("======================================== EIP7702 authList End   ========================================\n")
}

func recoverAuthority(auth types.SetCodeAuthorization) common.Address {
	sighash := sigHash(auth)

	// Create ECDSA signature in [R || S || V] format
	signature := make([]byte, 65)
	rBytes := auth.R.Bytes()
	sBytes := auth.S.Bytes()

	// Pad R and S to 32 bytes each
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)

	// Set recovery ID as the last byte
	signature[64] = auth.V

	pubkeyBytes, err := crypto.Ecrecover(sighash[:], signature)
	if err != nil {
		panic(err)
	}
	fmt.Printf("sighash = %x\n", sighash[:])
	fmt.Printf("authorityPubKey (derived from signature) = %x\n", pubkeyBytes)

	// convert uncompressed public key to ecdsa.PublicKey
	pubkey, err := crypto.UnmarshalPubkey(pubkeyBytes)
	checkErr(err)

	return crypto.PubkeyToAddress(*pubkey)
}

// sigHash calculates the pre signature hash for a SetCodeAuthorization.
func sigHash(auth types.SetCodeAuthorization) common.Hash {
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-7702.md#parameters
	MAGIC := byte(0x05)
	return prefixedRlpHash(MAGIC, []any{
		auth.ChainID,
		auth.Address,
		auth.Nonce,
	})
}

// prefixedRlpHash writes the prefix into the hasher before rlp-encoding x.
// It's used for typed transactions.
func prefixedRlpHash(prefix byte, x interface{}) common.Hash {
	sha := sha3.NewLegacyKeccak256()
	sha.Reset()
	sha.Write([]byte{prefix})
	err := rlp.Encode(sha, x)
	if err != nil {
		panic(err)
	}
	result := sha.Sum(nil)
	return common.BytesToHash(result)
}
