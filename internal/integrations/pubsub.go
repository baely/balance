package integrations

import (
	"context"
	"os"

	"cloud.google.com/go/pubsub"
)

func GetClient(ctx context.Context) *pubsub.Client {
	client, err := pubsub.NewClient(ctx, os.Getenv("GCP_PROJECT"))
	if err != nil {
		return nil
	}

	return client
}
