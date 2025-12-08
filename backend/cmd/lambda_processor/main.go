package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// Telemetry represents device sensor data
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

type AnomalyResult struct {
	IsAnomaly   bool
	AnomalyType string
	Reason      string
}

var (
	ddbClient   *dynamodb.Client
	snsClient   *sns.Client
	tableName   string
	snsTopicARN string
	hrThreshold float64 = 150.0
	tempThreshold float64 = 38.0
)

func init() {
	// Load configuration from environment
	tableName = os.Getenv("DDB_TABLE")
	snsTopicARN = os.Getenv("SNS_TOPIC_ARN")
	
	if tableName == "" {
		log.Fatal("DDB_TABLE environment variable not set")
	}
	if snsTopicARN == "" {
		log.Fatal("SNS_TOPIC_ARN environment variable not set")
	}
	
	// Initialize AWS clients
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Unable to load AWS config: %v", err)
	}
	
	ddbClient = dynamodb.NewFromConfig(cfg)
	snsClient = sns.NewFromConfig(cfg)
	
	log.Printf("Lambda initialized - Table: %s, SNS: %s", tableName, snsTopicARN)
}

func handler(ctx context.Context, kinesisEvent events.KinesisEvent) error {
	log.Printf("Processing %d records", len(kinesisEvent.Records))
	
	for _, record := range kinesisEvent.Records {
		// Parse telemetry
		var telemetry Telemetry
		if err := json.Unmarshal(record.Kinesis.Data, &telemetry); err != nil {
			log.Printf("❌ Failed to unmarshal record: %v", err)
			continue
		}
		
		log.Printf("📥 [%s] HR:%d Temp:%.1f SpO2:%d", 
			telemetry.DeviceID,
			telemetry.Metrics.HeartRate,
			telemetry.Metrics.TempC,
			telemetry.Metrics.SpO2,
		)
		
		// Detect anomalies
		anomaly := detectAnomaly(telemetry)
		
		if anomaly.IsAnomaly {
			log.Printf("⚠️  [%s] ANOMALY: %s - %s", 
				telemetry.DeviceID, 
				anomaly.AnomalyType, 
				anomaly.Reason,
			)
			
			// Send SNS alert
			if err := sendAlert(ctx, telemetry, anomaly); err != nil {
				log.Printf("❌ Failed to send alert: %v", err)
			}
		}
		
		// Store in DynamoDB
		if err := storeTelemetry(ctx, telemetry, anomaly); err != nil {
			log.Printf("❌ Failed to store telemetry: %v", err)
			return err // Return error to retry
		}
	}
	
	log.Printf("✅ Successfully processed %d records", len(kinesisEvent.Records))
	return nil
}

func detectAnomaly(t Telemetry) AnomalyResult {
	// Tachycardia check
	if float64(t.Metrics.HeartRate) > hrThreshold {
		return AnomalyResult{
			IsAnomaly:   true,
			AnomalyType: "tachycardia",
			Reason:      fmt.Sprintf("Heart rate %d exceeds threshold %.0f", t.Metrics.HeartRate, hrThreshold),
		}
	}
	
	// Fever check
	if t.Metrics.TempC >= tempThreshold {
		return AnomalyResult{
			IsAnomaly:   true,
			AnomalyType: "fever",
			Reason:      fmt.Sprintf("Temperature %.1f°C exceeds threshold %.1f°C", t.Metrics.TempC, tempThreshold),
		}
	}
	
	// Hypoxia check
	if t.Metrics.SpO2 < 90 {
		return AnomalyResult{
			IsAnomaly:   true,
			AnomalyType: "hypoxia",
			Reason:      fmt.Sprintf("SpO2 %d%% below threshold 90%%", t.Metrics.SpO2),
		}
	}
	
	return AnomalyResult{IsAnomaly: false}
}

func storeTelemetry(ctx context.Context, t Telemetry, anomaly AnomalyResult) error {
	// Create keys
	pk := fmt.Sprintf("TENANT#%s#DEVICE#%s", t.TenantID, t.DeviceID)
	sk := fmt.Sprintf("TS#%s", t.Timestamp)
	
	// TTL: 30 days from now
	ttl := time.Now().Add(30 * 24 * time.Hour).Unix()
	
	// Build item
	item := map[string]types.AttributeValue{
		"PK":          &types.AttributeValueMemberS{Value: pk},
		"SK":          &types.AttributeValueMemberS{Value: sk},
		"tenant_id":   &types.AttributeValueMemberS{Value: t.TenantID},
		"device_id":   &types.AttributeValueMemberS{Value: t.DeviceID},
		"timestamp":   &types.AttributeValueMemberS{Value: t.Timestamp},
		"hr_bpm":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", t.Metrics.HeartRate)},
		"temp_c":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%.2f", t.Metrics.TempC)},
		"spo2_pct":    &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", t.Metrics.SpO2)},
		"steps":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", t.Metrics.Steps)},
		"battery_pct": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", t.BatteryPct)},
		"fw_version":  &types.AttributeValueMemberS{Value: t.FWVersion},
		"anomaly_flag": &types.AttributeValueMemberBOOL{Value: anomaly.IsAnomaly},
		"ttl":         &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttl)},
	}
	
	if anomaly.IsAnomaly {
		item["anomaly_type"] = &types.AttributeValueMemberS{Value: anomaly.AnomalyType}
	}
	
	// Put item
	_, err := ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	
	return err
}

func sendAlert(ctx context.Context, t Telemetry, anomaly AnomalyResult) error {
	message := fmt.Sprintf(`🚨 HEALTH ALERT 🚨

Device: %s
Tenant: %s
Timestamp: %s

Anomaly Type: %s
Details: %s

Vitals:
- Heart Rate: %d bpm
- Temperature: %.1f°C
- SpO2: %d%%
- Steps: %d
- Battery: %d%%

Action Required: Please check patient immediately.`,
		t.DeviceID,
		t.TenantID,
		t.Timestamp,
		anomaly.AnomalyType,
		anomaly.Reason,
		t.Metrics.HeartRate,
		t.Metrics.TempC,
		t.Metrics.SpO2,
		t.Metrics.Steps,
		t.BatteryPct,
	)
	
	subject := fmt.Sprintf("[HealthSense] %s Alert - %s", anomaly.AnomalyType, t.DeviceID)
	
	_, err := snsClient.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(snsTopicARN),
		Subject:  aws.String(subject),
		Message:  aws.String(message),
	})
	
	return err
}

func main() {
	lambda.Start(handler)
}