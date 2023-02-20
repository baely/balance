package integrations

import (
	"context"

	"cloud.google.com/go/pubsub"
)

func GetClient() *pubsub.Client {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, "baileybutler-syd")
	if err != nil {
		return nil
	}

	return client
}
