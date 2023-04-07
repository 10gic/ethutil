package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

var balanceSortOpt string
var balanceUnit string
var balanceInputFile string

const sortNo = "no"
const sortAsc = "asc"
const sortDesc = "desc"

const unitWei = "wei"
const unitGwei = "gwei"
const unitEther = "ether"

func init() {
	balanceCmd.Flags().StringVarP(&balanceSortOpt, "sort", "s", "no", "no | asc | desc, sort result")
	balanceCmd.Flags().StringVarP(&balanceUnit, "unit", "u", "ether", "wei | gwei | ether, unit of balance")
	balanceCmd.Flags().StringVarP(&balanceInputFile, "input-file", "f", "", "read address from this file, file - means read stdin")
}

func validationBalanceCmdOpts() bool {
	// validation
	if !contains([]string{sortNo, sortAsc, sortDesc}, balanceSortOpt) {
		log.Printf("invalid option for --sort: %v", balanceSortOpt)
		return false
	}

	if !contains([]string{unitWei, unitGwei, unitEther}, balanceUnit) {
		log.Printf("invalid option for --unit: %v", balanceUnit)
		return false
	}

	return true
}

var addresses []string

var balanceCmd = &cobra.Command{
	Use:   "balance [eth-address1 eth-address2 ...]",
	Short: "Check eth balance for address",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && len(balanceInputFile) == 0 {
			return fmt.Errorf("requires an address at least or specify -f option")
		}

		if len(balanceInputFile) > 0 {
			var inputReader = cmd.InOrStdin()
			if balanceInputFile != "-" {
				// read from regular file
				file, err := os.Open(balanceInputFile)
				if err != nil {
					return fmt.Errorf("failed open file: %v", err)
				}
				inputReader = file
			}
			scanner := bufio.NewScanner(inputReader)
			for scanner.Scan() {
				line := scanner.Text()
				addresses = append(addresses, line)
			}
			if len(addresses) == 0 {
				return fmt.Errorf("file %v is empty, do nothing", balanceInputFile)
			}
		} else {
			addresses = args
		}

		// Validate each address
		for _, address := range addresses {
			if !isValidEthAddress(address) {
				return fmt.Errorf("%v is not a valid eth address", address)
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !validationBalanceCmdOpts() {
			_ = cmd.Help()
			os.Exit(1)
		}
		log.Printf("Current network is %v", globalOptNode)

		InitGlobalClient(globalOptNodeUrl)

		ctx := context.Background()

		type kv struct {
			addr    string
			balance big.Int
		}

		var results []kv
		var finishOutput = false

		for _, addr := range addresses {
			balance, err := globalClient.EthClient.BalanceAt(ctx, common.HexToAddress(addr), nil)
			checkErr(err)

			results = append(results, kv{addr, *balance})

			// print output immediately if no sort demand
			if balanceSortOpt == sortNo {
				if globalOptTerseOutput {
					fmt.Printf("%v %s\n", addr, wei2Other(bigInt2Decimal(balance), balanceUnit).String())
				} else {
					fmt.Printf("addr %v, balance %s %s\n", addr, wei2Other(bigInt2Decimal(balance), balanceUnit).String(), balanceUnit)
				}
				finishOutput = true
			}
		}

		if balanceSortOpt == sortAsc {
			sort.Slice(results, func(i, j int) bool {
				return results[i].balance.Cmp(&results[j].balance) < 0
			})
		} else if balanceSortOpt == sortDesc {
			sort.Slice(results, func(i, j int) bool {
				return results[i].balance.Cmp(&results[j].balance) >= 0
			})
		}

		if !finishOutput {
			for _, result := range results {
				if globalOptTerseOutput {
					fmt.Printf("%v %s\n", result.addr, wei2Other(bigInt2Decimal(&result.balance), balanceUnit).String())
				} else {
					fmt.Printf("addr %v, balance %s %s\n", result.addr, wei2Other(bigInt2Decimal(&result.balance), balanceUnit).String(), balanceUnit)
				}
			}
			finishOutput = true
		}
	},
}
