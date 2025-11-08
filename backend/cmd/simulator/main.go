package main

import ("context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Telemetry represents device sensor data
type Telemetry struct {
	TenantID   string    `json:"tenant_id"`
	DeviceID   string    `json:"device_id"`
	Timestamp  string    `json:"ts"`
	Metrics    Metrics   `json:"metrics"`
	BatteryPct int       `json:"battery_pct"`
	FWVersion  string    `json:"fw_version"`
}

type Metrics struct {
	HeartRate int     `json:"hr_bpm"`
	TempC     float64 `json:"temp_c"`
	SpO2      int     `json:"spo2_pct"`
	Steps     int     `json:"steps"`
}

func main() {
	// Command-line flags
	broker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	numDevices := flag.Int("devices", 5, "Number of simulated devices")
	interval := flag.Duration("interval", 2*time.Second, "Publishing interval")
	tenantID := flag.String("tenant", "acme-clinic", "Tenant ID")
	flag.Parse()

	log.Printf("Starting HealthSense Simulator")
	log.Printf("   Broker: %s", *broker)
	log.Printf("   Devices: %d", *numDevices)
	log.Printf("   Interval: %v", *interval)
	log.Printf("   Tenant: %s", *tenantID)

	// MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(*broker)
	opts.SetClientID(fmt.Sprintf("simulator-%d", time.Now().Unix()))
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetAutoReconnect(true)

	// Connect to broker
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to broker: %v", token.Error())
	}
	log.Printf("Connected to MQTT broker")

	// Wait group for graceful shutdown
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	// Start device goroutines
	for i := 0; i < *numDevices; i++ {
		wg.Add(1)
		deviceID := fmt.Sprintf("watch-%04d", i)
		go publishTelemetry(ctx, &wg, client, *tenantID, deviceID, *interval)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down simulator...")
	cancel()
	wg.Wait()
	client.Disconnect(250)
	log.Println("Simulator stopped")
}

func publishTelemetry(ctx context.Context, wg *sync.WaitGroup, client mqtt.Client, tenantID, deviceID string, interval time.Duration) {
	defer wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initialize baseline vitals (realistic values)
	baseHR := 70 + rand.Intn(30)      // 70-100 bpm
	baseTemp := 36.5 + rand.Float64() // 36.5-37.5°C
	baseSpO2 := 95 + rand.Intn(5)     // 95-100%
	steps := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Generate telemetry with small variations
			telemetry := Telemetry{
				TenantID:  tenantID,
				DeviceID:  deviceID,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Metrics: Metrics{
					HeartRate: baseHR + rand.Intn(21) - 10,              // ±10 bpm
					TempC:     baseTemp + (rand.Float64()*0.4 - 0.2),   // ±0.2°C
					SpO2:      baseSpO2 + rand.Intn(3) - 1,              // ±1%
					Steps:     steps + rand.Intn(50),                    // incremental
				},
				BatteryPct: 100 - rand.Intn(30), // 70-100%
				FWVersion:  "1.3.2",
			}
			steps = telemetry.Metrics.Steps

			// Occasionally simulate anomalies (10% chance)
			if rand.Float32() < 0.1 {
				telemetry.Metrics.HeartRate = 150 + rand.Intn(30) // Tachycardia
				telemetry.Metrics.TempC = 38.0 + rand.Float64()   // Fever
			}

			// Publish to MQTT
			topic := fmt.Sprintf("tenants/%s/devices/%s/telemetry", tenantID, deviceID)
			payload, _ := json.Marshal(telemetry)

			token := client.Publish(topic, 1, false, payload)
			token.Wait()

			if token.Error() != nil {
				log.Printf("[%s] Publish error: %v", deviceID, token.Error())
			} else {
				log.Printf("[%s] HR:%d Temp:%.1f SpO2:%d Steps:%d",
					deviceID,
					telemetry.Metrics.HeartRate,
					telemetry.Metrics.TempC,
					telemetry.Metrics.SpO2,
					telemetry.Metrics.Steps,
				)
			}
		}
	}
}