package balance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
	"github.com/baely/balance/internal/database"
	"github.com/baely/balance/internal/integrations"
	"github.com/baely/balance/internal/model"
	"github.com/baely/balance/internal/service"
	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/google/uuid"
)

func init() {
	functions.HTTP("balance", RetrieveAccountBalance)
	functions.HTTP("trigger-balance-update", TriggerBalanceUpdate)
	functions.CloudEvent("process-transaction", ProcessTransaction)
	functions.HTTP("register", RegisterWebhook)
}

func RetrieveAccountBalance(w http.ResponseWriter, r *http.Request) {
	// Retrieve current account balance from firestore
	dbClient, err := database.GetClient(os.Getenv("GCP_PROJECT"))
	defer dbClient.Close()
	if err != nil {
		fmt.Println(err)
		http.Error(w, "", http.StatusInternalServerError)
	}

	accountBalance, err := dbClient.GetAccountBalance()
	if err != nil {
		fmt.Println(err)
		http.Error(w, "", http.StatusInternalServerError)
	}

	// Write account balance to response
	io.WriteString(w, accountBalance)
}

func TriggerBalanceUpdate(w http.ResponseWriter, r *http.Request) {
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

	msg := &pubsub.Message{
		ID:   guid.String(),
		Data: body,
	}

	// Push event to pubsub topic
	ctx := context.Background()
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
	Data []byte `json:"data"`
}

func ProcessTransaction(ctx context.Context, e event.Event) error {
	// Process incoming event
	var upEvent model.WebhookEventCallback
	var msg MessagePublishedData

	err := e.DataAs(&msg)
	if err != nil {
		return err
	}

	b := bytes.NewBuffer(msg.Message.Data)
	json.NewDecoder(b).Decode(&upEvent)

	json.Unmarshal(msg.Message.Data, &msg)

	if upEvent.Data.Attributes.EventType != model.WebhookEventTypeEnum("TRANSACTION_CREATED") {
		fmt.Println("Stop processing. Transaction ID:", upEvent.Data.Relationships.Transaction.Data.Id)
		return nil
	}

	upClient := integrations.NewUpClient(os.Getenv("UP_TOKEN"))

	// Retrieve transaction details
	eventTransaction := upEvent.Data.Relationships.Transaction

	if eventTransaction == nil {
		fmt.Println("no transaction details")
		return nil
	}
	transaction, err := upClient.GetTransaction(eventTransaction.Data.Id)
	if err != nil {
		return err
	}

	// Retrieve account details
	accountId := transaction.Relationships.Account.Data.Id
	account, err := upClient.GetAccount(accountId)
	if err != nil {
		return err
	}

	if account.Attributes.AccountType != model.AccountTypeEnum("TRANSACTIONAL") {
		return nil
	}

	accountBalance := account.Attributes.Balance.Value

	// Update datastore
	dbClient, _ := database.GetClient(os.Getenv("GCP_PROJECT"))
	defer dbClient.Close()

	dbClient.UpdateAccountBalance(accountBalance)

	webhookUris, _ := dbClient.GetWebhookUris()
	rawWebhookUris, _ := dbClient.GetRawWebhookUris()

	for _, uri := range webhookUris {
		go func(uri string) {
			if err := service.SendWebhookEvent(uri, account, transaction); err != nil {
				fmt.Println("error sending webhook:", err)
			}
		}(uri)
	}

	for _, uri := range rawWebhookUris {
		go func(uri string) {
			if err := service.SendRawWebhookEvent(uri, account, transaction); err != nil {
				fmt.Println("error sending webhook:", err)
			}
		}(uri)
	}

	return nil
}

func RegisterWebhook(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("data error:", err)
		http.Error(w, "", http.StatusInternalServerError)
	}

	// Get URI from request
	uri := string(data)

	dbClient, _ := database.GetClient(os.Getenv("GCP_PROJECT"))
	defer dbClient.Close()

	// Add new URI to firestore
	err = dbClient.AddWebhook(uri)
	if err != nil {
		fmt.Println("database write error:", err)
		http.Error(w, "", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusCreated)
}
