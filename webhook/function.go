package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/baely/balance/common/database"
	"github.com/baely/balance/common/integrations"
	"github.com/baely/balance/common/model"
)

func init() {
	functions.HTTP("trigger-balance-update", triggerBalanceUpdate)
}

func triggerBalanceUpdate(w http.ResponseWriter, r *http.Request) {
	// Process incoming event
	var event model.WebhookEventResource
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "", http.StatusBadRequest)
	}

	upClient := integrations.NewUpClient("123456")

	// Retrieve transaction details
	eventTransaction := event.Relationships.Transaction
	if eventTransaction == nil {
		fmt.Println("no transaction details")
		return
	}
	transaction, err := upClient.GetTransaction(eventTransaction.Data.Id)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "", http.StatusInternalServerError)
	}

	// Retrieve account details
	accountId := transaction.Relationships.Account.Data.Id
	account, err := upClient.GetAccount(accountId)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "", http.StatusInternalServerError)
	}

	accountBalance := account.Attributes.Balance.Value

	// Update datastore
	dbClient, _ := database.GetClient(database.Config{})
	defer dbClient.Close()

	dbClient.UpdateAccountBalance(accountBalance)

}
