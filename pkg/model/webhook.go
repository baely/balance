package model

type WebhookEvent struct {
	TransactionDescription string `json:"transaction_description"`
	TransactionAmount      string `json:"transaction_amount"`
	AccountBalance         string `json:"account_balance"`
}

type RawWebhookEvent struct {
	Account     AccountResource     `json:"account"`
	Transaction TransactionResource `json:"transaction"`
}
