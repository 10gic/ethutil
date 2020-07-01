package cmd

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"
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
			return nil, fmt.Errorf("TransactionReceipt failL %w", err)
		}
	} else {
		// no error
		return rp, nil
	}

	if timeout > 0 && beginTime.Add(timeout).After(time.Now()){
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
