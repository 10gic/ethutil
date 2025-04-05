package cmd

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
	"io"
	"log"
	"math/big"
	"net/http"
	"regexp"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/shopspring/decimal"
)

var ethBip44Path = "m/44'/60'/0'/0/0"

var ethAddressRE = regexp.MustCompile("^(0x)?[0-9a-fA-F]{40}$")

// AA EntryPoint contract address
var aaEntryPoint = common.HexToAddress("0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789")

// SingletonFactory (EIP2470) contract address
var singletonFactoryAddr = common.HexToAddress("0xce0042B868300000d44A59004Da54A005ffdcf9f")

// AA Simple Account Factory init code
var aaSimpleAccountFactoryInitCode = "608060405234801561000f575f80fd5b5061098c8061001d5f395ff3fe608060405234801561000f575f80fd5b5060043610610034575f3560e01c8063729afa2c14610038578063921bd73214610067575b5f80fd5b61004b6100463660046101e6565b61007a565b6040516001600160a01b03909116815260200160405180910390f35b61004b6100753660046101e6565b610108565b6040515f903090829061008f602082016101be565b601f1982820381018352601f9091011660408181526001600160a01b03888116602084015287169082015260600160408051601f19818403018152908290526100db929160200161024c565b6040516020818303038152906040528051906020012090506100fe868284610195565b9695505050505050565b5f80848484604051610119906101be565b6001600160a01b039283168152911660208201526040018190604051809103905ff590508015801561014d573d5f803e3d5ffd5b506040516001600160a01b03821681529091507f805996f252884581e2f74cf3d2b03564d5ec26ccc90850ae12653dc1b72d1fa29060200160405180910390a1949350505050565b5f604051836040820152846020820152828152600b8101905060ff815360559020949350505050565b6106ee8061026983390190565b80356001600160a01b03811681146101e1575f80fd5b919050565b5f805f606084860312156101f8575f80fd5b83359250610208602085016101cb565b9150610216604085016101cb565b90509250925092565b5f81515f5b8181101561023e5760208185018101518683015201610224565b505f93019283525090919050565b5f61026061025a838661021f565b8461021f565b94935050505056fe60a060405234801561000f575f80fd5b506040516106ee3803806106ee83398101604081905261002e9161006d565b5f80546001600160a01b0319166001600160a01b039384161790551660805261009e565b80516001600160a01b0381168114610068575f80fd5b919050565b5f806040838503121561007e575f80fd5b61008783610052565b915061009560208401610052565b90509250929050565b6080516106326100bc5f395f8181607e015261039f01526106325ff3fe608060405234801561000f575f80fd5b5060043610610034575f3560e01c80633a871cdd14610038578063b61d27f61461005d575b5f80fd5b61004b610046366004610496565b610072565b60405190815260200160405180910390f35b61007061006b3660046104e5565b610394565b005b5f336001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016146100f05760405162461bcd60e51b815260206004820152601760248201527f6163636f756e743a206e6f7420456e747279506f696e7400000000000000000060448201526064015b60405180910390fd5b6040517f19457468657265756d205369676e6564204d6573736167653a0a3332000000006020820152603c81018490525f90605c0160408051601f19818403018152919052805160209091012090505f61014e610140870187610572565b8080601f0160208091040260200160405190810160405280939291908181526020018383808284375f92019190915250508251929350505060410361027b576020810151604082015160608301515f1a7f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a08211156101d45760019550505050505061038d565b604080515f8082526020820180845288905260ff841692820192909252606081018590526080810184905260019060a0016020604051602081039080840390855afa158015610225573d5f803e3d5ffd5b5050604051601f1901519150506001600160a01b03811661024f576001965050505050505061038d565b5f546001600160a01b03828116911614610272576001965050505050505061038d565b50505050610286565b60019250505061038d565b6102936040870187610572565b90505f03610334575f8054602088013591600160a01b9091046001600160601b03169060146102c1836105bc565b91906101000a8154816001600160601b0302191690836001600160601b031602179055506001600160601b0316146103345760405162461bcd60e51b81526020600482015260166024820152756163636f756e743a20696e76616c6964206e6f6e636560501b60448201526064016100e7565b8315610387576040515f9033905f1990879084818181858888f193505050503d805f811461037d576040519150601f19603f3d011682016040523d82523d5f602084013e610382565b606091505b505050505b5f925050505b9392505050565b336001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001614806103d457505f546001600160a01b031633145b6104205760405162461bcd60e51b815260206004820181905260248201527f6163636f756e743a206e6f74204f776e6572206f7220456e747279506f696e7460448201526064016100e7565b5f80856001600160a01b031685858560405161043d9291906105ed565b5f6040518083038185875af1925050503d805f8114610477576040519150601f19603f3d011682016040523d82523d5f602084013e61047c565b606091505b50915091508161048e57805160208201fd5b505050505050565b5f805f606084860312156104a8575f80fd5b833567ffffffffffffffff8111156104be575f80fd5b840161016081870312156104d0575f80fd5b95602085013595506040909401359392505050565b5f805f80606085870312156104f8575f80fd5b84356001600160a01b038116811461050e575f80fd5b935060208501359250604085013567ffffffffffffffff80821115610531575f80fd5b818701915087601f830112610544575f80fd5b813581811115610552575f80fd5b886020828501011115610563575f80fd5b95989497505060200194505050565b5f808335601e19843603018112610587575f80fd5b83018035915067ffffffffffffffff8211156105a1575f80fd5b6020019150368190038213156105b5575f80fd5b9250929050565b5f6001600160601b038083168181036105e357634e487b7160e01b5f52601160045260245ffd5b6001019392505050565b818382375f910190815291905056fea26469706673582212206858048b60b9bd4e581d3cc8719c728773e25511f4e7382368255c9b1cacbcbe64736f6c63430008150033a2646970667358221220f31ae80aa634b5c8f73c4ad48b0a980cbfbfaec75f752ad66ecf1033bfaa9ebc64736f6c63430008150033"

// AA Simple Account Factory salt
var aaSimpleAccountFactorySalt = "0x1c8aa5fdf34c07f6579b7fb620f073e48f45f0f405395f59205c0ad10bf0decc" // keccak_256(ethutil)

// getAASimpleAccountFactoryAddress returns address of AA Simple Account Factory
// AA Simple Account Factory contract address
// create2 arguments:
// deploy address: 0xce0042B868300000d44A59004Da54A005ffdcf9f (SingletonFactory (EIP2470) contract address)
// init code: aaSimpleAccountFactoryInitCode
// salt: aaSimpleAccountFactorySalt
func getAASimpleAccountFactoryAddress() common.Address {
	salt, _ := hexutil.Decode(aaSimpleAccountFactorySalt)
	var salt32 [32]byte
	copy(salt32[:], salt)
	return crypto.CreateAddress2(singletonFactoryAddr, salt32, crypto.Keccak256(common.FromHex(aaSimpleAccountFactoryInitCode)))
}

// contains returns true if array arr contains str.
func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

// checkErr panic if err != nil.
func checkErr(err error) {
	if err != nil {
		if rpcErr, ok := err.(rpc.DataError); ok {
			var errData = rpcErr.ErrorData()
			log.Printf("error: %s, data field in error: %v", err, errData)
			if errData != nil {
				if errStr, ok := errData.(string); ok && len(errStr) >= 10 {
					var funcHash = errStr[0:10]
					funcSig, err := GetFuncSig(funcHash)
					if err != nil {
						log.Printf("getFuncSig failed %v", err)
					}
					for _, data := range funcSig {
						log.Printf("%s is signature of %s", funcHash, data)
					}
				}
			}
		}
		log.Fatalf("%v", err)
		// panic(err)
	}
}

// isValidEthAddress returns true if v is a valid eth address.
func isValidEthAddress(v string) bool {
	return ethAddressRE.MatchString(v)
}

// isContractAddress returns true if address is a valid eth contract address.
func isContractAddress(client *ethclient.Client, address common.Address) (bool, error) {
	bytecode, err := client.CodeAt(context.Background(), address, nil) // nil is latest block
	if err != nil {
		return false, err
	}

	isContract := len(bytecode) > 0
	return isContract, nil
}

// has0xPrefix returns true if str starts with 0x or 0X.
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

// remove0xPrefix remove 0x prefix if
func remove0xPrefix(str string) string {
	if has0xPrefix(str) {
		return str[2:]
	}
	return str
}

// isValidHexString returns true if str is a valid hex string or empty string.
func isValidHexString(str string) bool {
	if str == "" {
		return true
	}
	_, err := hex.DecodeString(remove0xPrefix(str))
	if err != nil {
		return false
	}

	return true
}

// bigIntToDecimal converts x from big.Int to decimal.Decimal.
func bigIntToDecimal(x *big.Int) decimal.Decimal {
	if x == nil {
		return decimal.New(0, 0)
	}
	return decimal.NewFromBigInt(x, 0)
}

// hexToPrivateKey builds ecdsa.PrivateKey from hex string (the leading 0x is optional),
// it would panic if input an invalid hex string.
func hexToPrivateKey(privateKeyHex string) *ecdsa.PrivateKey {
	privateKeyHex = remove0xPrefix(privateKeyHex)
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		panic(fmt.Sprintf("parse private key failed: %s", err))
	}

	return privateKey
}

// hexToPublicKey builds ecdsa.PublicKey from hex string (the leading 0x is optional),
// it would panic if input an invalid hex string.
func hexToPublicKey(publicKeyHex string) (*ecdsa.PublicKey, error) {
	publicKeyHex = remove0xPrefix(publicKeyHex)

	// Convert the hexadecimal string to a byte slice
	pubKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string: %v", err)
	}

	uncompressedPubKey := make([]byte, 65)
	if len(pubKeyBytes) == 65 {
		// In the case of a uncompressed public key, directly use it
		uncompressedPubKey = pubKeyBytes
	} else {
		// In the case of a compressed public key, decompress it to x, y coordinates
		x, y := secp256k1.DecompressPubkey(pubKeyBytes)
		if x == nil {
			return nil, fmt.Errorf("failed to decompress public key")
		}

		// Convert the x, y coordinates to a uncompressed public key
		uncompressedPubKey[0] = 0x04
		x.FillBytes(uncompressedPubKey[1:33])
		y.FillBytes(uncompressedPubKey[33:])
	}

	// Parse the uncompressed public key
	return crypto.UnmarshalPubkey(uncompressedPubKey)
}

// wei2Other converts wei to other unit (specified by targetUnit).
func wei2Other(sourceAmtInWei decimal.Decimal, targetUnit string) decimal.Decimal {
	decimal.DivisionPrecision = 18
	if targetUnit == unitWei {
		return sourceAmtInWei
	} else if targetUnit == unitGwei {
		return sourceAmtInWei.Div(decimal.NewFromBigInt(big.NewInt(1), 9))
	} else if targetUnit == unitEther {
		return sourceAmtInWei.Div(decimal.NewFromBigInt(big.NewInt(1), 18))
	} else {
		panic(fmt.Sprintf("unrecognized unit %v", targetUnit))
	}
}

// unify2Wei converts any unit (specified by sourceUnit) to wei.
func unify2Wei(sourceAmt decimal.Decimal, sourceUnit string) decimal.Decimal {
	if sourceUnit == unitWei {
		return sourceAmt
	} else if sourceUnit == unitGwei {
		return sourceAmt.Mul(decimal.NewFromBigInt(big.NewInt(1), 9))
	} else if sourceUnit == unitEther {
		return sourceAmt.Mul(decimal.NewFromBigInt(big.NewInt(1), 18))
	} else {
		panic(fmt.Sprintf("unrecognized unit %v", sourceUnit))
	}
}

// extractAddressFromPrivateKey extracts address from ecdsa.PrivateKey.
func extractAddressFromPrivateKey(privateKey *ecdsa.PrivateKey) common.Address {
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		panic("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	return crypto.PubkeyToAddress(*publicKeyECDSA)
}

// getTxReceipt gets the receipt of tx, re-check util timeout if tx not found.
func getTxReceipt(client *ethclient.Client, txHash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	var beginTime = time.Now()

recheck:
	if rp, err := client.TransactionReceipt(context.Background(), txHash); err != nil {
		if err == ethereum.NotFound {
			log.Printf("tx %v not found (may be pending) in network", txHash.String())
		} else {
			return nil, fmt.Errorf("TransactionReceipt fail: %w", err)
		}
	} else {
		// no error
		return rp, nil
	}

	if timeout > 0 && beginTime.Add(timeout).After(time.Now()) {
		// timeout
		return nil, fmt.Errorf("GetReceipt timeout")
	}

	// not timeout
	log.Printf("re-check tx %v after 5 seconds", txHash.String())
	time.Sleep(time.Second * 5)
	goto recheck
}

const EthGasStationUrl = "https://ethgasstation.info/json/ethgasAPI.json"

// GasStationPrice the struct of response of EthGasStationUrl
type GasStationPrice struct {
	Fast        float64
	Fastest     float64
	SafeLow     float64
	Average     float64
	SafeLowWait float64
	AvgWait     float64
	FastWait    float64
	FastestWait float64
}

// getGasPrice, get gas price from EthGasStationUrl, built-in method client.SuggestGasPrice is not good enough.
func getGasPriceFromEthGasStation() (*big.Int, error) {
	var gasStationPrice GasStationPrice
	resp, err := http.Get(EthGasStationUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &gasStationPrice)
	if err != nil {
		return nil, err
	}

	// we use `fast`
	gasPrice := big.NewInt(int64(gasStationPrice.Fast * 100000000))
	return gasPrice, nil
}

// GenRawTx return raw tx, a hex string with 0x prefix
func GenRawTx(signedTx *types.Transaction) (string, error) {
	data, err := signedTx.MarshalBinary()
	if err != nil {
		return "", err
	}

	return hexutil.Encode(data), nil
}

// GetRawTx get raw tx from rpc node
func GetRawTx(rpcClient *rpc.Client, txHash string) (string, error) {
	if !has0xPrefix(txHash) {
		// Add 0x prefix, as eth_getRawTransactionByHash need it
		txHash = "0x" + txHash
	}
	var result hexutil.Bytes
	err := rpcClient.CallContext(context.Background(), &result, "eth_getRawTransactionByHash", txHash)
	if err != nil {
		return "", err
	}

	return hexutil.Encode(result), nil
}

// SendSignedTx broadcast signed tx and return tx returned by rpc node
func SendSignedTx(rpcClient *rpc.Client, signedTx *types.Transaction) (*common.Hash, error) {
	rawTx, err := GenRawTx(signedTx)
	if err != nil {
		return nil, err
	}

	return SendRawTransaction(rpcClient, rawTx)
}

// SendRawTransaction broadcast signedTxHexStr and return tx returned by rpc node
func SendRawTransaction(rpcClient *rpc.Client, signedTxHexStr string) (*common.Hash, error) {
	var result hexutil.Bytes
	err := rpcClient.CallContext(context.Background(), &result, "eth_sendRawTransaction", signedTxHexStr)
	if err != nil {
		return nil, err
	}

	var hash = common.HexToHash(hexutil.Encode(result))
	return &hash, nil
}

// BuildTx builds transaction
func BuildTx(client *ethclient.Client, privateKey *ecdsa.PrivateKey, fromAddress, /* fromAddress is only needed when privateKey is nil */
	toAddress *common.Address, amount, gasPrice *big.Int, data []byte,
) (*types.Transaction, error) {
	var nonce uint64
	var err error
	if globalOptNonce < 0 {

		var account common.Address
		if privateKey != nil {
			account = extractAddressFromPrivateKey(privateKey)
		} else {
			account = *fromAddress
		}

		nonce, err = client.PendingNonceAt(context.Background(), account)
		if err != nil {
			return nil, fmt.Errorf("PendingNonceAt fail: %w", err)
		}
	} else {
		nonce = uint64(globalOptNonce)
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

	var tx *types.Transaction

	if globalOptTxType == txTypeEip1559 {
		var maxFeePerGasEstimate = new(big.Int)
		var maxPriorityFeePerGasEstimate = new(big.Int)
		var err error
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

		tx = types.NewTx(&types.DynamicFeeTx{
			Nonce:     nonce,
			To:        toAddress, // nil means contract creation
			Value:     amount,
			Gas:       gasLimit,
			GasTipCap: maxPriorityFeePerGas,
			GasFeeCap: maxFeePerGas,
			Data:      data,
		})
	} else {
		// if not specified
		if gasPrice == nil {
			gasPrice, err = getGasPrice(globalClient.EthClient)
			checkErr(err)
		}

		tx = types.NewTx(&types.LegacyTx{
			Nonce:    nonce,
			To:       toAddress, // nil means contract creation
			Value:    amount,
			Gas:      gasLimit,
			GasPrice: gasPrice,
			Data:     data,
		})
	}

	return tx, err
}

// BuildSignedTx builds signed transaction
func BuildSignedTx(
	client *ethclient.Client, privateKey *ecdsa.PrivateKey, fromAddress, /* fromAddress is only needed when privateKey is nil */
	toAddress *common.Address, amount, gasPrice *big.Int, data []byte, sigData []byte,
) (*types.Transaction, error) {
	tx, err := BuildTx(client, privateKey, fromAddress, toAddress, amount, gasPrice, data)
	if err != nil {
		return nil, fmt.Errorf("BuildTx fail: %w", err)
	}

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("NetworkID fail: %w", err)
	}

	signer := types.NewLondonSigner(chainID)

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

// Transact invokes the (paid) contract method.
func Transact(rpcClient *rpc.Client, client *ethclient.Client, privateKey *ecdsa.PrivateKey, toAddress *common.Address, amount *big.Int, gasPrice *big.Int, data []byte) (string, error) {
	fromAddress := extractAddressFromPrivateKey(privateKey)

	signedTx, err := BuildSignedTx(client, privateKey, &fromAddress, toAddress, amount, gasPrice, data, nil)
	if err != nil {
		return "", fmt.Errorf("BuildSignedTx fail: %w", err)
	}

	if globalOptShowRawTx {
		rawTx, err := GenRawTx(signedTx)
		if err != nil {
			return "", fmt.Errorf("GenRawTx fail: %w", err)
		}
		log.Printf("raw tx = %v", rawTx)
	}

	if globalOptShowEstimateGas {
		// EstimateGas
		msg := ethereum.CallMsg{
			From:     fromAddress,
			To:       toAddress,
			Gas:      signedTx.Gas(),
			GasPrice: gasPrice,
			Value:    amount,
			Data:     data,
		}
		gas, err := client.EstimateGas(context.Background(), msg)
		if err != nil {
			return "", fmt.Errorf("EstimateGas fail: %w", err)
		}
		log.Printf("estimate gas = %v", gas)
	}

	if globalOptDryRun {
		// return tx directly, do not broadcast it
		return signedTx.Hash().String(), nil
	}

	rpcReturnTx, err := SendSignedTx(rpcClient, signedTx)
	if err != nil {
		return "", fmt.Errorf("SendSignedTx fail: %w", err)
	}

	if signedTx.Hash() != *rpcReturnTx {
		log.Printf("warning: tx not same. the computed tx is %v, but rpc eth_sendRawTransaction return tx %v, use the later", signedTx.Hash(), rpcReturnTx)
	}

	if transferNotCheck {
		return rpcReturnTx.String(), nil
	}

	rp, err := getTxReceipt(client, *rpcReturnTx, 0)
	if err != nil {
		return "", fmt.Errorf("getTxReceipt fail: %w", err)
	}

	if !globalOptTerseOutput {
		// show tx explorer url only when globalOptNodeUrl in map nodeUrlMap
		for k, v := range nodeUrlMap {
			if v == globalOptNodeUrl {
				log.Printf(nodeTxExplorerUrlMap[k] + rpcReturnTx.String())
				break
			}
		}
	}

	if rp.Status != types.ReceiptStatusSuccessful {
		return "", fmt.Errorf("tx %v minted, but status is failed, please check it in block explorer", rpcReturnTx.String())
	}

	if toAddress == nil {
		log.Printf("the new contract deployed at %v", crypto.CreateAddress(fromAddress, signedTx.Nonce()))
	}

	return rpcReturnTx.String(), nil
}

// getEIP1559GasPrice returns maxFeePerGasEstimate and maxPriorityFeePerGasEstimate
// See https://github.com/stackup-wallet/userop.js/blob/148b5abbc9fb4f570e87f9d41d7971560098406e/src/preset/middleware/gasPrice.ts#L4
func getEIP1559GasPrice(client *ethclient.Client) (*big.Int, *big.Int, error) {
	tip, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		return nil, nil, fmt.Errorf("SuggestGasTipCap failed: %w", err)
	}

	// buffer = tip * 13%
	buffer := new(big.Int).Div(tip, big.NewInt(100))
	buffer = new(big.Int).Mul(buffer, big.NewInt(13))

	// maxPriorityFeePerGas = tip * 113%
	maxPriorityFeePerGas := new(big.Int).Add(tip, buffer)

	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("HeaderByNumber failed: %w", err)
	}
	baseFee := header.BaseFee
	var maxFeePerGas *big.Int
	if baseFee == nil {
		maxFeePerGas = maxPriorityFeePerGas
	} else {
		// maxFeePerGas = 2 * baseFee + maxPriorityFeePerGas
		baseFeeMul2 := new(big.Int).Mul(baseFee, big.NewInt(2))
		maxFeePerGas = new(big.Int).Add(baseFeeMul2, maxPriorityFeePerGas)
		//log.Printf("baseFee %s", baseFee.String())
		//log.Printf("baseFeeMul2 %s", baseFeeMul2.String())
		//log.Printf("maxPriorityFeePerGas %s", maxPriorityFeePerGas.String())
		//log.Printf("maxFeePerGas %s", maxFeePerGas.String())
	}

	return maxFeePerGas, maxPriorityFeePerGas, nil
}

// getEIP1559GasPriceByFeeHistory returns maxFeePerGasEstimate and maxPriorityFeePerGasEstimate
func getEIP1559GasPriceByFeeHistory(client *ethclient.Client) (*big.Int, *big.Int, error) {
	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("HeaderByNumber failed: %w", err)
	}

	var maxFeePerGasEstimate = new(big.Int)
	var maxPriorityFeePerGasEstimate = new(big.Int)

	// Use rpc eth_feeHistory to estimate default maxPriorityFeePerGas and maxFeePerGas
	// See https://docs.alchemy.com/docs/how-to-build-a-gas-fee-estimator-using-eip-1559
	//
	// $ curl -X POST --data '{ "id": 1, "jsonrpc": "2.0", "method": "eth_feeHistory", "params": ["0x4", "latest", [5, 50, 95]] }' https://mainnet.infura.io/v3/21a9f5ba4bce425795cac796a66d7472
	// {
	//  "jsonrpc": "2.0",
	//  "id": 1,
	//  "result": {
	//    "baseFeePerGas": [
	//      "0x4ed3ef336",
	//      "0x4d2c282cd",
	//      "0x4db586991",
	//      "0x4d8275e8e",
	//      "0x4b5fb0a47"
	//    ],
	//    "gasUsedRatio": [
	//      0.41600023333333336,
	//      0.5278128666666667,
	//      0.4897323,
	//      0.3897776666666667
	//    ],
	//    "oldestBlock": "0xffc0a9",
	//    "reward": [
	//      [
	//        "0x6b51f67",
	//        "0x3b9aca00",
	//        "0x106853ddd8"
	//      ],
	//      [
	//        "0xa9970dc",
	//        "0x1dcd6500",
	//        "0x10abffd64"
	//      ],
	//      [
	//        "0x6190547",
	//        "0x1dcd6500",
	//        "0x9becf3d3c"
	//      ],
	//      [
	//        "0x94a104a",
	//        "0x1dcd6500",
	//        "0x1032d8cdb"
	//      ]
	//    ]
	//  }
	// }
	feeHistory, err := client.FeeHistory(context.Background(), 4, nil, []float64{5, 50, 95})
	if err != nil {
		return nil, nil, fmt.Errorf("FeeHistory failed: %w", err)
	}
	//var slow big.Int
	//slow.Add(feeHistory.Reward[0][0], feeHistory.Reward[1][0])
	//slow.Add(&slow, feeHistory.Reward[2][0])
	//slow.Div(&slow, big.NewInt(3))

	// log.Printf("feeHistory = %+v", feeHistory)

	if len(feeHistory.Reward) == 1 {
		maxPriorityFeePerGasEstimate = feeHistory.Reward[0][2] // Use the fastest value (95 percentile)
		maxFeePerGasEstimate = maxFeePerGasEstimate.Add(header.BaseFee, maxPriorityFeePerGasEstimate)
		return maxFeePerGasEstimate, maxPriorityFeePerGasEstimate, nil
	}

	var average big.Int
	average.Add(feeHistory.Reward[0][1], feeHistory.Reward[1][1])
	average.Add(&average, feeHistory.Reward[2][1])
	average.Div(&average, big.NewInt(3))

	//var fast big.Int
	//fast.Add(feeHistory.Reward[0][2], feeHistory.Reward[1][2])
	//fast.Add(&fast, feeHistory.Reward[2][2])
	//fast.Div(&fast, big.NewInt(3))

	// Currently, slow/fast are not used. we use average value
	maxPriorityFeePerGasEstimate = &average
	// log.Printf("maxPriorityFeePerGasEstimate = %v", maxPriorityFeePerGasEstimate.String())

	maxFeePerGasEstimate = maxFeePerGasEstimate.Add(header.BaseFee, maxPriorityFeePerGasEstimate)
	// log.Printf("maxFeePerGasEstimate = %v", maxFeePerGasEstimate.String())

	return maxFeePerGasEstimate, maxPriorityFeePerGasEstimate, nil
}

// Call invokes the (constant) contract method.
func Call(rpcClient *rpc.Client, toAddress common.Address, data []byte) ([]byte, error) {
	opts := new(bind.CallOpts)
	msg := ethereum.CallMsg{From: opts.From, To: &toAddress, Data: data}

	var result hexutil.Bytes
	err := rpcClient.CallContext(context.Background(), &result, "eth_call", toCallArg(msg), "latest")
	if err != nil {
		return nil, err
	}

	return result, nil
}

// toCallArg build call argument
//
// Similar with func toCallArg in https://github.com/ethereum/go-ethereum/blob/14eb8967be7acc54c5dc9a416151ac45c01251b6/ethclient/ethclient.go#L642
// The only difference: we use 'data' field, rather than 'input' field
//
// Background:
// The documentation of rpc api has changed the name of the parameter from 'data' to 'input',
// See https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_call
//
// Ethereum support both 'data' and 'input', but some network (Tron) only support the old field 'data',
// See https://developers.tron.network/reference/eth_call
// So, 'data' has better compatibility
func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		// For compatibility, we use 'data', rather than 'input'
		// See: https://github.com/ethereum/go-ethereum/issues/28608
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	if msg.GasFeeCap != nil {
		arg["maxFeePerGas"] = (*hexutil.Big)(msg.GasFeeCap)
	}
	if msg.GasTipCap != nil {
		arg["maxPriorityFeePerGas"] = (*hexutil.Big)(msg.GasTipCap)
	}
	if msg.AccessList != nil {
		arg["accessList"] = msg.AccessList
	}
	if msg.BlobGasFeeCap != nil {
		arg["maxFeePerBlobGas"] = (*hexutil.Big)(msg.BlobGasFeeCap)
	}
	if msg.BlobHashes != nil {
		arg["blobVersionedHashes"] = msg.BlobHashes
	}
	return arg
}

// getRecoveryId gets ecdsa recover id (0 or 1) from v.
func getRecoveryId(v *big.Int) int {
	// Note: can be simplified by checking parity (i.e. odd-even)
	var recoveryId int
	if v.Int64() == 0 || v.Int64() == 1 { // v in eip2718
		recoveryId = int(v.Int64())
	} else if v.Int64() == 27 || v.Int64() == 28 { // v before eip155
		recoveryId = int(v.Int64()) - 27
	} else { // v in eip155
		// derive chainId
		var chainId = int((v.Int64() - 35) / 2)
		// derive recoveryId
		recoveryId = int(v.Int64()) - 35 - 2*chainId
	}
	return recoveryId
}

// buildECDSASignature builds a 65-byte compact ECDSA signature (containing the recovery id as the last element)
func buildECDSASignature(v, r, s *big.Int) []byte {
	var recoveryId = getRecoveryId(v)

	var rBytes = make([]byte, 32, 32)
	var sBytes = make([]byte, 32, 32)
	copy(rBytes[32-len(r.Bytes()):], r.Bytes())
	copy(sBytes[32-len(s.Bytes()):], s.Bytes())

	var rsBytes = append(rBytes, sBytes...)
	return append(rsBytes, byte(recoveryId))
}

// RecoverPubkey recover public key, returns 65 bytes uncompressed public key
func RecoverPubkey(v, r, s *big.Int, msg []byte) ([]byte, error) {
	signature := buildECDSASignature(v, r, s)

	// recover public key from msg (hash of data) and ECDSA signature
	// crypto.Ecrecover msg: 32 bytes hash
	// crypto.Ecrecover signature: 65-byte compact ECDSA signature
	// crypto.Ecrecover return 65 bytes uncompressed public key
	return crypto.Ecrecover(msg, signature)
}

// GetFuncSig recover function signature from 4 bytes hash
func GetFuncSig(funcHash string) ([]string, error) {
	sigs, err := GetFuncSigFromOpenchain(funcHash)
	if err != nil || len(sigs) == 0 {
		sigs, err = GetFuncSigFrom4Byte(funcHash)
	}
	return sigs, err
}

// GetFuncSig recover function signature from 4 bytes hash
// For example:
//
//	param: "0x8c905368"
//	return: ["NotEnoughFunds(uint256,uint256)"]
//
// This function uses openchain API
// $ curl -X 'GET' 'https://api.openchain.xyz/signature-database/v1/lookup?function=0x8c905368&filter=true'
// {"ok":true,"result":{"event":{},"function":{"0x8c905368":[{"name":"NotEnoughFunds(uint256,uint256)","filtered":false}]}}}
// See https://openchain.xyz/signatures
func GetFuncSigFromOpenchain(funcHash string) ([]string, error) {
	var url = fmt.Sprintf("https://api.openchain.xyz/signature-database/v1/lookup?function=%s&filter=true", funcHash)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	type funcSig struct {
		Name     string `json:"name"`
		Filtered bool   `json:"filtered"`
	}
	type respMsg struct {
		Ok     bool `json:"ok"`
		Result struct {
			Function map[string][]funcSig `json:"function"`
		} `json:"result"`
	}
	var data respMsg
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var rc []string
	for _, data := range data.Result.Function[funcHash] {
		rc = append(rc, data.Name)
	}

	return rc, nil
}

// GetFuncSigFrom4Byte recover function signature from 4 bytes hash from 4byte API
// For example:
//
//	param: "0x8c905368"
//	return: ["NotEnoughFunds(uint256,uint256)"]
//
// This function uses 4byte API
// $ curl -X 'GET' 'https://www.4byte.directory/api/v1/signatures/?hex_signature=0x275fb869'
//
//	{
//	 "count": 1,
//	 "next": null,
//	 "previous": null,
//	 "results": [
//	   {
//	     "id": 1136703,
//	     "created_at": "2025-03-22T17:59:30.142145Z",
//	     "text_signature": "InsufficientSlippage(uint256,uint256)",
//	     "hex_signature": "0x275fb869",
//	     "bytes_signature": "'_¸i"
//	   }
//	 ]
//	}
//
// See https://www.4byte.directory/docs/
func GetFuncSigFrom4Byte(funcHash string) ([]string, error) {
	var url = fmt.Sprintf("https://www.4byte.directory/api/v1/signatures/?hex_signature=%s", funcHash)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	type respMsg struct {
		Count  uint64 `json:"count"`
		Result []struct {
			ID             uint64 `json:"id"`
			CreatedAt      string `json:"created_at"`
			TextSignature  string `json:"text_signature"`
			HexSignature   string `json:"hex_signature"`
			BytesSignature string `json:"bytes_signature"`
		} `json:"results"`
	}
	var data respMsg
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var rc []string
	if data.Count == 0 {
		// not found
		return rc, nil
	}

	for _, data := range data.Result {
		rc = append(rc, data.TextSignature)
	}
	return rc, nil
}

// MnemonicToPrivateKey generate private key from mnemonic words
func MnemonicToPrivateKey(mnemonic string, derivationPath string) (*ecdsa.PrivateKey, error) {
	// Generate a Bip32 HD wallet for the mnemonic and a user supplied password
	seed := bip39.NewSeed(mnemonic, "")
	// Generate a new master node using the seed.
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, err
	}

	childIdxs, err := parseDerivationPath(derivationPath)
	if err != nil {
		return nil, err
	}

	currentKey := masterKey
	for _, childIdx := range childIdxs {
		currentKey, err = currentKey.NewChildKey(childIdx)
		if err != nil {
			return nil, err
		}
	}

	privateKeyBytes := currentKey.Key // 32 bytes private key

	privateKey := hexToPrivateKey(hexutil.Encode(privateKeyBytes))
	return privateKey, nil
}
