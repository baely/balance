package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"golang.org/x/text/currency"

	"github.com/baely/balance/internal/model"
)

type WebhookEvent struct {
	TransactionDescription string `json:"transaction_description"`
	TransactionAmount      string `json:"transaction_amount"`
	AccountBalance         string `json:"account_balance"`
}

func formatCurrency(value string, iso string) string {
	code, err := currency.ParseISO(value)
	if err != nil {
		log.Printf("failed to parse currency code: %v", err)
		return value + " " + iso
	}

	s := code.String() + value + " (" + iso + ")"
	return s
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

	amtText := fmt.Sprintf("$%s", amt)
	if foreign {
		amtText = formatCurrency(amt, transaction.Attributes.ForeignAmount.CurrencyCode)
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
