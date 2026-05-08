package mgoutil

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connect attempts to establish a client with the given dsn
func Connect(ctx context.Context, dsn string) (*mongo.Client, error) {
	client, err := mongo.NewClient(options.Client().
		ApplyURI(dsn).
		SetAppName("bell").
		SetMaxPoolSize(200).
		SetMinPoolSize(20))
	if err != nil {
		return nil, fmt.Errorf("new client: %w", err)
	}

	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	return client, nil
}
