package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var downloadSrcCmdSaveDir string

func init() {
	downloadSrcCmd.Flags().StringVarP(&downloadSrcCmdSaveDir, "directory", "d", "./", "the directory of contract")
}

var downloadSrcCmd = &cobra.Command{
	Use:   "download-src [flags] <contract-address>",
	Short: "Download source code of contract from block explorer platform, eg. etherscan.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := downloadSrc(args[0]); err != nil {
			log.Fatalf("downloadSrc failed %v", err)
		}
	},
}

func downloadSrc(contractAddress string) error {
	var requestUrl = fmt.Sprintf(nodeApiUrlMap[globalOptChain], contractAddress)

	resp, err := http.Get(requestUrl)
	if err != nil {
		// handle error
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// handle error
		return err
	}

	type respMsg struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  []struct {
			SourceCode   string `json:"SourceCode"`
			ContractName string `json:"ContractName"`
		} `json:"result"`
	}

	var data respMsg
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	// log.Printf("%+v", data)

	var sourceCode = data.Result[0].SourceCode
	if len(sourceCode) == 0 {
		log.Fatalf("Contract %v is not found or not verified", contractAddress)
	}

	// make sure downloadSrcCmdSaveDir exist
	err = os.MkdirAll(downloadSrcCmdSaveDir, os.ModePerm)
	checkErr(err)

	if strings.HasPrefix(sourceCode, "{{") {
		// Solidity Standard Json-Input
		// An example: https://api.etherscan.io/api?module=contract&action=getsourcecode&address=0xa7EE3E16367D6Bd9dC59cc32Cdcc2eE51b663a4F

		// Remove one leading "{" and one trailing "}"
		sourceCode = strings.TrimPrefix(sourceCode, "{")
		sourceCode = strings.TrimSuffix(sourceCode, "}")

		type SourcesInfo struct {
			Content string `json:"content"`
		}
		type sourceJson struct {
			Language string                  `json:"language"`
			Sources  map[string]*SourcesInfo `json:"sources"`
		}

		var data sourceJson
		if err := json.Unmarshal([]byte(sourceCode), &data); err != nil {
			return err
		}

		// log.Printf("%+v", data)

		for contractName, contractContent := range data.Sources {
			var contractFileName = filepath.Join(downloadSrcCmdSaveDir, contractName)
			// Make sure parent directory of contractFileName exist
			var dir = filepath.Dir(contractFileName)
			err = os.MkdirAll(dir, os.ModePerm)
			checkErr(err)

			saveContract(contractFileName, contractContent.Content)
			checkErr(err)
		}

	} else if strings.HasPrefix(sourceCode, "{") {
		// Solidity Multiple files format
		// An example: https://api.etherscan.io/api?module=contract&action=getsourcecode&address=0x35036A4b7b012331f23F2945C08A5274CED38AC2

		type sourceJson struct {
			X map[string]struct {
				Content string `json:"content"`
			} `json:"-"` // Rest of the fields should go here.
			// See https://stackoverflow.com/questions/33436730/unmarshal-json-with-some-known-and-some-unknown-field-names
		}
		var data sourceJson
		if err := json.Unmarshal([]byte(sourceCode), &data.X); err != nil {
			return err
		}

		for contractName, contractContent := range data.X {
			saveContract(filepath.Join(downloadSrcCmdSaveDir, contractName), contractContent.Content)
			checkErr(err)
		}
	} else {
		// Solidity Single file
		// An example: https://api.etherscan.io/api?module=contract&action=getsourcecode&address=0xdac17f958d2ee523a2206206994597c13d831ec7
		saveContract(filepath.Join(downloadSrcCmdSaveDir, data.Result[0].ContractName+".sol"), sourceCode)
	}

	return nil
}

func saveContract(fileName string, data string) {
	log.Printf("saving %v", fileName)
	err := os.WriteFile(fileName, []byte(data), 0644)
	checkErr(err)
}
