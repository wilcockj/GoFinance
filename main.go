package main

import (
	"fmt"
	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v2"
	"os"
)

type Investment struct {
	name                 string
	balance              decimal.Decimal
	monthly_contribution decimal.Decimal
	expected_return      decimal.Decimal
	pretax               bool
}

type Assets struct {
	investments           []Investment
	monthly_pretax_income int64
	monthly_income        Income
	monthly_expenses      int64
	months_invested       int
}

type Income struct {
	pretax   int64
	aftertax int64
}

// add a pre tax income and post tax income field
// add a mark for investment if it is from pretax income

func get_expenses(config map[string]interface{}) float64 {
	expenses, ok := config["expenses"].(map[interface{}]interface{})
	if !ok {
		fmt.Println("No expenses found in config")
		return 0.0
	}

	var tot_expenses float64 = 0.0
	for _, amount := range expenses {
		switch v := amount.(type) {
		case float64:
			tot_expenses += v
		case int:
			tot_expenses += float64(v)
		case int64:
			tot_expenses += float64(v)
		default:
			fmt.Printf("Unexpected type %T for value %v\n", v, v)
		}
	}

	return tot_expenses
}

func get_investments_total_balance(config map[string]interface{}) []Investment {
	var investments []Investment
	accounts, ok := config["investments"].(map[interface{}]interface{})
	if !ok {
		fmt.Println("No investment accounts found")
		return investments
	}

	for key, acc := range accounts {
		var new_investment Investment
		account, ok := acc.(map[interface{}]interface{})
		if !ok {
			fmt.Println("No account found")
			return investments
		}

		balance, ok := account["balance"].(float64)

		if !ok {
			balanceInt, ok := account["balance"].(int)
			if ok {
				balance = float64(balanceInt)
			} else {
				fmt.Println("Balance not found or is not a member")
			}

		}

		new_investment.name = key.(string)
		if new_investment.name == "401k" {
			new_investment.pretax = true
		} else {
			new_investment.pretax = false
		}
		new_investment.balance = decimal.NewFromFloat(balance)

		expected, ok := account["expected_return"].(float64)

		expected_return := decimal.NewFromFloat(expected / 100)
		new_investment.expected_return = expected_return

		monthly, ok := account["monthly_contribution"].(float64)

		new_investment.monthly_contribution = decimal.NewFromFloat(monthly)

		investments = append(investments, new_investment)

	}

	return investments

}

func print_investments(investments []Investment) {
	tot_assets := 0
	for _, investment := range investments {
		fmt.Printf("%s\n", investment.name)
		fmt.Println(investment.balance)
		fmt.Println(investment.monthly_contribution)
		fmt.Println(investment.expected_return)
		fmt.Println()
		tot_assets += int(investment.balance.InexactFloat64())
	}
	fmt.Printf("Total assets %d", tot_assets)

}

func get_income(config map[string]interface{}) Income {
	var income Income
	expenses, ok := config["income"].(map[interface{}]interface{})
	if !ok {
		fmt.Println("income")
		return income
	}

	incomes, ok := expenses["job"].(map[interface{}]interface{})

	pretax_income := int64(incomes["pretax"].(int))
	aftertax_income := int64(incomes["aftertax"].(int))

	income.pretax = pretax_income
	income.aftertax = aftertax_income

	return income
}

func step_investments(assets *Assets, num_months int) {
	investments := &assets.investments
	for range num_months {
		for i := range *investments {
			(*investments)[i].balance = (*investments)[i].balance.Add((*investments)[i].balance.Mul((*investments)[i].expected_return).Div((decimal.NewFromInt(12))))
			(*investments)[i].balance = (*investments)[i].balance.Add((*investments)[i].monthly_contribution)
			if (*investments)[i].name == "cash" {
				net_income := decimal.NewFromInt(assets.monthly_income.aftertax - assets.monthly_expenses)
				fmt.Printf("added %f to cash\n", net_income.InexactFloat64())
				// need to subtract monthly contributions
				// or add that to expenses
				(*investments)[i].balance = (*investments)[i].balance.Add(net_income)

			}
			if (*investments)[i].name == "401k" {
				(*investments)[i].balance = (*investments)[i].balance.Add(decimal.NewFromFloat(float64(assets.monthly_income.pretax) * float64(.085)))
			}
		}
		assets.months_invested++
		fmt.Printf("months invested = %d\n", assets.months_invested)
		if (assets.months_invested+1)%12 == 0 {
			// yearly income increase of 2%
			fmt.Printf("income changed from %d\n", assets.monthly_income.aftertax)
			assets.monthly_income.aftertax = int64(float64(assets.monthly_income.aftertax) * float64(1.02))
			fmt.Printf("to %d\n", assets.monthly_income.aftertax)

		}
	}
}

func main() {
	// command line application
	// config file with assets,
	// cash,
	// investments with expected returns, dividends if appropriate
	// 401k, expected investment per month + returns per year
	// take home income -> how it will be dispersed
	args := os.Args[1:]
	yamlFile, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Printf("yamlFile.Get err #%v ", err)
	}

	obj := make(map[string]interface{})
	err = yaml.Unmarshal(yamlFile, obj)
	if err != nil {
		fmt.Printf("Unmarshal: %v", err)
	}
	var tot_expenses float64 = get_expenses(obj)

	var investments []Investment = get_investments_total_balance(obj)
	for _, investment := range investments {
		if !investment.pretax {
			tot_expenses += investment.monthly_contribution.InexactFloat64()
		}
	}
	income := get_income(obj)

	my_assets := Assets{investments: investments, monthly_expenses: int64(tot_expenses), monthly_income: income}
	step_investments(&my_assets, 14)

	fmt.Printf("Got total expenses per month of %f\n\n", tot_expenses)

	print_investments(investments)

}
