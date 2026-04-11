package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"reflect"
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
	Use:   "encode-param <function-signature> [arg1 arg2 ...]",
	Short: "Encode input arguments, it's useful when you call contract's method manually",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		funcSignature := args[0]
		inputArgData := args[1:]

		if callCmdABIFile != "" {
			abiContent, err := os.ReadFile(callCmdABIFile)
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
	}
	// if funcName is empty, only encode arguments

	if len(funcArgTypes) != len(inputArgData) {
		return nil, fmt.Errorf("invalid input, there are %v args in signature, but %v args are provided", len(funcArgTypes), len(inputArgData))
	}

	// Parse each string argument into structured data using type information
	structured := make([]any, len(inputArgData))
	for i, raw := range inputArgData {
		val, err := parseArgValue(funcArgTypes[i], raw)
		if err != nil {
			return nil, fmt.Errorf("parse arg %d failed: %w", i, err)
		}
		structured[i] = val
	}

	data, err := encodeParametersFromValues(funcArgTypes, structured)
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
// Examples:
// fun1  ->  fun1
// fun1(uint256)  -> fun1
// function fun1  -> fun1
// function fun1(uint256)  ->  fun1
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
// Example 2:
// input: "function add(uint256   xx, address xx, bool xx) returns (address)"
// output: "add", ["uint256", "address", "bool"], nil
//
// Example 3 (no function name):
// input: "(uint256, address, bool)"
// output: "", ["uint256", "address", "bool"], nil
//
// Example 4 (no parenthesis):
// input: "test"
// output: "test", [], nil
//
// Example 5 (with tuple):
// input:  "function fn1((uint256, address), bool)"
// output: "fn1", ["(uint256, address)", "bool"], nil
func parseFuncSignature(input string) (string, []string, error) {
	if strings.HasPrefix(input, "function ") {
		input = input[len("function "):] // remove leading string "function "
	}

	if strings.Index(input, "(") < 0 && strings.Index(input, ")") < 0 {
		// no parenthesis found
		return strings.Trim(input, " "), []string{}, nil
	}

	input = strings.TrimLeft(input, " ")

	// remove function returns declaration
	returnsLoc := strings.LastIndex(input, "returns")
	if returnsLoc > 0 {
		input = input[:returnsLoc] // `fn1(bool) returns (address)` -> `fn1(bool)`
	}

	leftParenthesisLoc := strings.Index(input, "(")
	if leftParenthesisLoc < 0 {
		return "", nil, fmt.Errorf("char ( is not found in function signature")
	}
	funcName := input[:leftParenthesisLoc] // remove all characters from char '('
	funcName = strings.TrimSpace(funcName)

	rightParenthesisLoc := strings.LastIndex(input, ")")
	if rightParenthesisLoc < 0 {
		return "", nil, fmt.Errorf("char ) is not found in function signature")
	}
	argsPart := input[leftParenthesisLoc+1 : rightParenthesisLoc]
	if strings.TrimSpace(argsPart) == "" {
		return funcName, nil, nil
	}
	args := splitTopLevel(argsPart)
	for index, arg := range args {
		normalized, err := normalizeSignatureArg(arg)
		if err != nil {
			return "", nil, err
		}
		args[index] = normalized
	}

	return funcName, args, nil
}

func normalizeSignatureArg(arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", fmt.Errorf("signature arg is empty")
	}

	if strings.HasPrefix(arg, "(") {
		tupleText, afterTuple, err := extractLeadingTuple(arg)
		if err != nil {
			return "", err
		}
		if len(tupleText) < 2 {
			return "", fmt.Errorf("invalid tuple type %q", arg)
		}

		inner := tupleText[1 : len(tupleText)-1]
		innerArgs := splitTopLevel(inner)
		var normalizedInner []string
		for _, innerArg := range innerArgs {
			normalizedArg, err := normalizeSignatureArg(innerArg)
			if err != nil {
				return "", err
			}
			normalizedInner = append(normalizedInner, normalizedArg)
		}
		normalizedTuple := "(" + strings.Join(normalizedInner, ",") + ")"

		arrayPart, err := extractLeadingArraySuffix(afterTuple)
		if err != nil {
			return "", err
		}
		return normalizedTuple + arrayPart, nil
	}

	fields := strings.Fields(arg)
	if len(fields) == 0 {
		return "", fmt.Errorf("signature arg %q invalid type missing", arg)
	}

	// first field is type. for example,
	// "uint256 xx", first field is uint256
	// "uint256[] xx", first field is uint256[]
	typ := typeNormalize(fields[0])

	if len(fields) >= 2 && fields[0] == "address" && strings.HasPrefix(fields[1], "payable[") {
		// handle case:
		// f1(address payable[] memory a, uint256 b)
		// f1(address payable[3] memory a, uint256 b)
		typ = fields[0] + strings.Replace(fields[1], "payable", "", 1) // typ = address[] or address[3]
	} else if len(fields) >= 2 && fields[0] == "address" && fields[1] == "payable" {
		// handle "address payable x"
		typ = fields[0]
	}

	return typ, nil
}

func extractLeadingTuple(arg string) (string, string, error) {
	arg = strings.TrimSpace(arg)
	if !strings.HasPrefix(arg, "(") {
		return "", "", fmt.Errorf("tuple arg must start with '('")
	}

	depth := 0
	for index, ch := range arg {
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 {
				return arg[:index+1], arg[index+1:], nil
			}
		}
	}
	return "", "", fmt.Errorf("invalid tuple arg %q: unmatched parenthesis", arg)
}

func extractLeadingArraySuffix(input string) (string, error) {
	input = strings.TrimSpace(input)

	var suffix = ""
	for strings.HasPrefix(input, "[") {
		rightBracket := strings.Index(input, "]")
		if rightBracket < 0 {
			return "", fmt.Errorf("invalid array suffix in %q", input)
		}
		suffix += input[:rightBracket+1]
		input = strings.TrimSpace(input[rightBracket+1:])
	}

	return suffix, nil
}

// parseArgValue parses a CLI string argument into a structured Go value based
// on the ABI type. Scalars are returned as strings (for later conversion by
// buildConcreteData). Arrays and tuples are returned as []any with recursively
// parsed elements.
//
// Examples:
//
//	parseArgValue("uint256", "123")                → "123"
//	parseArgValue("address", "0xabc...")            → "0xabc..."
//	parseArgValue("string", "hello, world")         → "hello, world"
//	parseArgValue("uint256[]", "[1, 2, 3]")         → []any{"1", "2", "3"}
//	parseArgValue("(uint256,bool)", "(123, true)")  → []any{"123", "true"}
//	parseArgValue("(string,uint256)", `("hello, world", 123)`) → []any{"hello, world", "123"}
func parseArgValue(argType string, rawArg string) (any, error) {
	argType = strings.TrimSpace(argType)

	isTuple := strings.HasPrefix(argType, "(") && strings.HasSuffix(argType, ")")
	isTupleArray := tupleArrayRE.MatchString(argType)
	isArray := !isTupleArray && !isTuple && strings.HasSuffix(argType, "]")

	if isTupleArray {
		// e.g. (uint256,bool)[3] or (uint256,bool)[]
		elemType := argType[:strings.LastIndex(argType, "[")]
		return parseArrayValue(elemType, rawArg)
	}

	if isTuple {
		return parseTupleValue(argType, rawArg)
	}

	if isArray {
		// e.g. uint256[], address[3], bytes[2][3]
		bracketIdx := strings.LastIndex(argType, "[")
		elemType := argType[:bracketIdx]
		return parseArrayValue(elemType, rawArg)
	}

	// Scalar type — return as-is string
	return rawArg, nil
}

// parseTupleValue parses a tuple data string like "(123, true)" given
// the tuple type string "(uint256,bool)". Returns []any.
func parseTupleValue(tupleType string, rawData string) (any, error) {
	// Parse element types from the type string
	inner := tupleType[1 : len(tupleType)-1]
	elemTypes := splitTopLevel(inner)

	// Strip outer parens from data
	rawData = strings.TrimSpace(rawData)
	if strings.HasPrefix(rawData, "(") && strings.HasSuffix(rawData, ")") {
		rawData = rawData[1 : len(rawData)-1]
	}

	elemValues := splitTopLevel(rawData)
	if len(elemValues) != len(elemTypes) {
		return nil, fmt.Errorf("tuple type has %d elements but data has %d", len(elemTypes), len(elemValues))
	}

	result := make([]any, len(elemTypes))
	for i, et := range elemTypes {
		// Normalize the element type (strip names, normalize int→int256, etc.)
		normalized, err := normalizeSignatureArg(et)
		if err != nil {
			return nil, fmt.Errorf("normalize tuple element type failed: %w", err)
		}
		val, err := parseArgValue(normalized, elemValues[i])
		if err != nil {
			return nil, fmt.Errorf("parse tuple element %d failed: %w", i, err)
		}
		result[i] = val
	}
	return result, nil
}

// parseArrayValue parses an array data string like "[1, 2, 3]" given
// the element type string. Returns []any.
func parseArrayValue(elemType string, rawData string) (any, error) {
	rawData = strings.TrimSpace(rawData)
	if strings.HasPrefix(rawData, "[") && strings.HasSuffix(rawData, "]") {
		rawData = rawData[1 : len(rawData)-1]
	}

	if strings.TrimSpace(rawData) == "" {
		return []any{}, nil
	}

	elemValues := splitTopLevel(rawData)
	result := make([]any, len(elemValues))
	for i, ev := range elemValues {
		val, err := parseArgValue(elemType, ev)
		if err != nil {
			return nil, fmt.Errorf("parse array element %d failed: %w", i, err)
		}
		result[i] = val
	}
	return result, nil
}

// encodeParametersFromValues encodes structured parameter values into ABI bytes.
// Unlike encodeParameters which takes []string, this function takes []any where:
//   - scalar types: string (e.g. "123", "0xabc...", "true")
//   - arrays: []any of recursively structured elements
//   - tuples: []any of recursively structured elements
func encodeParametersFromValues(inputArgTypes []string, inputArgData []any) ([]byte, error) {
	if len(inputArgTypes) != len(inputArgData) {
		return nil, fmt.Errorf("type count %d != data count %d", len(inputArgTypes), len(inputArgData))
	}

	var abiArgs abi.Arguments
	var packedData []any

	for i, argType := range inputArgTypes {
		typ, val, err := buildTypedValue(argType, inputArgData[i])
		if err != nil {
			return nil, fmt.Errorf("arg %d: %w", i, err)
		}
		abiArgs = append(abiArgs, abi.Argument{Type: typ})
		packedData = append(packedData, val)
	}

	return abiArgs.Pack(packedData...)
}

// buildTypedValue builds an abi.Type and a corresponding Go value from a type
// string and a structured data value (string for scalars, []any for compounds).
func buildTypedValue(argType string, data any) (abi.Type, any, error) {
	argType = strings.TrimSpace(argType)

	isTuple := strings.HasPrefix(argType, "(") && strings.HasSuffix(argType, ")")
	isTupleArray := tupleArrayRE.MatchString(argType)
	isArray := !isTupleArray && !isTuple && strings.HasSuffix(argType, "]")

	if isTupleArray {
		return buildTupleArrayValue(argType, data)
	}
	if isTuple {
		return buildTupleValue(argType, data)
	}
	if isArray {
		return buildArrayValue(argType, data)
	}

	// Scalar
	str, ok := data.(string)
	if !ok {
		return abi.Type{}, nil, fmt.Errorf("expected string for scalar type %q, got %T", argType, data)
	}
	typ, err := abi.NewType(typeNormalize(argType), "", nil)
	if err != nil {
		return abi.Type{}, nil, fmt.Errorf("abi.NewType(%q) failed: %w", argType, err)
	}
	val, err := buildConcreteData(typeNormalize(argType), str)
	if err != nil {
		return abi.Type{}, nil, err
	}
	return typ, val, nil
}

// buildTupleValue builds abi.Type and Go struct value for a tuple type.
func buildTupleValue(tupleType string, data any) (abi.Type, any, error) {
	items, ok := data.([]any)
	if !ok {
		return abi.Type{}, nil, fmt.Errorf("expected []any for tuple type %q, got %T", tupleType, data)
	}

	typ, err := buildTupleType(tupleType)
	if err != nil {
		return abi.Type{}, nil, err
	}

	if len(typ.TupleElems) != len(items) {
		return abi.Type{}, nil, fmt.Errorf("tuple %q has %d fields but got %d values", tupleType, len(typ.TupleElems), len(items))
	}

	v := reflect.New(typ.TupleType).Elem()
	for i, name := range typ.TupleRawNames {
		_, val, err := buildTypedValue(typ.TupleElems[i].String(), items[i])
		if err != nil {
			return abi.Type{}, nil, fmt.Errorf("tuple field %s: %w", name, err)
		}
		field := v.FieldByName(name)
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Ptr && field.Kind() == reflect.Struct {
			rv = rv.Elem()
		}
		field.Set(rv)
	}

	return typ, v.Addr().Interface(), nil
}

// buildArrayValue builds abi.Type and Go slice value for an array type.
func buildArrayValue(arrayType string, data any) (abi.Type, any, error) {
	items, ok := data.([]any)
	if !ok {
		return abi.Type{}, nil, fmt.Errorf("expected []any for array type %q, got %T", arrayType, data)
	}

	bracketIdx := strings.LastIndex(arrayType, "[")
	elemType := arrayType[:bracketIdx]

	typ, err := abi.NewType(typeNormalize(arrayType), "", nil)
	if err != nil {
		return abi.Type{}, nil, fmt.Errorf("abi.NewType(%q) failed: %w", arrayType, err)
	}

	if len(items) == 0 {
		goType := typ.GetType()
		if goType.Kind() == reflect.Array {
			return typ, reflect.New(goType).Elem().Interface(), nil
		}
		return typ, reflect.MakeSlice(goType, 0, 0).Interface(), nil
	}

	// Build each element
	var elemVals []any
	for i, item := range items {
		_, val, err := buildTypedValue(elemType, item)
		if err != nil {
			return abi.Type{}, nil, fmt.Errorf("array element %d: %w", i, err)
		}
		elemVals = append(elemVals, val)
	}

	// Build properly typed slice using reflect
	goType := typ.GetType()
	var sliceVal reflect.Value
	if goType.Kind() == reflect.Array {
		if len(elemVals) != goType.Len() {
			return abi.Type{}, nil, fmt.Errorf("array %q expects %d elements but got %d", arrayType, goType.Len(), len(elemVals))
		}
		sliceVal = reflect.New(goType).Elem()
		for i, val := range elemVals {
			rv := reflect.ValueOf(val)
			if rv.Kind() == reflect.Ptr && sliceVal.Index(i).Kind() == reflect.Struct {
				rv = rv.Elem()
			}
			sliceVal.Index(i).Set(rv)
		}
	} else {
		sliceVal = reflect.MakeSlice(goType, len(elemVals), len(elemVals))
		for i, val := range elemVals {
			rv := reflect.ValueOf(val)
			if rv.Kind() == reflect.Ptr && sliceVal.Index(i).Kind() == reflect.Struct {
				rv = rv.Elem()
			}
			sliceVal.Index(i).Set(rv)
		}
	}

	return typ, sliceVal.Interface(), nil
}

// buildTupleArrayValue builds abi.Type and Go slice value for a tuple array type.
func buildTupleArrayValue(arrayType string, data any) (abi.Type, any, error) {
	items, ok := data.([]any)
	if !ok {
		return abi.Type{}, nil, fmt.Errorf("expected []any for tuple array type %q, got %T", arrayType, data)
	}

	typ, err := BuildTupleArrayType(arrayType)
	if err != nil {
		return abi.Type{}, nil, err
	}

	elemTupleType := arrayType[:strings.LastIndex(arrayType, "[")]

	if len(items) == 0 {
		goType := typ.GetType()
		if goType.Kind() == reflect.Array {
			return typ, reflect.New(goType).Elem().Interface(), nil
		}
		return typ, reflect.MakeSlice(goType, 0, 0).Interface(), nil
	}

	goType := typ.GetType()
	var sliceVal reflect.Value
	if goType.Kind() == reflect.Array {
		if len(items) != goType.Len() {
			return abi.Type{}, nil, fmt.Errorf("tuple array %q expects %d elements but got %d", arrayType, goType.Len(), len(items))
		}
		sliceVal = reflect.New(goType).Elem()
	} else {
		sliceVal = reflect.MakeSlice(goType, len(items), len(items))
	}

	for i, item := range items {
		_, val, err := buildTupleValue(elemTupleType, item)
		if err != nil {
			return abi.Type{}, nil, fmt.Errorf("tuple array element %d: %w", i, err)
		}
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		sliceVal.Index(i).Set(rv)
	}

	return typ, sliceVal.Interface(), nil
}

// encodeParameters Encode parameters
// An example:
// inputArgTypes: ["uint256", "address", "bool"]
// inputArgData: ["123", "0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb", "true"]
// return: 000000000000000000000000000000000000000000000000000000000000007b0000000000000000000000008f36975cdea2e6e64f85719788c8efbbe89dfbbb0000000000000000000000000000000000000000000000000000000000000001
// encodeParameters Encode parameters (backward-compatible shim).
// Parses each string argument using type information, then delegates to
// encodeParametersFromValues.
func encodeParameters(inputArgTypes, inputArgData []string) ([]byte, error) {
	structured := make([]any, len(inputArgData))
	for i, raw := range inputArgData {
		val, err := parseArgValue(inputArgTypes[i], raw)
		if err != nil {
			return nil, fmt.Errorf("parse arg %d fail: %w", i, err)
		}
		structured[i] = val
	}
	return encodeParametersFromValues(inputArgTypes, structured)
}

// buildTupleType build tuple abi.Type
// An example:
// tupleType: "(uint256, bool)"
// return: abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "Field0", Type: "uint256"}, {Name: "Field1", Type: "bool"}})
func buildTupleType(tupleType string) (abi.Type, error) {
	components, err := buildTupleComponents(tupleType)
	if err != nil {
		return abi.Type{}, err
	}
	return abi.NewType("tuple", "", components)
}

// BuildTupleArrayType build tuple array abi.Type
// An example:
// tupleType: "(uint256, bool)[5]"
// return: abi.NewType("tuple[5]", "", []abi.ArgumentMarshaling{{Name: "Field0", Type: "uint256"}, {Name: "Field1", Type: "bool"}})
func BuildTupleArrayType(tupleType string) (abi.Type, error) {
	// If tupleType is (bool,uint256)[5], then
	// tupleTypePart1 is (bool,uint256)
	// tupleTypePart2 is [5]
	tupleTypePart1 := tupleType[0:strings.LastIndex(tupleType, "[")]
	tupleTypePart2 := tupleType[strings.LastIndex(tupleType, "["):]
	re := regexp.MustCompile("[0-9]+")
	intz := re.FindString(tupleTypePart2)

	components, err := buildTupleComponents(tupleTypePart1)
	if err != nil {
		return abi.Type{}, err
	}
	return abi.NewType(fmt.Sprintf("tuple[%s]", intz), "", components)
}

// buildTupleComponents parses a tuple type string like "(uint256,bool)" or
// "(int192,((bytes30,bytes16),address))" and returns the ArgumentMarshaling
// slice, recursively handling nested tuples.
func buildTupleComponents(tupleType string) ([]abi.ArgumentMarshaling, error) {
	// Strip outer parens and split by top-level commas
	inner := strings.TrimSpace(tupleType)
	if strings.HasPrefix(inner, "(") && strings.HasSuffix(inner, ")") {
		inner = inner[1 : len(inner)-1]
	}
	arrayOfType := splitTopLevel(inner)
	var components []abi.ArgumentMarshaling
	for index, typ := range arrayOfType {
		m := abi.ArgumentMarshaling{
			Name: "Field" + strconv.Itoa(index),
		}

		// Check for tuple array: (type,type)[N] or (type,type)[]
		if tupleArrayRE.MatchString(typ) {
			tuplePartEnd := strings.LastIndex(typ, "[")
			tuplePart := typ[:tuplePartEnd]
			arraySuffix := typ[tuplePartEnd:]
			re := regexp.MustCompile("[0-9]+")
			intz := re.FindString(arraySuffix)

			sub, err := buildTupleComponents(tuplePart)
			if err != nil {
				return nil, err
			}
			m.Type = fmt.Sprintf("tuple[%s]", intz)
			m.Components = sub
		} else if strings.HasPrefix(typ, "(") && strings.HasSuffix(typ, ")") {
			// Nested tuple
			sub, err := buildTupleComponents(typ)
			if err != nil {
				return nil, err
			}
			m.Type = "tuple"
			m.Components = sub
		} else {
			m.Type = typ
		}

		components = append(components, m)
	}
	return components, nil
}

func buildConcreteData(inputType string, data string) (any, error) {
	if inputType == "string" {
		return data, nil
	} else if inputType == "int8" {
		i, err := parseIntAuto(data, 64)
		if err != nil {
			return nil, fmt.Errorf("%s cannot covert to type %v", data, inputType)
		}
		return int8(i), nil
	} else if inputType == "int16" {
		i, err := parseIntAuto(data, 64)
		if err != nil {
			return nil, fmt.Errorf("%s cannot covert to type %v", data, inputType)
		}
		return int16(i), nil
	} else if inputType == "int32" {
		i, err := parseIntAuto(data, 64)
		if err != nil {
			return nil, fmt.Errorf("%s cannot covert to type %v", data, inputType)
		}
		return int32(i), nil
	} else if inputType == "int64" {
		i, err := parseIntAuto(data, 64)
		if err != nil {
			return nil, fmt.Errorf("%s cannot covert to type %v", data, inputType)
		}
		return int64(i), nil
	} else if inputType == "uint8" {
		i, err := parseUintAuto(data, 64)
		if err != nil {
			return nil, fmt.Errorf("%s cannot covert to type %v", data, inputType)
		}
		return uint8(i), nil
	} else if inputType == "uint16" {
		i, err := parseUintAuto(data, 64)
		if err != nil {
			return nil, fmt.Errorf("%s cannot covert to type %v", data, inputType)
		}
		return uint16(i), nil
	} else if inputType == "uint32" {
		i, err := parseUintAuto(data, 64)
		if err != nil {
			return nil, fmt.Errorf("%s cannot covert to type %v", data, inputType)
		}
		return uint32(i), nil
	} else if inputType == "uint64" {
		i, err := parseUintAuto(data, 64)
		if err != nil {
			return nil, fmt.Errorf("%s cannot covert to type %v", data, inputType)
		}
		return uint64(i), nil
	} else if strings.Contains(inputType, "int") { // other cases: int24, int40, ..., int256, uint24, uint40, ..., uint256, etc
		argData := data

		if !isValidInt(inputType) {
			return nil, fmt.Errorf("type %v not a valid type", inputType)
		}

		if (inputType == "uint256" || inputType == "uint") && strings.Contains(argData, "e") {
			// example:
			// convert 1e18 to 1000000000000000000
			var err error
			argData, err = scientificNotation2Decimal(argData)
			checkErr(err)
		}

		n := new(big.Int)
		n, ok := n.SetString(argData, 10)
		if !ok {
			return nil, fmt.Errorf("%s cannot covert to type %v", argData, inputType)
		}
		return n, nil
	} else if inputType == "bool" {
		if strings.EqualFold(data, "true") {
			return true, nil
		} else if strings.EqualFold(data, "false") {
			return false, nil
		} else {
			return nil, fmt.Errorf("%s cannot covert to type %v", data, inputType)
		}
	} else if inputType == "address" {
		return common.HexToAddress(data), nil
	} else if inputType == "bytes" {
		var inputHex = data
		if strings.HasPrefix(data, "0x") {
			inputHex = data[2:]
		}
		decoded, err := hex.DecodeString(inputHex)
		if err != nil {
			return nil, fmt.Errorf("%s cannot covert to type %v", data, inputType)
		}
		return decoded, nil
	} else if strings.Contains(inputType, "bytes") { // bytes1, bytes2, ..., bytes32
		var inputHex = data
		if strings.HasPrefix(data, "0x") {
			inputHex = data[2:]
		}
		decoded, err := hex.DecodeString(inputHex)
		if err != nil {
			return nil, fmt.Errorf("%s cannot covert to type %v", data, inputType)
		}
		if inputType == "bytes1" {
			var data [1]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes2" {
			var data [2]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes3" {
			var data [3]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes4" {
			var data [4]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes5" {
			var data [5]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes6" {
			var data [6]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes7" {
			var data [7]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes8" {
			var data [8]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes9" {
			var data [9]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes10" {
			var data [10]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes11" {
			var data [11]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes12" {
			var data [12]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes13" {
			var data [13]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes14" {
			var data [14]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes15" {
			var data [15]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes16" {
			var data [16]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes17" {
			var data [17]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes18" {
			var data [18]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes19" {
			var data [19]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes20" {
			var data [20]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes21" {
			var data [21]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes22" {
			var data [22]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes23" {
			var data [23]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes24" {
			var data [24]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes25" {
			var data [25]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes26" {
			var data [26]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes27" {
			var data [27]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes28" {
			var data [28]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes29" {
			var data [29]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes30" {
			var data [30]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes31" {
			var data [31]byte
			copy(data[:], decoded)
			return data, nil
		} else if inputType == "bytes32" {
			var data [32]byte
			copy(data[:], decoded)
			return data, nil
		} else {
			return nil, fmt.Errorf("type %v not implemented currently", inputType)
		}
	} else {
		return nil, fmt.Errorf("type %v not implemented currently", inputType)
	}
}

// parseIntAuto parses an integer string, auto-detecting hex (0x prefix) or decimal.
func parseIntAuto(s string, bitSize int) (int64, error) {
	if has0xPrefix(s) {
		return strconv.ParseInt(remove0xPrefix(s), 16, bitSize)
	}
	return strconv.ParseInt(s, 10, bitSize)
}

// parseUintAuto parses an unsigned integer string, auto-detecting hex (0x prefix) or decimal.
func parseUintAuto(s string, bitSize int) (uint64, error) {
	if has0xPrefix(s) {
		return strconv.ParseUint(remove0xPrefix(s), 16, bitSize)
	}
	return strconv.ParseUint(s, 10, bitSize)
}

// splitTopLevel splits a string by top-level commas, respecting nested
// parentheses, brackets, and double-quoted strings. It does NOT strip
// outer wrappers.
//
// Double-quoted strings are treated as atomic tokens: commas, parentheses,
// and brackets inside quotes do not affect splitting. The quotes themselves
// are stripped from the output. Use \" to include a literal quote.
//
//	"address, (uint256, bool)"            ----> ["address", "(uint256, bool)"]
//	`"hello, world", 123`                ----> ["hello, world", "123"]
//	`("hello, world", true)`             ----> [`("hello, world", true)`]  (inside parens, not split)
func splitTopLevel(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	var rv []string
	var curArg strings.Builder
	var depth = 0     // combined depth for () and []
	var inQuote bool  // inside double-quoted string
	var hadQuote bool // current arg contained quoted content (don't trim)
	var prevCh rune

	appendArg := func() {
		s := curArg.String()
		if !hadQuote {
			s = strings.TrimSpace(s)
		}
		rv = append(rv, s)
		curArg.Reset()
		hadQuote = false
	}

	for _, ch := range input {
		if inQuote {
			if ch == '"' && prevCh != '\\' {
				inQuote = false
				// don't append the closing quote
			} else if ch == '\\' && prevCh != '\\' {
				// escape char — don't append, wait for next char
			} else {
				curArg.WriteRune(ch)
			}
			prevCh = ch
			continue
		}

		switch ch {
		case '"':
			if depth == 0 {
				// Only process quotes at top level — nested quotes are preserved
				// literally for recursive parsing in inner layers.
				inQuote = true
				hadQuote = true
				// Discard leading whitespace before the quote
				if strings.TrimSpace(curArg.String()) == "" {
					curArg.Reset()
				}
			} else {
				curArg.WriteRune(ch)
			}
		case '(', '[':
			depth++
			curArg.WriteRune(ch)
		case ')', ']':
			depth--
			curArg.WriteRune(ch)
		case ',':
			if depth > 0 {
				curArg.WriteRune(',')
			} else {
				appendArg()
			}
		default:
			curArg.WriteRune(ch)
		}
		prevCh = ch
	}
	appendArg()

	return rv
}

// uint -> uint256
// int -> int256
// uint[] -> uint256[]
// int[] -> int256[]
func typeNormalize(input string) string {
	re := regexp.MustCompile(`\b(u?int)\b`)
	return re.ReplaceAllString(input, "${1}256")
}

// ABI example:
// [
//
//	 {
//	     "inputs": [],
//	     "stateMutability": "nonpayable",
//	     "type": "constructor"
//	 },
//		{
//			"inputs": [
//				{
//					"internalType": "uint256[]",
//					"name": "_a",
//					"type": "uint256[]"
//				},
//				{
//					"internalType": "address[]",
//					"name": "_addr",
//					"type": "address[]"
//				}
//			],
//			"name": "f1",
//			"outputs": [],
//			"stateMutability": "nonpayable",
//			"type": "function"
//		},
//		{
//			"inputs": [],
//			"name": "f2",
//			"outputs": [
//				{
//					"internalType": "uint256",
//					"name": "",
//					"type": "uint256"
//				}
//			],
//			"stateMutability": "view",
//			"type": "function"
//		},
//
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

type ErrFuncNotFound struct {
	FuncName string
}

func (e ErrFuncNotFound) Error() string { return fmt.Sprintf("function %s not found", e.FuncName) }

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
		return "", &ErrFuncNotFound{
			FuncName: funcName,
		}
	}

	// Example of ret: `f1(uint256[], address[]) returns (uint256)`
	return ret, nil
}

// isValidInt return true if intType is valid solidity int type
func isValidInt(intType string) bool {
	switch intType {
	case
		"int",
		"int8",
		"int16",
		"int24",
		"int32",
		"int40",
		"int48",
		"int56",
		"int64",
		"int72",
		"int80",
		"int88",
		"int96",
		"int104",
		"int112",
		"int120",
		"int128",
		"int136",
		"int144",
		"int152",
		"int160",
		"int168",
		"int176",
		"int184",
		"int192",
		"int200",
		"int208",
		"int216",
		"int224",
		"int232",
		"int240",
		"int248",
		"int256",
		"uint",
		"uint8",
		"uint16",
		"uint24",
		"uint32",
		"uint40",
		"uint48",
		"uint56",
		"uint64",
		"uint72",
		"uint80",
		"uint88",
		"uint96",
		"uint104",
		"uint112",
		"uint120",
		"uint128",
		"uint136",
		"uint144",
		"uint152",
		"uint160",
		"uint168",
		"uint176",
		"uint184",
		"uint192",
		"uint200",
		"uint208",
		"uint216",
		"uint224",
		"uint232",
		"uint240",
		"uint248",
		"uint256":
		return true
	}
	return false
}

func scientificNotation2Decimal(input string) (string, error) {
	r := regexp.MustCompile(`^([0-9]*)([.]?)([0-9]+)e([0-9]+)$`)
	matches := r.FindStringSubmatch(input)

	part1 := matches[1] // group 1
	part2 := matches[2] // group 2
	part3 := matches[3] // group 3
	part4 := matches[4] // group 4

	part4Int, err := strconv.ParseInt(part4, 10, 64)
	checkErr(err)

	var result = ""
	if part2 == "." {
		// has dot, for example 12.1e3
		if part1 == "0" {
			// for example 0.3e5
			result = part3 + strings.Repeat("0", int(part4Int)-1)
		} else {
			result = part1 + part3 + strings.Repeat("0", int(part4Int)-1)
		}
	} else {
		// no dot
		result = part3 + strings.Repeat("0", int(part4Int))
	}

	log.Printf("convert %v to %v", input, result)
	return result, nil
}
