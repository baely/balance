package integrations

import "github.com/baely/balance/common/model"

type UpClient struct {
}

func NewUpClient() *UpClient {
	return nil
}

func (c *UpClient) GetAccount(accountId string) model.AccountResource {
	return model.AccountResource{}
}

func (c *UpClient) GetTransaction(transactionId string) model.TransactionResource {
	return model.TransactionResource{}
}
