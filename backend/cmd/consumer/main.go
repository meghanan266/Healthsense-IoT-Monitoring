package main

import (
	"bytes"
	"net/http"
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/meghanan266/healthsense/backend/pkg/anomaly"
	"github.com/meghanan266/healthsense/backend/pkg/cache"
	"github.com/meghanan266/healthsense/backend/pkg/db"
)

// Telemetry matches simulator output
type Telemetry struct {
	TenantID   string  `json:"tenant_id"`
	DeviceID   string  `json:"device_id"`
	Timestamp  string  `json:"ts"`
	Metrics    Metrics `json:"metrics"`
	BatteryPct int     `json:"battery_pct"`
	FWVersion  string  `json:"fw_version"`
}

type Metrics struct {
	HeartRate int     `json:"hr_bpm"`
	TempC     float64 `json:"temp_c"`
	SpO2      int     `json:"spo2_pct"`
	Steps     int     `json:"steps"`
}

// Add this function to push telemetry to API for WebSocket broadcast
func pushToWebSocket(telemetry Telemetry) {
	apiURL := "http://localhost:8080/api/v1/internal/broadcast"
	
	payload, err := json.Marshal(map[string]interface{}{
		"type":      "telemetry",
		"device_id": telemetry.DeviceID,
		"tenant_id": telemetry.TenantID,
		"timestamp": telemetry.Timestamp,
		"data": map[string]interface{}{
			"hr_bpm":      telemetry.Metrics.HeartRate,
			"temp_c":      telemetry.Metrics.TempC,
			"spo2_pct":    telemetry.Metrics.SpO2,
			"steps":       telemetry.Metrics.Steps,
			"battery_pct": telemetry.BatteryPct,
		},
	})
	
	if err != nil {
		log.Printf("Failed to marshal WebSocket payload: %v", err)
		return
	}
	
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		// Don't log errors - API might not be running, that's ok
		return
	}
	defer resp.Body.Close()
}

func main() {
	// Flags
	broker := flag.String("broker", "tcp://localhost:1883", "MQTT broker")
	topic := flag.String("topic", "tenants/+/devices/+/telemetry", "MQTT topic pattern")
	ddbEndpoint := flag.String("ddb-endpoint", "http://localhost:8000", "DynamoDB endpoint")
	ddbTable := flag.String("ddb-table", "healthsense-telemetry-dev", "DynamoDB table")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	flag.Parse()

	log.Println("Starting HealthSense Consumer")

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

	// Initialize anomaly detector
	detector := anomaly.NewSimpleDetector()

	// MQTT message handler
	messageHandler := func(client mqtt.Client, msg mqtt.Message) {
		var telemetry Telemetry
		if err := json.Unmarshal(msg.Payload(), &telemetry); err != nil {
			log.Printf("Invalid JSON: %v", err)
			return
		}

		log.Printf("[%s] HR:%d Temp:%.1f SpO2:%d",
			telemetry.DeviceID,
			telemetry.Metrics.HeartRate,
			telemetry.Metrics.TempC,
			telemetry.Metrics.SpO2,
		)

		// Detect anomalies
		anomalyResult := detector.Detect(
			telemetry.Metrics.HeartRate,
			telemetry.Metrics.TempC,
			telemetry.Metrics.SpO2,
		)

		if anomalyResult.IsAnomaly {
			log.Printf("[%s] ANOMALY DETECTED: %s - %s",
				telemetry.DeviceID,
				anomalyResult.AnomalyType,
				anomalyResult.Reason,
			)
		}

		// Store in DynamoDB
		record := db.TelemetryRecord{
			TenantID:    telemetry.TenantID,
			DeviceID:    telemetry.DeviceID,
			Timestamp:   telemetry.Timestamp,
			HeartRate:   telemetry.Metrics.HeartRate,
			TempC:       telemetry.Metrics.TempC,
			SpO2:        telemetry.Metrics.SpO2,
			Steps:       telemetry.Metrics.Steps,
			BatteryPct:  telemetry.BatteryPct,
			FWVersion:   telemetry.FWVersion,
			AnomalyFlag: anomalyResult.IsAnomaly,
			AnomalyType: anomalyResult.AnomalyType,
		}

		if err := ddbClient.PutTelemetry(ctx, record); err != nil {
			log.Printf("Failed to store in DynamoDB: %v", err)
		}

		// Cache in Redis
		ts, _ := time.Parse(time.RFC3339, telemetry.Timestamp)
		cacheData := cache.LatestTelemetry{
			DeviceID:   telemetry.DeviceID,
			Timestamp:  ts,
			HeartRate:  telemetry.Metrics.HeartRate,
			TempC:      telemetry.Metrics.TempC,
			SpO2:       telemetry.Metrics.SpO2,
			Steps:      telemetry.Metrics.Steps,
			BatteryPct: telemetry.BatteryPct,
		}

		if err := redisClient.SetLatest(ctx, telemetry.TenantID, telemetry.DeviceID, cacheData); err != nil {
			log.Printf("Failed to cache in Redis: %v", err)
		}

		// Push to WebSocket
		pushToWebSocket(telemetry)
	}

	// Connect to MQTT
	opts := mqtt.NewClientOptions()
	opts.AddBroker(*broker)
	opts.SetClientID("consumer-main")
	opts.SetDefaultPublishHandler(messageHandler)
	opts.SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect: %v", token.Error())
	}

	log.Printf("Connected to MQTT broker")

	// Subscribe
	if token := client.Subscribe(*topic, 1, nil); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to subscribe: %v", token.Error())
	}

	log.Printf("Subscribed to: %s", *topic)
	log.Println("Listening for telemetry...")

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	client.Disconnect(250)
}

