package database

import (
	"context"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
)

type Config struct {
	ProjectID string
}

type Client struct {
	firestoreClient *firestore.Client
}

func GetClient(cfg Config) (*Client, error) {
	ctx := context.Background()
	firebaseCfg := &firebase.Config{
		ProjectID: cfg.ProjectID,
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
