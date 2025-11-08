package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBClient struct {
	client    *dynamodb.Client
	tableName string
}

// Telemetry record structure
type TelemetryRecord struct {
	TenantID    string
	DeviceID    string
	Timestamp   string
	HeartRate   int
	TempC       float64
	SpO2        int
	Steps       int
	BatteryPct  int
	FWVersion   string
	AnomalyFlag bool
	AnomalyType string
}

// NewDynamoDBClient creates a new DynamoDB client
// For local dev, use endpoint=http://localhost:8000
// For AWS, use endpoint=""
func NewDynamoDBClient(ctx context.Context, endpoint, region, tableName string) (*DynamoDBClient, error) {
	var cfg aws.Config
	var err error

	if endpoint != "" {
		// Local DynamoDB
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: endpoint}, nil
				})),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "")),
		)
	} else {
		// AWS DynamoDB
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(region))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	log.Printf("DynamoDB client initialized (table: %s, endpoint: %s)", tableName, endpoint)

	return &DynamoDBClient{
		client:    client,
		tableName: tableName,
	}, nil
}

// PutTelemetry stores a telemetry record
func (d *DynamoDBClient) PutTelemetry(ctx context.Context, record TelemetryRecord) error {
	// Partition Key: TENANT#tenant_id#DEVICE#device_id
	// Sort Key: TS#timestamp
	pk := fmt.Sprintf("TENANT#%s#DEVICE#%s", record.TenantID, record.DeviceID)
	sk := fmt.Sprintf("TS#%s", record.Timestamp)

	// TTL: 30 days from now (Unix timestamp)
	ttl := time.Now().Add(30 * 24 * time.Hour).Unix()

	item := map[string]types.AttributeValue{
		"PK":           &types.AttributeValueMemberS{Value: pk},
		"SK":           &types.AttributeValueMemberS{Value: sk},
		"tenant_id":    &types.AttributeValueMemberS{Value: record.TenantID},
		"device_id":    &types.AttributeValueMemberS{Value: record.DeviceID},
		"timestamp":    &types.AttributeValueMemberS{Value: record.Timestamp},
		"hr_bpm":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", record.HeartRate)},
		"temp_c":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%.2f", record.TempC)},
		"spo2_pct":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", record.SpO2)},
		"steps":        &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", record.Steps)},
		"battery_pct":  &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", record.BatteryPct)},
		"fw_version":   &types.AttributeValueMemberS{Value: record.FWVersion},
		"anomaly_flag": &types.AttributeValueMemberBOOL{Value: record.AnomalyFlag},
		"ttl":          &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttl)},
	}

	if record.AnomalyType != "" {
		item["anomaly_type"] = &types.AttributeValueMemberS{Value: record.AnomalyType}
	}

	_, err := d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}

	return nil
}

// GetLatestTelemetry retrieves the most recent reading for a device
func (d *DynamoDBClient) GetLatestTelemetry(ctx context.Context, tenantID, deviceID string) (*TelemetryRecord, error) {
	pk := fmt.Sprintf("TENANT#%s#DEVICE#%s", tenantID, deviceID)

	result, err := d.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.tableName),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: pk},
		},
		ScanIndexForward: aws.Bool(false), // Descending order (latest first)
		Limit:            aws.Int32(1),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("no data found for device %s", deviceID)
	}

	// Parse the result (simplified - production code would be more robust)
	item := result.Items[0]
	
	return &TelemetryRecord{
		TenantID:  tenantID,
		DeviceID:  deviceID,
		Timestamp: item["timestamp"].(*types.AttributeValueMemberS).Value,
		// Add more fields as needed
	}, nil
}