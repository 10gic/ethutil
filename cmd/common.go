package cmd

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func isValidEthAddress(v string) bool {
	var ethAddressRE = regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return ethAddressRE.MatchString(v)
}

func isContractAddress(client *ethclient.Client, address common.Address) (bool, error) {
	bytecode, err := client.CodeAt(context.Background(), address, nil) // nil is latest block
	if err != nil {
		return false, err
	}

	isContract := len(bytecode) > 0
	return isContract, nil
}

func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

func isValidHexString(str string) bool {
	if str == "" {
		return true
	}
	var hexWithout0x = str
	if has0xPrefix(str) {
		hexWithout0x = str[2:]
	}
	_, err := hex.DecodeString(hexWithout0x)
	if err != nil {
		return false
	}

	return true
}

func bigInt2Decimal(x *big.Int) decimal.Decimal {
	if x == nil {
		return decimal.New(0, 0)
	}
	return decimal.NewFromBigInt(x, 0)
}

func buildPrivateKeyFromHex(privateKeyHex string) *ecdsa.PrivateKey {
	if strings.HasPrefix(privateKeyHex, "0x") {
		privateKeyHex = privateKeyHex[2:] // remove leading 0x
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	checkErr(err)

	return privateKey
}

func wei2Other(sourceAmtInWei decimal.Decimal, targetUnit string) decimal.Decimal {
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

func extractAddressFromPrivateKey(privateKey *ecdsa.PrivateKey) common.Address {
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		panic("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	return crypto.PubkeyToAddress(*publicKeyECDSA)
}

func getReceipt(client *ethclient.Client, txHash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	var beginTime = time.Now()

recheck:
	if rp, err := client.TransactionReceipt(context.Background(), txHash); err != nil {
		if err == ethereum.NotFound {
			log.Printf("tx %v not found (may be pending) in ethereum", txHash.String())
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
	log.Printf("re-check after 10 seconds")
	time.Sleep(time.Second * 10)
	goto recheck
}

const EthGasstationUrl = "https://ethgasstation.info/json/ethgasAPI.json"

// GasStationPrice, the struct of response of EthGasstationUrl
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

// getGasPrice, get gas price from ethgasstation, built-in method client.SuggestGasPrice is not good enough.
func getGasPriceFromEthgasstation() (*big.Int, error) {
	var gasStationPrice GasStationPrice
	resp, err := http.Get(EthGasstationUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &gasStationPrice)
	if err != nil {
		return nil, err
	}

	// we use Average
	gasPrice := big.NewInt(int64(gasStationPrice.Average * 100000000))
	return gasPrice, nil
}

// Transact invokes the (paid) contract method.
func Transact(client *ethclient.Client, privateKey *ecdsa.PrivateKey, toAddress *common.Address, amount *big.Int, gasPrice *big.Int, data []byte) (string, error) {
	fromAddress := extractAddressFromPrivateKey(privateKey)

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", fmt.Errorf("PendingNonceAt fail: %w", err)
	}

	gasLimit := gasLimitOpt
	if gasLimit == 0 { // if user not specified
		gasLimit = uint64(gasUsedByTransferEth)

		if toAddress == nil {
			gasLimit = 7000000 // the default gas limit for deploy contract, can be overwrite by option
		} else {
			isContract, err := isContractAddress(client, *toAddress)
			if err != nil {
				return "", fmt.Errorf("isContractAddress fail: %w", err)
			}
			if isContract { // gasUsedByTransferEth may be not enough if send to contract
				gasLimit = 900000
			}
			if len(data) > 0 { // gasUsedByTransferEth may be not enough if with payload data
				gasLimit = 900000
			}
		}
	}

	var tx *types.Transaction
	if toAddress == nil {
		// send data to null address means deploy contract
		tx = types.NewContractCreation(nonce, amount, gasLimit, gasPrice, data)
	} else {
		tx = types.NewTransaction(nonce, *toAddress, amount, gasLimit, gasPrice, data)
	}

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

	if transferNotCheck {
		return signedTx.Hash().String(), nil
	}
	//log.Printf("tx sent: %s", signedTx.Hash().Hex())

	rp, err := getReceipt(client, signedTx.Hash(), 0)
	if err != nil {
		return "", fmt.Errorf("getReceipt fail: %w", err)
	}

	if !terseOutputOpt {
		log.Printf(nodeTxLinkMap[nodeOpt] + signedTx.Hash().String())
	}
	if rp.Status != types.ReceiptStatusSuccessful {
		return "", fmt.Errorf("tx (%v) fail", signedTx.Hash().String())
	}

	return signedTx.Hash().String(), nil
}

// Call invokes the (constant) contract method.
func Call(client *ethclient.Client, toAddress common.Address, data []byte) ([]byte, error) {
	opts := new(bind.CallOpts)
	msg := ethereum.CallMsg{From: opts.From, To: &toAddress, Data: data}
	ctx := context.TODO()
	return client.CallContract(ctx, msg, nil)
}

func getRecoveryId(v *big.Int) int {
	var recoveryId int
	// Note: can be simplified by checking parity (i.e. odd-even)
	if v.Int64() == 27 || v.Int64() == 28 { // v before bip155
		recoveryId = int(v.Int64()) - 27
	} else { // v after bip155
		// derive chainId
		var chainId = int((v.Int64() - 35) / 2)
		// derive recoveryId
		recoveryId = int(v.Int64()) - 35 - 2*chainId
	}
	return recoveryId
}

// Build a 65-byte compact ECDSA signature (containing the recovery id as the last element)
func buildECDSASignature(v, r, s *big.Int) []byte {
	var recoveryId = getRecoveryId(v)
	// println("recoveryId", recoveryId)

	var r_bytes = make([]byte, 32, 32)
	var s_bytes = make([]byte, 32, 32)
	copy(r_bytes[32-len(r.Bytes()):], r.Bytes())
	copy(s_bytes[32-len(s.Bytes()):], s.Bytes())

	var rs_bytes = append(r_bytes, s_bytes...)
	return append(rs_bytes, byte(recoveryId))
}

// Recover public key, return 65 bytes uncompressed public key
func RecoverPubkey(v, r, s *big.Int, msg []byte) ([]byte, error) {
	signature := buildECDSASignature(v, r, s)

	// recover public key from msg (hash of data) and ECDSA signature
	// crypto.Ecrecover msg: 32 bytes hash
	// crypto.Ecrecover signature: 65-byte compact ECDSA signature
	// crypto.Ecrecover return 65 bytes uncompressed public key
	return crypto.Ecrecover(msg, signature)
}
