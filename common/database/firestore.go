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

func GetClient(cfg Config) (*firestore.Client, error) {
	ctx := context.Background()
	firebaseCfg := &firebase.Config{
		ProjectID: cfg.ProjectID,
	}
	app, err := firebase.NewApp(ctx, firebaseCfg)
	if err != nil {
		// Do something
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		// Do something
	}

	return client, nil
}

func (c *Client) Close() {
	c.firestoreClient.Close()
}

func (c *Client) UpdateAccountBalance(value string) {

}
