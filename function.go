package balance

import (
	"bytes"
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudevents/sdk-go/v2/event"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/baely/balance/pkg/database"
	"github.com/baely/balance/pkg/integrations"
	"github.com/baely/balance/pkg/model"
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
	dbClient, err := database.GetClient("baileybutler-syd")
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

	if !integrations.ValidateWebhookEvent(
		r.Body,
		r.Header.Get("X-Up-Authenticity-Signature"),
	) {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	topic := client.Topic("webhook-events")

	guid, _ := uuid.NewRandom()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("read error:", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	msg := &pubsub.Message{
		ID:   guid.String(),
		Data: b,
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
	dbClient, _ := database.GetClient("baileybutler-syd")
	defer dbClient.Close()

	dbClient.UpdateAccountBalance(accountBalance)

	webhookUris, _ := dbClient.GetWebhookUris()

	for _, uri := range webhookUris {
		uriReader := strings.NewReader(uri)
		http.Post(uri, "application/json", uriReader)
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

	dbClient, _ := database.GetClient("baileybutler-syd")
	defer dbClient.Close()

	// Add new URI to firestore
	err = dbClient.AddWebhook(uri)
	if err != nil {
		fmt.Println("database write error:", err)
		http.Error(w, "", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusCreated)
}
