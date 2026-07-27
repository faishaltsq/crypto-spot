package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func NewCache(addr, password string, db int) *Cache {
	return &Cache{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) SaveScanner(ctx context.Context, features []domain.FeatureSnapshot) error {
	data, err := json.Marshal(features)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, "scanner:latest", data, 30*time.Second).Err()
}

func (c *Cache) LoadScanner(ctx context.Context) ([]domain.FeatureSnapshot, error) {
	data, err := c.client.Get(ctx, "scanner:latest").Bytes()
	if err != nil {
		return nil, err
	}
	var features []domain.FeatureSnapshot
	if err := json.Unmarshal(data, &features); err != nil {
		return nil, err
	}
	return features, nil
}
