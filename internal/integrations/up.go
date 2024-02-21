package integrations

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/baely/balance/internal/model"
)

const upBaseUri = "https://api.up.com.au/api/v1/"

type UpClient struct {
	accessToken string
	client      *http.Client
}

func NewUpClient(accessToken string) *UpClient {
	return &UpClient{
		accessToken: accessToken,
		client:      &http.Client{},
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

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
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

func ValidateWebhookEvent(payload []byte, signature string) bool {
	sig, _ := hex.DecodeString(signature)

	secret := os.Getenv("UP_WEBHOOK_SECRET")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)

	calculatedSignature := mac.Sum(nil)

	return hmac.Equal(sig, calculatedSignature)

}
