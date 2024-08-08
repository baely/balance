package database

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/iterator"
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

func (c *Client) UpdateAccountBalance(ctx context.Context, value string) error {
	_, err := c.firestoreClient.Collection("balance").Doc("account-balance").Set(ctx, map[string]interface{}{
		"balance": value,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GetAccountBalance(ctx context.Context) (string, error) {
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

func (c *Client) AddWebhook(ctx context.Context, uri string) error {
	_, _, err := c.firestoreClient.Collection("webhooks").Add(ctx, map[string]interface{}{
		"uri": uri,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) getUris(ctx context.Context, path string) ([]string, error) {
	var uris []string

	iter := c.firestoreClient.Collection(path).Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println("error getting doc:", err)
			continue
		}

		data, err := doc.DataAt("uri")
		if err != nil {
			fmt.Println("error getting uri:", err)
			continue
		}

		uri, ok := data.(string)
		if !ok {
			fmt.Println("error parsing uri to string. data:", data)
			continue
		}

		uris = append(uris, uri)
	}

	return uris, nil
}

func (c *Client) GetWebhookUris(ctx context.Context) ([]string, error) {
	return c.getUris(ctx, "webhooks")
}

func (c *Client) GetRawWebhookUris(ctx context.Context) ([]string, error) {
	return c.getUris(ctx, "raw-webhooks")
}
