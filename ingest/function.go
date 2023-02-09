package ingest

import (
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("ingest-transactions", ingestTransactions)
}

func ingestTransactions(w http.ResponseWriter, r *http.Request) {
	// Process incoming transactions

	// 1. Push to pubsub
	// 2. Return 200
}
