package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisClient struct {
	client *redis.Client
}

// LatestTelemetry represents cached device data
type LatestTelemetry struct {
	DeviceID   string    `json:"device_id"`
	Timestamp  time.Time `json:"timestamp"`
	HeartRate  int       `json:"hr_bpm"`
	TempC      float64   `json:"temp_c"`
	SpO2       int       `json:"spo2_pct"`
	Steps      int       `json:"steps"`
	BatteryPct int       `json:"battery_pct"`
}

// NewRedisClient creates a Redis client
func NewRedisClient(addr string) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // No password for local dev
		DB:       0,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{client: client}, nil
}

// SetLatest caches the latest telemetry for a device
func (r *RedisClient) SetLatest(ctx context.Context, tenantID, deviceID string, data LatestTelemetry) error {
	key := fmt.Sprintf("latest:%s:%s", tenantID, deviceID)
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Cache for 10 minutes
	err = r.client.Set(ctx, key, jsonData, 10*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// GetLatest retrieves cached telemetry
func (r *RedisClient) GetLatest(ctx context.Context, tenantID, deviceID string) (*LatestTelemetry, error) {
	key := fmt.Sprintf("latest:%s:%s", tenantID, deviceID)
	
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("no cached data for device %s", deviceID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	var data LatestTelemetry
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return &data, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}