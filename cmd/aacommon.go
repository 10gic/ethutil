package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stackup-wallet/stackup-bundler/pkg/userop"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"
)

var bundlerUrl = "https://api.stackup.sh/v1/node/65ba56ebadb9aa140bac2f88508d05ef233a3b502baac1e6f4b6674f890e3eba"

// estimateUserOperationGas returns preVerificationGas, verificationGasLimit, callGasLimit
func estimateUserOperationGas(uo userop.UserOperation, entryPointAddr common.Address) (*big.Int, *big.Int, *big.Int, error) {
	/*
		curl --request POST \
		     --url https://api.stackup.sh/v1/node/65ba56ebadb9aa140bac2f88508d05ef233a3b502baac1e6f4b6674f890e3eba \
		     --header 'accept: application/json' \
		     --header 'content-type: application/json' \
		     --data '
		{
		  "jsonrpc": "2.0",
		  "id": 1,
		  "method": "eth_estimateUserOperationGas",
		  "params": [
		     {"sender":"0xc4426ac4a89972f2b9fcc84247598b7e2a58040f","nonce":"0x1","initCode":"0x","callData":"0xb61d27f600000000000000000000000046ce18b119d0eb454cdbd37545bbca791bf325b300000000000000000000000000000000000000000000000000038d7ea4c6800000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000000","callGasLimit":"0x0","verificationGasLimit":"0x0","preVerificationGas":"0x0","maxFeePerGas":"0x103149","maxPriorityFeePerGas":"0xb5","paymasterAndData":"0x","signature":"0x"},
		     "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789"
		  ]
		}'
	*/

	// Convert to json
	userOpBytes, err := uo.MarshalJSON()
	if err != nil {
		return nil, nil, nil, err
	}

	var requestBody = fmt.Sprintf(`
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "eth_estimateUserOperationGas",
  "params": [
	%s,
	"%s"
  ]
}`, userOpBytes, entryPointAddr.String())

	log.Printf("eth_estimateUserOperationGas request body: %s", requestBody)

	req, err := http.NewRequest(http.MethodPost, bundlerUrl, strings.NewReader(requestBody))
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	var client = &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("http failed: %w", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ReadAll failed: %w", err)
	}

	// Example of error response:
	// {"error":{"code":-32500,"data":{"OpIndex":0,"Reason":"AA10 sender already constructed"},"message":"AA10 sender already constructed"},"id":1,"jsonrpc":"2.0"}
	// Example of success response:
	// {"id":1,"jsonrpc":"2.0","result":{"preVerificationGas":42832,"verificationGas":19364,"callGasLimit":33100}}

	// Check error response
	if strings.Contains(string(respBody), "error") {
		return nil, nil, nil, fmt.Errorf("eth_estimateUserOperationGas failed: %s", respBody)
	}

	type SuccessRespResult struct {
		PreVerificationGas int64 `json:"preVerificationGas"`
		VerificationGas    int64 `json:"verificationGas"`
		CallGasLimit       int64 `json:"callGasLimit"`
	}
	type SuccessResp struct {
		Result SuccessRespResult `json:"result"`
	}

	var successResp SuccessResp
	err = json.Unmarshal(respBody, &successResp)
	if err != nil {
		return nil, nil, nil, err
	}

	return big.NewInt(successResp.Result.PreVerificationGas),
		big.NewInt(successResp.Result.VerificationGas),
		big.NewInt(successResp.Result.CallGasLimit),
		nil
}

// sendUserOperation send user operation via stackup bundler api
func sendUserOperation(uo userop.UserOperation, entryPointAddr common.Address) error {
	// Convert to json
	userOpBytes, err := uo.MarshalJSON()
	if err != nil {
		return err
	}

	var requestBody = fmt.Sprintf(`
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "eth_sendUserOperation",
  "params": [
	%s,
	"%s"
  ]
}`, userOpBytes, entryPointAddr.String())

	log.Printf("eth_sendUserOperation request body: %s", requestBody)
	if globalOptDryRun {
		return nil
	}

	req, err := http.NewRequest(http.MethodPost, bundlerUrl, strings.NewReader(requestBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	var client = &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http failed: %w", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ReadAll failed: %w", err)
	}

	// Example of error response:
	// {"error":{"code":-32507,"data":null,"message":"Invalid UserOp signature or paymaster signature"},"id":1,"jsonrpc":"2.0"}%
	// Example of success response:
	// {"id":1,"jsonrpc":"2.0","result":"0x492f319cff749092586e90d3bf0437bc4942fdeb9852c8441ae733c6136b802d"}

	// Check error response
	if strings.Contains(string(respBody), "error") {
		return fmt.Errorf("eth_sendUserOperation failed: %s", respBody)
	}

	return nil
}
