package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/baely/balance/internal/database"
	"github.com/baely/balance/internal/integrations"
	"github.com/baely/balance/internal/service"
	"github.com/baely/balance/pkg/model"
)

const (
	tracerName = "balance"
)

type Server struct {
	http.Server
}

func newServer() *Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := chi.NewRouter()

	r.Handle("/account-balance", otelhttp.NewHandler(http.HandlerFunc(RetrieveAccountBalance), "RetrieveAccountBalance"))
	r.Handle("/webhook", otelhttp.NewHandler(http.HandlerFunc(TriggerBalanceUpdate), "TriggerBalanceUpdate"))
	r.Handle("/register", otelhttp.NewHandler(http.HandlerFunc(RegisterWebhook), "RegisterWebhook"))
	r.Handle("/process", otelhttp.NewHandler(http.HandlerFunc(ProcessTransaction), "ProcessTransaction"))

	return &Server{
		http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: r,
		},
	}
}

func (s *Server) Start() error {
	fmt.Println("Server listening on port", s.Addr)
	return s.ListenAndServe()
}

func RetrieveAccountBalance(w http.ResponseWriter, r *http.Request) {
	// Retrieve current account balance from firestore
	dbClient, err := database.GetClient(os.Getenv("GCP_PROJECT"))
	if err != nil {
		fmt.Println(err)
		http.Error(w, "", http.StatusInternalServerError)
	}
	defer dbClient.Close()

	accountBalance, err := dbClient.GetAccountBalance()
	if err != nil {
		fmt.Println(err)
		http.Error(w, "", http.StatusInternalServerError)
	}

	// Write account balance to response
	io.WriteString(w, accountBalance)
}

func TriggerBalanceUpdate(w http.ResponseWriter, r *http.Request) {
	span := trace.SpanFromContext(r.Context())
	defer span.End()

	client := integrations.GetClient()
	defer client.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("read error:", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	if !integrations.ValidateWebhookEvent(
		body,
		r.Header.Get("X-Up-Authenticity-Signature"),
	) {
		http.Error(w, "", http.StatusUnauthorized)
		fmt.Println("error: failed to validate incoming event")
		return
	}

	topic := client.Topic("webhook-events")

	guid, _ := uuid.NewRandom()
	span.SetAttributes(attribute.String("message-id", guid.String()))

	msg := &pubsub.Message{
		ID:   guid.String(),
		Data: body,
	}

	// Push event to pubsub topic
	ctx := r.Context()
	res := topic.Publish(ctx, msg)
	_, err = res.Get(ctx)
	if err != nil {
		fmt.Println("publish error:", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}

// MessagePublishedData contains the full Pub/Sub message
// See the documentation for more details:
// https://cloud.google.com/eventarc/docs/cloudevents#pubsub
type MessagePublishedData struct {
	Message PubSubMessage
}

// PubSubMessage is the payload of a Pub/Sub event.
// See the documentation for more details:
// https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
type PubSubMessage struct {
	ID   string `json:"id"`
	Data []byte `json:"data"`
}

func unmarshall[T any](r io.Reader, v T) (string, error) {
	var e MessagePublishedData

	if err := json.NewDecoder(r).Decode(&e); err != nil {
		return "", err
	}
	return e.Message.ID, json.Unmarshal(e.Message.Data, &v)
}

func ProcessTransaction(w http.ResponseWriter, r *http.Request) {
	span := trace.SpanFromContext(r.Context())
	defer span.End()

	ctx := r.Context()

	var upEvent model.WebhookEventCallback
	messageId, err := unmarshall(r.Body, &upEvent)
	if err != nil {
		fmt.Println("unmarshall error:", err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	span.SetAttributes(attribute.String("message-id", messageId))

	upClient := integrations.NewUpClient(os.Getenv("UP_TOKEN"))

	// Retrieve transaction details
	eventTransaction := upEvent.Data.Relationships.Transaction

	if eventTransaction == nil {
		fmt.Println("no transaction details")
		return
	}
	transaction, err := upClient.GetTransaction(ctx, eventTransaction.Data.Id)
	if err != nil {
		fmt.Println("error retrieving transaction:", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// Retrieve account details
	accountId := transaction.Relationships.Account.Data.Id
	account, err := upClient.GetAccount(ctx, accountId)
	if err != nil {
		fmt.Println("error retrieving account:", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	accountBalance := account.Attributes.Balance.Value

	// Update datastore
	dbClient, err := database.GetClient(os.Getenv("GCP_PROJECT"))
	if err != nil {
		fmt.Println("database error:", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer dbClient.Close()

	dbClient.UpdateAccountBalance(accountBalance)

	webhookUris, _ := dbClient.GetWebhookUris()
	rawWebhookUris, _ := dbClient.GetRawWebhookUris()

	wg := &sync.WaitGroup{}
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
			if err := service.SendWebhookEvent(uri, account, transaction); err != nil {
				fmt.Println("error sending webhook:", err)
			}
			wg.Done()
		}(uri)
	}

	fmt.Println("sending raw webhook events. count:", len(rawWebhookUris))
	for _, uri := range rawWebhookUris {
		wg.Add(1)
		go func(uri string) {
			fmt.Println("sending raw webhook to:", uri)
			if err := service.SendRawWebhookEvent(uri, account, transaction); err != nil {
				fmt.Println("error sending raw webhook:", err)
			}
			wg.Done()
		}(uri)
	}

	wg.Wait()

	// push the message to pubsub
	type TransactionEvent struct {
		Account     model.AccountResource
		Transaction model.TransactionResource
	}

	data, _ := json.Marshal(TransactionEvent{
		Account:     account,
		Transaction: transaction,
	})

	client := integrations.GetClient()
	topic := client.Topic("transactions")
	res := topic.Publish(ctx, &pubsub.Message{
		ID:   uuid.NewString(),
		Data: data,
	})
	id, err := res.Get(context.Background())
	if err != nil {
		fmt.Println("error publishing message:", err)
	} else {
		fmt.Println("new published message:", id)
	}

	return
}

func RegisterWebhook(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("data error:", err)
		http.Error(w, "", http.StatusInternalServerError)
	}

	// Get URI from request
	uri := string(data)

	dbClient, err := database.GetClient(os.Getenv("GCP_PROJECT"))
	if err != nil {
		fmt.Println("database error:", err)
		http.Error(w, "", http.StatusInternalServerError)
	}
	defer dbClient.Close()

	// Add new URI to firestore
	err = dbClient.AddWebhook(uri)
	if err != nil {
		fmt.Println("database write error:", err)
		http.Error(w, "", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusCreated)
}
