package api

import (
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("retrieve-account-balance", retrieveAccountBalance)
}

func retrieveAccountBalance(w http.ResponseWriter, r *http.Request) {
	// Retrieve current account balance

	// 1. Read from firestore
	// 2. Return value
}
