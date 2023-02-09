package database

import (
	"context"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
)

type Client struct {
	firestoreClient *firestore.Client
}

func GetClient(projectId string) (*Client, error) {
	ctx := context.Background()
	firebaseCfg := &firebase.Config{
		ProjectID: projectId,
	}
	app, err := firebase.NewApp(ctx, firebaseCfg)
	if err != nil {
		return nil, err
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{
		client,
	}, nil
}

func (c *Client) Close() {
	c.firestoreClient.Close()
}

func (c *Client) UpdateAccountBalance(value string) {

}

func (c *Client) GetAccountBalance() string {
	return "100"
}
