package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bharathbbg/delivery-service/internal/config"
	"github.com/bharathbbg/delivery-service/internal/model"
	"github.com/go-redis/redis/v8"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(config config.RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	return &RedisCache{client: client}, nil
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

func (c *RedisCache) CacheDelivery(ctx context.Context, delivery *model.Delivery) error {
	key := fmt.Sprintf("delivery:%s", delivery.ID)
	data, err := json.Marshal(delivery)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (c *RedisCache) GetCachedDelivery(ctx context.Context, deliveryID string) (*model.Delivery, error) {
	key := fmt.Sprintf("delivery:%s", deliveryID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var delivery model.Delivery
	if err := json.Unmarshal(data, &delivery); err != nil {
		return nil, err
	}

	return &delivery, nil
}

func (c *RedisCache) CacheDeliveryByTracking(ctx context.Context, delivery *model.Delivery) error {
	key := fmt.Sprintf("tracking:%s", delivery.TrackingNumber)
	data, err := json.Marshal(delivery)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (c *RedisCache) GetCachedDeliveryByTracking(ctx context.Context, trackingNumber string) (*model.Delivery, error) {
	key := fmt.Sprintf("tracking:%s", trackingNumber)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var delivery model.Delivery
	if err := json.Unmarshal(data, &delivery); err != nil {
		return nil, err
	}

	return &delivery, nil
}

func (c *RedisCache) CacheDeliveryEvents(ctx context.Context, deliveryID string, events []*model.DeliveryEvent) error {
	key := fmt.Sprintf("delivery_events:%s", deliveryID)
	data, err := json.Marshal(events)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (c *RedisCache) GetCachedDeliveryEvents(ctx context.Context, deliveryID string) ([]*model.DeliveryEvent, error) {
	key := fmt.Sprintf("delivery_events:%s", deliveryID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var events []*model.DeliveryEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}

	return events, nil
}