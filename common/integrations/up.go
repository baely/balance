package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/baely/balance/common/model"
)

const upBaseUri = "https://api.up.com.au/api/v1/"

type UpClient struct {
	accessToken string
}

func NewUpClient(accessToken string) *UpClient {
	return &UpClient{
		accessToken: accessToken,
	}
}

func (c *UpClient) request(endpoint string, ret interface{}) error {
	var b []byte
	r := bytes.NewBuffer(b)

	uri := fmt.Sprintf("%s%s", upBaseUri, endpoint)

	req, err := http.NewRequest(http.MethodGet, uri, r)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	err = json.NewDecoder(resp.Body).Decode(ret)
	if err != nil {
		return err
	}

	return nil
}

func (c *UpClient) GetAccount(accountId string) (model.AccountResource, error) {
	var resp model.GetAccountResponse

	endpoint := fmt.Sprintf("accounts/%s", accountId)

	err := c.request(endpoint, &resp)
	if err != nil {
		return model.AccountResource{}, err
	}

	return resp.Data, nil
}

func (c *UpClient) GetTransaction(transactionId string) (model.TransactionResource, error) {
	var resp model.GetTransactionResponse

	endpoint := fmt.Sprintf("transactions/%s", transactionId)

	err := c.request(endpoint, &resp)
	if err != nil {
		return model.TransactionResource{}, err
	}

	return resp.Data, nil
}
