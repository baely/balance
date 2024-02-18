package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/baely/balance/internal/model"
)

type WebhookEvent struct {
	TransactionDescription string `json:"transaction_description"`
	TransactionAmount      string `json:"transaction_amount"`
	AccountBalance         string `json:"account_balance"`
}

func getCurrencySymbol(currencyCode string) string {
	symbols := map[string]string{
		"JPY": "¥",
		"SGD": "S$",
		"USD": "US$",
		"NTD": "NT$",
		"SKW": "₩",
	}

	symbol, ok := symbols[currencyCode]

	if !ok {
		return "$"
	}

	return symbol
}

func SendWebhookEvent(uri string, account model.AccountResource, transaction model.TransactionResource) error {
	_, err := url.Parse(uri)
	if err != nil {
		return err
	}

	foreign := false
	amt := transaction.Attributes.Amount.Value

	if transaction.Attributes.ForeignAmount != nil {
		foreign = true
		amt = transaction.Attributes.ForeignAmount.Value
	}

	// Validate amount is negative
	if len(amt) == 0 || amt[0] != '-' {
		fmt.Println("non neg amount.", transaction.Attributes.Description, amt)
		return nil
	}

	amt = amt[1:]

	var amtText string
	if foreign {
		currencySymbol := getCurrencySymbol(transaction.Attributes.ForeignAmount.CurrencyCode)
		amtText = fmt.Sprintf("%s%s", currencySymbol, amt)
	} else {
		amtText = fmt.Sprintf("$%s", amt)
	}

	event := WebhookEvent{
		TransactionDescription: transaction.Attributes.Description,
		TransactionAmount:      amtText,
		AccountBalance:         account.Attributes.Balance.Value,
	}

	eventMsg, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := bytes.NewReader(eventMsg)

	resp, err := http.Post(uri, "application/json", msg)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid request")
	}

	return nil
}
