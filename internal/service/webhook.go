package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/baely/balance/internal/model"
)

//  outbound_msg = {
//      "transaction_description": request_json["transaction_description"],
//      "transaction_amount": "Example text",
//      "account_balance": "Example text"
//      }

type WebhookEvent struct {
	TransactionDescription string `json:"transaction_description"`
	TransactionAmount      string `json:"transaction_amount"`
	AccountBalance         string `json:"account_balance"`
}

func SendWebhookEvent(uri string, account model.AccountResource, transaction model.TransactionResource) error {
	_, err := url.Parse(uri)
	if err != nil {
		return err
	}

	desc := transaction.Attributes.Description

	// Validate amount is negative
	if len(desc) > 0 && desc[1] != '-' {
		return nil
	}

	desc = desc[1:]

	event := WebhookEvent{
		TransactionDescription: desc,
		TransactionAmount:      transaction.Attributes.Amount.Value,
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
