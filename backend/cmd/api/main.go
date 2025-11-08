package main

import (
	"context"
	"flag"
	"log"

	"github.com/meghanan266/healthsense/backend/api"
	"github.com/meghanan266/healthsense/backend/pkg/cache"
	"github.com/meghanan266/healthsense/backend/pkg/db"
)

func main() {
	// Flags
	port := flag.String("port", ":8080", "API server port")
	ddbEndpoint := flag.String("ddb-endpoint", "http://localhost:8000", "DynamoDB endpoint")
	ddbTable := flag.String("ddb-table", "healthsense-telemetry-dev", "DynamoDB table")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	flag.Parse()

	ctx := context.Background()

	// Initialize DynamoDB
	ddbClient, err := db.NewDynamoDBClient(ctx, *ddbEndpoint, "us-east-1", *ddbTable)
	if err != nil {
		log.Fatalf("Failed to create DynamoDB client: %v", err)
	}

	// Initialize Redis
	redisClient, err := cache.NewRedisClient(*redisAddr)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redisClient.Close()

	// Create and start server
	server := api.NewServer(ddbClient, redisClient)
	
	log.Printf("API Documentation:")
	log.Printf("   GET  /health")
	log.Printf("   GET  /api/v1/devices")
	log.Printf("   GET  /api/v1/devices/:id/latest")
	log.Printf("   GET  /api/v1/devices/:id/timeseries")
	log.Printf("   GET  /api/v1/ws (WebSocket)")
	
	if err := server.Start(*port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}