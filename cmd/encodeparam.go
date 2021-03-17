package cmd

import (
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

var encodeParamCmdABIFile string

func init() {
	encodeParamCmd.Flags().StringVarP(&encodeParamCmdABIFile, "abi-file", "", "", "the path of abi file, if this option specified, 'function signature' can be just function name")
}

var encodeParamCmd = &cobra.Command{
	Use:   "encode-param 'function signature' arg1 arg2 ...",
	Short: "Encode input arguments, it's useful when you call contract's method manually",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		funcSignature := args[0]
		inputArgData := args[1:]

		if callCmdABIFile != "" {
			abiContent, err := ioutil.ReadFile(callCmdABIFile)
			if err != nil {
				log.Fatal(err)
			}
			funcName := funcSignature
			funcSignature, err = extractFuncDefinition(string(abiContent), extractFuncName(funcName))
			checkErr(err)
		}

		txInputData, err := buildTxInputData(funcSignature, inputArgData)
		checkErr(err)

		dumpTxInputData(txInputData)

		fmt.Printf("encoded parameters (input data) = %v\n", hexutil.Encode(txInputData))
	},
}

// buildTxInputData build tx input data
func buildTxInputData(funcSignature string, inputArgData []string) ([]byte, error) {
	funcName, funcArgTypes, err := parseFuncSignature(funcSignature)
	if err != nil {
		return nil, err
	}

	functionSelector := make([]byte, 0)
	if len(funcName) > 0 {
		funcSign := funcName + "(" + strings.Join(funcArgTypes, ",") + ")"
		functionSelector = crypto.Keccak256([]byte(funcSign))[0:4]
	} else {
		// log.Printf("function name is not found, only encode arguments")
	}

	if len(funcArgTypes) != len(inputArgData) {
		return nil, fmt.Errorf("invalid input, there are %v args in signature, but %v args are provided", len(funcArgTypes), len(inputArgData))
	}
	data, err := encodeParameters(funcArgTypes, inputArgData)
	if err != nil {
		return nil, fmt.Errorf("encodeParameters fail: %v", err)
	}

	return append(functionSelector, data...), nil
}

// dumpTxInputData dump tx input data
// An example of output:
// MethodID: 0x7ff36ab5
// [0]:  0000000000000000000000000000000000000000000002bd79cff41cc68c1f27
// [1]:  0000000000000000000000000000000000000000000000000000000000000080
// [2]:  00000000000000000000000095206727fa3dd2fa32cd0bfe1fd40736b525cf11
// [3]:  0000000000000000000000000000000000000000000000000000000060517ba6
// [4]:  0000000000000000000000000000000000000000000000000000000000000002
// [5]:  000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2
// [6]:  0000000000000000000000009ed8e7c9604790f7ec589f99b94361d8aab64e5e
func dumpTxInputData(txInputData []byte) {
	if len(txInputData)%32 == 4 {
		// has function selector in txInputData
		// print function selector (first 4 bytes)
		fmt.Printf("MethodID: %v\n", hexutil.Encode(txInputData[0:4]))
		txInputData = txInputData[4:]
	}

	num := len(txInputData) / 32
	for i := 0; i <= num-1; i++ {
		fmt.Printf("[%d]:  %v\n", i, hexutil.Encode(txInputData[32*i:32*(i+1)]))
	}
}

// extractFuncName extracts function name from arg input
// input example:
// fun1
// fun1(uint256)
// function fun1
// function fun1(uint256)
func extractFuncName(input string) string {
	if strings.HasPrefix(input, "function ") {
		input = input[len("function "):] // remove leading string "function "
	}
	funcName := strings.TrimLeft(input, " ")

	leftParenthesisLoc := strings.Index(funcName, "(")
	if leftParenthesisLoc >= 0 { // ( found
		funcName := funcName[:leftParenthesisLoc] // remove all characters from char '('
		funcName = strings.TrimSpace(funcName)
	}
	return funcName
}

// parseFuncSignature parse function signature to `function name` and `function args`.
// Example 1:
// input: "function add(uint256   xx, address xx, bool xx)"
// output: "add", ["uint256", "address", "bool"], nil
//
// Example 2 (no function name):
// input: "(uint256, address, bool)"
// output: "", ["uint256", "address", "bool"], nil
func parseFuncSignature(input string) (string, []string, error) {
	if strings.HasPrefix(input, "function ") {
		input = input[len("function "):] // remove leading string "function "
	}
	input = strings.TrimLeft(input, " ")

	leftParenthesisLoc := strings.Index(input, "(")
	if leftParenthesisLoc < 0 {
		return "", nil, fmt.Errorf("char ( is not found in function signature")
	}
	funcName := input[:leftParenthesisLoc] // remove all characters from char '('
	funcName = strings.TrimSpace(funcName)

	rightParenthesisLoc := strings.Index(input, ")")
	if rightParenthesisLoc < 0 {
		return "", nil, fmt.Errorf("char ) is not found in function signature")
	}
	argsPart := input[leftParenthesisLoc+1 : rightParenthesisLoc]
	if strings.TrimSpace(argsPart) == "" {
		return funcName, nil, nil
	}
	args := strings.Split(argsPart, ",")
	for index, arg := range args {
		fields := strings.Fields(arg)
		if len(fields) == 0 {
			return "", nil, fmt.Errorf("signature `%v` invalid type missing in args", input)
		}
		args[index] = typeNormalize(fields[0]) // first field is type. for example, "uint256 xx", first field is uint256

		if len(fields) >= 2 && fields[0] == "address" && strings.HasPrefix(fields[1], "payable[") {
			// handle case:
			// f1(address payable[] memory a, uint256 b)
			// f1(address payable[3] memory a, uint256 b)
			args[index] = strings.Replace(fields[1], "payable", "address", 1) // address[] or address[3]
		}
	}

	return funcName, args, nil
}

// encodeParameters Encode parameters
// An example:
// inputArgTypes: ["uint256", "address", "bool"]
// inputArgData: ["123", "0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb", "true"]
// return: 000000000000000000000000000000000000000000000000000000000000007b0000000000000000000000008f36975cdea2e6e64f85719788c8efbbe89dfbbb0000000000000000000000000000000000000000000000000000000000000001
func encodeParameters(inputArgTypes, inputArgData []string) ([]byte, error) {
	var theTypes abi.Arguments
	var theArgData []interface{}
	theTypes, theArgData, err := buildArgumentAndData(inputArgTypes, inputArgData)
	if err != nil {
		return nil, fmt.Errorf("buildArgumentAndData fail: %s", err)
	}
	bytes, err := theTypes.Pack(theArgData...)
	if err != nil {
		return nil, fmt.Errorf("pack fail: %s", err)
	}
	return bytes, nil
}

func buildArgumentAndData(inputArgTypes, inputArgData []string) (abi.Arguments, []interface{}, error) {
	var theTypes abi.Arguments
	var theArgData []interface{}
	for index, inputType := range inputArgTypes {
		typ, err := abi.NewType(typeNormalize(inputType), "", nil)
		if err != nil {
			return nil, nil, fmt.Errorf("abi.NewType fail: %w", err)
		}
		theTypes = append(theTypes, abi.Argument{Type: typ})
		if strings.Count(inputType, "[") == 1 && strings.Count(inputType, "]") == 1 { // handle array type
			var arrayElementType string
			leftParenthesisLoc := strings.Index(inputType, "[")
			arrayElementType = inputType[:leftParenthesisLoc] // remove all characters from char '['

			var arrayOfTypes, arrayOfData []string
			arrayOfData, err := parseArrayData(inputArgData[index])
			if err != nil {
				return nil, nil, fmt.Errorf("parseArrayData fail: %w", err)
			}
			for _ = range arrayOfData {
				arrayOfTypes = append(arrayOfTypes, typeNormalize(arrayElementType)) // `address[3]`  -> `[address, address, address]`
			}

			_, datas, err := buildArgumentAndData(arrayOfTypes, arrayOfData)
			if err != nil {
				return nil, nil, fmt.Errorf("buildArgumentAndData fail: %w", err)
			}
			if arrayElementType == "string" {
				// datas ([]interface {})   --->  elementData ([]string)
				var elementData []string
				for _, data := range datas {
					elementData = append(elementData, data.(string))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "int8" {
				// datas ([]interface {})   --->  elementData ([]int8)
				var elementData []int8
				for _, data := range datas {
					elementData = append(elementData, data.(int8))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "int16" {
				var elementData []int16
				for _, data := range datas {
					elementData = append(elementData, data.(int16))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "int32" {
				var elementData []int32
				for _, data := range datas {
					elementData = append(elementData, data.(int32))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "int64" {
				var elementData []int64
				for _, data := range datas {
					elementData = append(elementData, data.(int64))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "uint8" {
				var elementData []uint8
				for _, data := range datas {
					elementData = append(elementData, data.(uint8))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "uint16" {
				var elementData []uint16
				for _, data := range datas {
					elementData = append(elementData, data.(uint16))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "uint32" {
				var elementData []uint32
				for _, data := range datas {
					elementData = append(elementData, data.(uint32))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "uint64" {
				var elementData []uint64
				for _, data := range datas {
					elementData = append(elementData, data.(uint64))
				}
				theArgData = append(theArgData, elementData)
			} else if strings.Contains(arrayElementType, "int") {
				var elementData []*big.Int
				for _, data := range datas {
					elementData = append(elementData, data.(*big.Int))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bool" {
				var elementData []bool
				for _, data := range datas {
					elementData = append(elementData, data.(bool))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "address" {
				var elementData []common.Address
				for _, data := range datas {
					elementData = append(elementData, data.(common.Address))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes" {
				var elementData [][]byte
				for _, data := range datas {
					elementData = append(elementData, data.([]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes1" {
				var elementData [][1]byte
				for _, data := range datas {
					elementData = append(elementData, data.([1]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes2" {
				var elementData [][2]byte
				for _, data := range datas {
					elementData = append(elementData, data.([2]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes3" {
				var elementData [][3]byte
				for _, data := range datas {
					elementData = append(elementData, data.([3]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes4" {
				var elementData [][4]byte
				for _, data := range datas {
					elementData = append(elementData, data.([4]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes5" {
				var elementData [][5]byte
				for _, data := range datas {
					elementData = append(elementData, data.([5]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes6" {
				var elementData [][6]byte
				for _, data := range datas {
					elementData = append(elementData, data.([6]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes7" {
				var elementData [][7]byte
				for _, data := range datas {
					elementData = append(elementData, data.([7]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes8" {
				var elementData [][8]byte
				for _, data := range datas {
					elementData = append(elementData, data.([8]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes9" {
				var elementData [][9]byte
				for _, data := range datas {
					elementData = append(elementData, data.([9]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes10" {
				var elementData [][10]byte
				for _, data := range datas {
					elementData = append(elementData, data.([10]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes11" {
				var elementData [][11]byte
				for _, data := range datas {
					elementData = append(elementData, data.([11]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes12" {
				var elementData [][12]byte
				for _, data := range datas {
					elementData = append(elementData, data.([12]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes13" {
				var elementData [][13]byte
				for _, data := range datas {
					elementData = append(elementData, data.([13]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes14" {
				var elementData [][14]byte
				for _, data := range datas {
					elementData = append(elementData, data.([14]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes15" {
				var elementData [][15]byte
				for _, data := range datas {
					elementData = append(elementData, data.([15]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes16" {
				var elementData [][16]byte
				for _, data := range datas {
					elementData = append(elementData, data.([16]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes17" {
				var elementData [][17]byte
				for _, data := range datas {
					elementData = append(elementData, data.([17]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes18" {
				var elementData [][18]byte
				for _, data := range datas {
					elementData = append(elementData, data.([18]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes19" {
				var elementData [][19]byte
				for _, data := range datas {
					elementData = append(elementData, data.([19]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes20" {
				var elementData [][20]byte
				for _, data := range datas {
					elementData = append(elementData, data.([20]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes21" {
				var elementData [][21]byte
				for _, data := range datas {
					elementData = append(elementData, data.([21]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes22" {
				var elementData [][22]byte
				for _, data := range datas {
					elementData = append(elementData, data.([22]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes23" {
				var elementData [][23]byte
				for _, data := range datas {
					elementData = append(elementData, data.([23]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes24" {
				var elementData [][24]byte
				for _, data := range datas {
					elementData = append(elementData, data.([24]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes25" {
				var elementData [][25]byte
				for _, data := range datas {
					elementData = append(elementData, data.([25]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes26" {
				var elementData [][26]byte
				for _, data := range datas {
					elementData = append(elementData, data.([26]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes27" {
				var elementData [][27]byte
				for _, data := range datas {
					elementData = append(elementData, data.([27]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes28" {
				var elementData [][28]byte
				for _, data := range datas {
					elementData = append(elementData, data.([28]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes29" {
				var elementData [][29]byte
				for _, data := range datas {
					elementData = append(elementData, data.([29]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes30" {
				var elementData [][30]byte
				for _, data := range datas {
					elementData = append(elementData, data.([30]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes31" {
				var elementData [][31]byte
				for _, data := range datas {
					elementData = append(elementData, data.([31]byte))
				}
				theArgData = append(theArgData, elementData)
			} else if arrayElementType == "bytes32" {
				var elementData [][32]byte
				for _, data := range datas {
					elementData = append(elementData, data.([32]byte))
				}
				theArgData = append(theArgData, elementData)
			} else {
				return nil, nil, fmt.Errorf("type %v not implemented in array type currently", inputType)
			}
		} else if inputType == "string" {
			theArgData = append(theArgData, inputArgData[index])
		} else if inputType == "int8" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, int8(i))
		} else if inputType == "int16" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, int16(i))
		} else if inputType == "int32" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, int32(i))
		} else if inputType == "int64" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, int64(i))
		} else if inputType == "uint8" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, uint8(i))
		} else if inputType == "uint16" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, uint16(i))
		} else if inputType == "uint32" {
			i, err := strconv.ParseInt(inputArgData[index], 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, uint32(i))
		} else if inputType == "uint64" {
			i, err := strconv.ParseUint(inputArgData[index], 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, uint64(i))
		} else if strings.Contains(inputType, "int") { // 其它包含 int 的情况，如 int24, int256, uint48, uint256, etc
			n := new(big.Int)
			n, ok := n.SetString(inputArgData[index], 10)
			if !ok {
				return nil, nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, n)

		} else if inputType == "bool" {
			if strings.EqualFold(inputArgData[index], "true") {
				theArgData = append(theArgData, true)
			} else if strings.EqualFold(inputArgData[index], "false") {
				theArgData = append(theArgData, false)
			} else {
				return nil, nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
		} else if inputType == "address" {
			theArgData = append(theArgData, common.HexToAddress(inputArgData[index]))
		} else if inputType == "bytes" {
			var inputHex = inputArgData[index]
			if strings.HasPrefix(inputArgData[index], "0x") {
				inputHex = inputArgData[index][2:]
			}
			decoded, err := hex.DecodeString(inputHex)
			if err != nil {
				return nil, nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			theArgData = append(theArgData, decoded)
		} else if strings.Contains(inputType, "bytes") { // bytes1, bytes2, ..., bytes32
			var inputHex = inputArgData[index]
			if strings.HasPrefix(inputArgData[index], "0x") {
				inputHex = inputArgData[index][2:]
			}
			decoded, err := hex.DecodeString(inputHex)
			if err != nil {
				return nil, nil, fmt.Errorf("arg (position %v) invalid, %s cannot covert to type %v", index, inputArgData[index], inputType)
			}
			if inputType == "bytes1" {
				var data [1]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes2" {
				var data [2]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes3" {
				var data [3]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes4" {
				var data [4]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes5" {
				var data [5]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes6" {
				var data [6]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes7" {
				var data [7]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes8" {
				var data [8]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes9" {
				var data [9]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes10" {
				var data [10]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes11" {
				var data [11]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes12" {
				var data [12]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes13" {
				var data [13]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes14" {
				var data [14]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes15" {
				var data [15]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes16" {
				var data [16]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes17" {
				var data [17]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes18" {
				var data [18]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes19" {
				var data [19]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes20" {
				var data [20]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes21" {
				var data [21]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes22" {
				var data [22]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes23" {
				var data [23]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes24" {
				var data [24]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes25" {
				var data [25]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes26" {
				var data [26]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes27" {
				var data [27]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes28" {
				var data [28]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes29" {
				var data [29]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes30" {
				var data [30]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes31" {
				var data [31]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else if inputType == "bytes32" {
				var data [32]byte
				copy(data[:], decoded)
				theArgData = append(theArgData, data)
			} else {
				return nil, nil, fmt.Errorf("type %v not implemented currently", inputType)
			}
		} else {
			return nil, nil, fmt.Errorf("type %v not implemented currently", inputType)
		}
	}

	return theTypes, theArgData, nil
}

// `["abc", "xyz"]`     ----> abc, xyz
// `[abc, xyz]`         ----> abc, xyz
// `["a(a,a)", "abcd"]` ----> a(a,a), abcd
// `abc, xyz`           ----> abc, xyz
func parseArrayData(input string) ([]string, error) {
	input = strings.TrimSpace(input)
	input = strings.TrimLeft(input, "[")
	input = strings.TrimRight(input, "]")
	r := csv.NewReader(strings.NewReader(input))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	records, err := r.Read()
	if err != nil {
		if err == io.EOF {
			return []string{}, nil
		} else {
			println(err.Error())
		}
	}
	return records, nil
}

// uint -> uint256
// int -> int256
// uint[] -> uint256[]
// int[] -> int256[]
func typeNormalize(input string) string {
	re := regexp.MustCompile(`\b([u]int)\b`)
	return re.ReplaceAllString(input, "${1}256")
}

// ABI example:
// [
//  {
//      "inputs": [],
//      "stateMutability": "nonpayable",
//      "type": "constructor"
//  },
//	{
//		"inputs": [
//			{
//				"internalType": "uint256[]",
//				"name": "_a",
//				"type": "uint256[]"
//			},
//			{
//				"internalType": "address[]",
//				"name": "_addr",
//				"type": "address[]"
//			}
//		],
//		"name": "f1",
//		"outputs": [],
//		"stateMutability": "nonpayable",
//		"type": "function"
//	},
//	{
//		"inputs": [],
//		"name": "f2",
//		"outputs": [
//			{
//				"internalType": "uint256",
//				"name": "",
//				"type": "uint256"
//			}
//		],
//		"stateMutability": "view",
//		"type": "function"
//	},
// ......
// ]
type AbiData struct {
	Inputs []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"inputs"`
	Name    string `json:"name"`
	Type    string `json:"type"` // constructor, function, etc.
	Outputs []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"outputs"`
}

type AbiJSONData struct {
	ABI []AbiData `json:"abi"`
}

func extractFuncDefinition(abi string, funcName string) (string, error) {
	// log.Printf("abi = %s\nfuncName = %s", abi, funcName)
	abi = strings.TrimSpace(abi)
	if len(abi) == 0 {
		return "", fmt.Errorf("abi is empty")
	}

	var parsedABI []AbiData

	if abi[0:1] == "[" {
		if err := json.Unmarshal([]byte(abi), &parsedABI); err != nil {
			return "", fmt.Errorf("unmarshal fail: %w", err)
		}
	} else if abi[0:1] == "{" {
		var abiJSONData AbiJSONData
		if err := json.Unmarshal([]byte(abi), &abiJSONData); err != nil {
			return "", fmt.Errorf("unmarshal fail: %w", err)
		}
		parsedABI = abiJSONData.ABI
	} else {
		return "", fmt.Errorf("abi invalid")
	}

	var ret = funcName + "("

	if len(parsedABI) == 0 {
		return "", fmt.Errorf("parsedABI is empty")
	}

	var foundFunc = false
	for _, item := range parsedABI {
		if funcName == "constructor" { // constructor
			if item.Type == "constructor" {
				foundFunc = true
			}
		} else { // normal function
			if item.Type == "function" && item.Name == funcName {
				foundFunc = true
			}
		}
		if foundFunc == true {
			for index, input := range item.Inputs {
				ret += input.Type

				if index < len(item.Inputs)-1 { // not the last input
					ret += ", "
				}
			}

			ret += ")"

			if len(item.Outputs) > 0 {
				ret += " returns ("
				for index, output := range item.Outputs {
					ret += output.Type

					if index < len(item.Outputs)-1 { // not the last input
						ret += ", "
					}
				}

				ret += ")"
			}

			break
		}
	}

	if !foundFunc {
		return "", fmt.Errorf("function %v not found in ABI", funcName)
	}

	// Example of ret: `f1(uint256[], address[]) returns (uint256)`
	return ret, nil
}
