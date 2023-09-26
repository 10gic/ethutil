package cmd

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var eip712TypedDataFile string

func init() {
	eip712SignCmd.Flags().StringVarP(&eip712TypedDataFile, "eip712-typed-data-file", "", "", "the path of EIP712 typed data json file")
}

// eip712SignCmd represents the eip712Sign command
var eip712SignCmd = &cobra.Command{
	Use:   "eip712-sign",
	Short: "Create EIP712 sign",
	Long: `Create EIP712 sign, here is an example of input json file (specified by --eip712-typed-data-file):
{
    "types": {
        "EIP712Domain": [
            {
                "name": "name",
                "type": "string"
            },
            {
                "name": "version",
                "type": "string"
            },
            {
                "name": "chainId",
                "type": "uint256"
            },
            {
                "name": "verifyingContract",
                "type": "address"
            }
        ],
        "Person": [
            {
                "name": "name",
                "type": "string"
            },
            {
                "name": "wallet",
                "type": "address"
            }
        ],
        "Mail": [
            {
                "name": "from",
                "type": "Person"
            },
            {
                "name": "to",
                "type": "Person"
            },
            {
                "name": "contents",
                "type": "string"
            }
        ]
    },
    "primaryType": "Mail",
    "domain": {
        "name": "Ether Mail",
        "version": "1",
        "chainId": 1,
        "verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
    },
    "message": {
        "from": {
            "name": "Cow",
            "wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"
        },
        "to": {
            "name": "Bob",
            "wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"
        },
        "contents": "Hello, Bob!"
    }
}
`,
	Run: func(cmd *cobra.Command, args []string) {
		if globalOptPrivateKey == "" {
			log.Fatalf("--private-key is required for this command")
		}
		if eip712TypedDataFile == "" {
			log.Fatalf("--eip712-typed-data-file is required for this command")
		}
		eip712TypedDataJson, err := os.ReadFile(eip712TypedDataFile)

		privateKey := buildPrivateKeyFromHex(globalOptPrivateKey)
		var sig []byte
		sigV, sigR, sigS, err := eip712Sign(eip712TypedDataJson, privateKey)
		checkErr(err)
		sig = append(sig, sigR...)
		sig = append(sig, sigS...)
		sig = append(sig, byte(sigV))
		fmt.Printf("Signer address: %s\nEIP712 sign v: %d\nEIP712 sign r: %s\nEIP712 sign s: %s\nEIP712 sign (rsv): %s\n",
			extractAddressFromPrivateKey(privateKey).String(),
			sigV, hexutil.Encode(sigR), hexutil.Encode(sigS), hexutil.Encode(sig))
	},
}

// eip712Sign Returns EIP712 signature data
func eip712Sign(eip712TypedDataJson []byte, privateKey *ecdsa.PrivateKey) (int, []byte, []byte, error) {
	var typedData apitypes.TypedData
	if err := json.Unmarshal(eip712TypedDataJson, &typedData); err != nil {
		return 0, nil, nil, fmt.Errorf("json.Unmarshal failed: %w", err)
	}

	preHash, err := computePreHash(typedData)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("computePreHash failed: %w", err)
	}

	signatureBytes, err := crypto.Sign(preHash.Bytes(), privateKey)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("sign failed: %w", err)
	}

	signatureBytes[64] += 27
	return int(signatureBytes[64]), signatureBytes[0:32], signatureBytes[32:64], nil
}

// computePreHash Prepare the pre hash
// https://eips.ethereum.org/EIPS/eip-712
func computePreHash(typedData apitypes.TypedData) (hash common.Hash, err error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	hash = common.BytesToHash(crypto.Keccak256(rawData))
	return
}
