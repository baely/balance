package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/baely/balance/internal/integrations"
	"github.com/baely/balance/pkg/model"
)

var (
	secret = "up:yeah:zs7n6TaKmVv7g6DC15jHyqdBL5kFrQCuDPWxJ9PaxORWotKs9gNKHQeYBCd7LDPGiZgCWfGT2yzLHdGc6uBnq40tWV67rIBs2LciApL7vAjciDMiEmLFODxs7qUnvF9i"
	loc, _ = time.LoadLocation("Australia/Melbourne")
)

func main() {
	client := integrations.NewUpClient(secret)

	transactions, err := client.GetTransactions()
	if err != nil {
		panic(err)
	}

	for _, transaction := range transactions {
		fmt.Println(transaction.Attributes.Description)
	}

	transactions = filter(transactions)

	b, err := json.Marshal(transactions)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("transactions-all.json", b, 0644)
	if err != nil {
		panic(err)
	}
}

func filter(txns []model.TransactionResource) []model.TransactionResource {
	var out []model.TransactionResource

	for _, txn := range txns {
		if check(txn) { //amountBetween(-700, -400),         // between -$7 and -$4
			//timeBetween(6, 12),                // between 6am and 12pm
			//weekday(),                         // on a weekday
			//notForeign(),                      // not a foreign transaction
			//category("restaurants-and-cafes"), // in the restaurants-and-cafes category)

			out = append(out, txn)
		}
	}

	return out
}

type decider func(model.TransactionResource) bool

func check(transaction model.TransactionResource, deciders ...decider) bool {
	for _, d := range deciders {
		if !d(transaction) {
			return false
		}
	}
	return true
}

func amountBetween(minBaseUnits, maxBaseUnits int) decider {
	return func(transaction model.TransactionResource) bool {
		valueInBaseUnits := transaction.Attributes.Amount.ValueInBaseUnits
		return valueInBaseUnits >= minBaseUnits && valueInBaseUnits <= maxBaseUnits
	}
}

func timeBetween(minHour, maxHour int) decider {
	return func(transaction model.TransactionResource) bool {
		hour := transaction.Attributes.CreatedAt.Hour()
		return hour >= minHour && hour <= maxHour
	}
}

func weekday() decider {
	return func(transaction model.TransactionResource) bool {
		day := transaction.Attributes.CreatedAt.Weekday()
		return day >= 1 && day <= 5
	}
}

func fresh() decider {
	return func(transaction model.TransactionResource) bool {
		now := time.Now().In(loc)
		fmt.Println(now)
		fmt.Println(transaction)
		if now.Year() == transaction.Attributes.CreatedAt.Year() &&
			now.Month() == transaction.Attributes.CreatedAt.Month() &&
			now.Day() == transaction.Attributes.CreatedAt.Day() {
			return true
		}
		return false
	}
}

func notForeign() decider {
	return func(transaction model.TransactionResource) bool {
		return transaction.Attributes.ForeignAmount == nil
	}
}

func category(categoryId string) decider {
	return func(transaction model.TransactionResource) bool {
		if transaction.Relationships.Category.Data == nil {
			return false
		}

		return transaction.Relationships.Category.Data.Id == categoryId
	}
}
