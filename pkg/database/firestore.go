package database

import (
	"context"
	"fmt"
	"io"

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

func (c *Client) UpdateAccountBalance(value string) error {
	ctx := context.Background()
	_, err := c.firestoreClient.Collection("balance").Doc("account-balance").Set(ctx, map[string]interface{}{
		"balance": value,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GetAccountBalance() (string, error) {
	ctx := context.Background()
	iter, err := c.firestoreClient.Collection("balance").Doc("account-balance").Get(ctx)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	data, err := iter.DataAt("balance")
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	balance, ok := data.(string)
	if !ok {
		return "", fmt.Errorf("casting of data failed. data: %v", data)
	}

	return balance, nil
}

func (c *Client) AddWebhook(uri string) error {
	ctx := context.Background()

	_, _, err := c.firestoreClient.Collection("webhooks").Add(ctx, map[string]interface{}{
		"uri": uri,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GetWebhookUris() ([]string, error) {
	var uris []string

	ctx := context.Background()
	iter := c.firestoreClient.Collection("webhooks").Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == io.EOF {
			break
		}

		data, err := doc.DataAt("uri")
		if err != nil {
			return nil, err
		}

		uri, ok := data.(string)
		if !ok {
			return nil, fmt.Errorf("casting of data failed. data: %v", data)
		}

		uris = append(uris, uri)
	}

	return uris, nil
}
