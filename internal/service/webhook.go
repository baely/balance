package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/baely/balance/pkg/model"
)

func formatCurrency(value string, iso string) string {
	iso = strings.ToUpper(iso)
	code, ok := map[string]string{
		"AUD": "$",
		"JPY": "¥",
		"SGD": "S$",
		"KRW": "₩",
		"TWD": "NT$",
	}[iso]
	if !ok {
		return value + " " + iso
	}

	s := code + value
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

	event := model.WebhookEvent{
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

func SendRawWebhookEvent(uri string, account model.AccountResource, transaction model.TransactionResource) error {
	_, err := url.Parse(uri)
	if err != nil {
		return err
	}

	event := model.RawWebhookEvent{
		Account:     account,
		Transaction: transaction,
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
		return fmt.Errorf("failed to post")
	}

	return nil
}
