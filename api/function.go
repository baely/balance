package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/baely/balance/common/database"
)

func init() {
	functions.HTTP("retrieve-account-balance", retrieveAccountBalance)
}

func retrieveAccountBalance(w http.ResponseWriter, r *http.Request) {
	// Retrieve current account balance
	dbClient, err := database.GetClient(database.Config{})
	if err != nil {
		fmt.Println(err)
		http.Error(w, "", http.StatusInternalServerError)
	}

	accountBalance := dbClient.GetAccountBalance()

	// Write account balance to response
	io.WriteString(w, accountBalance)
}
