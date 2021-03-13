package cmd

import (
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

const sortNo = "no"
const sortAsc = "asc"
const sortDesc = "desc"

const unitWei = "wei"
const unitGwei = "gwei"
const unitEther = "ether"

func init() {
	balanceCmd.Flags().StringVarP(&balanceSortOpt, "sort", "s", "no", "no | asc | desc, sort result")
	balanceCmd.Flags().StringVarP(&balanceUnit, "unit", "u", "ether", "wei | gwei | ether, unit of balance")
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

var balanceCmd = &cobra.Command{
	Use:   "balance eth-address1 eth-address2 ...",
	Short: "Check eth balance for address",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires an address at least")
		}
		for _, arg := range args {
			if !isValidEthAddress(arg) {
				return fmt.Errorf("%v is not a valid eth address", arg)
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !validationBalanceCmdOpts() {
			_ = cmd.Help()
			os.Exit(1)
		}

		addresses := args

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
