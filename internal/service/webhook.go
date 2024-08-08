package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/baely/balance/internal/database"
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

func SendWebhookEvent(ctx context.Context, uri string, account model.AccountResource, transaction model.TransactionResource) error {
	ctx, span := otel.Tracer("balance").Start(ctx, "send-webhook-event", trace.WithAttributes(attribute.String("uri", uri)))
	defer span.End()

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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, msg)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid request")
	}

	return nil
}

func SendRawWebhookEvent(ctx context.Context, uri string, account model.AccountResource, transaction model.TransactionResource) error {
	ctx, span := otel.Tracer("balance").Start(ctx, "send-raw-webhook-event", trace.WithAttributes(attribute.String("uri", uri)))
	defer span.End()

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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, msg)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to post")
	}

	return nil
}

func FireAll(ctx context.Context, wg *sync.WaitGroup, dbClient *database.Client, upEvent model.WebhookEventCallback, account model.AccountResource, transaction model.TransactionResource) {
	go fireWebhook(ctx, wg, dbClient, upEvent, account, transaction)
	go fireRawWebhook(ctx, wg, dbClient, account, transaction)
}

func fireWebhook(ctx context.Context, wg *sync.WaitGroup, dbClient *database.Client, upEvent model.WebhookEventCallback, account model.AccountResource, transaction model.TransactionResource) {
	webhookUris, _ := dbClient.GetWebhookUris(ctx)
	fmt.Println("sending webhook events. count:", len(webhookUris))
	for _, uri := range webhookUris {
		if upEvent.Data.Attributes.EventType != model.WebhookEventTypeEnum("TRANSACTION_CREATED") {
			// Skip sending webhook events for non-transaction created events
			break
		}
		if account.Attributes.AccountType != model.AccountTypeEnum("TRANSACTIONAL") {
			// Skip sending webhook events for non-transactional accounts
			break
		}

		wg.Add(1)
		go func(uri string) {
			fmt.Println("sending webhook to:", uri)
			if err := SendWebhookEvent(ctx, uri, account, transaction); err != nil {
				fmt.Println("error sending webhook:", err)
			}
			wg.Done()
		}(uri)
	}
}

func fireRawWebhook(ctx context.Context, wg *sync.WaitGroup, dbClient *database.Client, account model.AccountResource, transaction model.TransactionResource) {
	rawWebhookUris, _ := dbClient.GetRawWebhookUris(ctx)
	fmt.Println("sending raw webhook events. count:", len(rawWebhookUris))
	for _, uri := range rawWebhookUris {
		wg.Add(1)
		go func(uri string) {
			fmt.Println("sending raw webhook to:", uri)
			if err := SendRawWebhookEvent(ctx, uri, account, transaction); err != nil {
				fmt.Println("error sending raw webhook:", err)
			}
			wg.Done()
		}(uri)
	}
}
