package cmd

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

const abiTestVectorsPath = "../testdata/abi_test_vectors.json"

type abiTestVectorFile struct {
	Version int             `json:"version"`
	Vectors []abiTestVector `json:"vectors"`
}

type abiTestVector struct {
	Signature      string         `json:"signature"`
	Calldata       string         `json:"calldata"`
	Params         map[string]any `json:"params,omitempty"`
	SkipDecodeTest string         `json:"skipDecodeTest,omitempty"`
	SkipEncodeTest string         `json:"skipEncodeTest,omitempty"`
}

func loadABITestVectors(t *testing.T) abiTestVectorFile {
	t.Helper()
	data, err := os.ReadFile(abiTestVectorsPath)
	if err != nil {
		t.Fatalf("read vector file failed: %v", err)
	}

	var vectorFile abiTestVectorFile
	if err := json.Unmarshal(data, &vectorFile); err != nil {
		t.Fatalf("unmarshal vector file failed: %v", err)
	}
	if len(vectorFile.Vectors) == 0 {
		t.Fatalf("vector file is empty")
	}
	return vectorFile
}

func jsonValueEqual(v1, v2 any) bool {
	b1, err := json.Marshal(v1)
	if err != nil {
		return false
	}
	b2, err := json.Marshal(v2)
	if err != nil {
		return false
	}

	var o1, o2 any
	if err = json.Unmarshal(b1, &o1); err != nil {
		return false
	}
	if err = json.Unmarshal(b2, &o2); err != nil {
		return false
	}
	return reflect.DeepEqual(o1, o2)
}
