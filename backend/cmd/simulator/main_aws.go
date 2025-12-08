package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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
	"io/ioutil"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Telemetry represents device sensor data
type TelemetryAWS struct {
	TenantID   string       `json:"tenant_id"`
	DeviceID   string       `json:"device_id"`
	Timestamp  string       `json:"ts"`
	Metrics    MetricsAWS   `json:"metrics"`
	BatteryPct int          `json:"battery_pct"`
	FWVersion  string       `json:"fw_version"`
}

type MetricsAWS struct {
	HeartRate int     `json:"hr_bpm"`
	TempC     float64 `json:"temp_c"`
	SpO2      int     `json:"spo2_pct"`
	Steps     int     `json:"steps"`
}

type MetricsTrackerAWS struct {
	mu             sync.RWMutex
	publishCount   int64
	publishErrors  int64
	startTime      time.Time
}

var globalMetricsAWS *MetricsTrackerAWS

func main() {
	// Command-line flags
	awsEndpoint := flag.String("endpoint", "a3fedtp0rmwbku-ats.iot.us-east-1.amazonaws.com", "AWS IoT Core endpoint")
	certFile := flag.String("cert", "../../../certs/certificate.pem.crt", "Device certificate file")
	keyFile := flag.String("key", "../../../certs/private.pem.key", "Private key file")
	caFile := flag.String("ca", "../../../certs/AmazonRootCA1.pem", "Root CA file")
	numDevices := flag.Int("devices", 5, "Number of simulated devices")
	interval := flag.Duration("interval", 2*time.Second, "Publishing interval")
	tenantID := flag.String("tenant", "acme-clinic", "Tenant ID")
	duration := flag.Duration("duration", 0, "Test duration (0 = infinite)")
	flag.Parse()

	log.Printf("🚀 Starting HealthSense AWS IoT Simulator")
	log.Printf("   Endpoint: %s", *awsEndpoint)
	log.Printf("   Devices: %d", *numDevices)
	log.Printf("   Interval: %v", *interval)
	log.Printf("   Tenant: %s", *tenantID)
	if *duration > 0 {
		log.Printf("   Duration: %v", *duration)
	}

	// Initialize metrics
	globalMetricsAWS = &MetricsTrackerAWS{
		startTime: time.Now(),
	}

	// Start metrics reporter
	go metricsReporterAWS()

	// Load TLS certificates
	tlsConfig, err := newTLSConfig(*certFile, *keyFile, *caFile)
	if err != nil {
		log.Fatalf("❌ Failed to load certificates: %v", err)
	}

	// MQTT client options for AWS IoT
	broker := fmt.Sprintf("ssl://%s:8883", *awsEndpoint)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(fmt.Sprintf("healthsense-simulator-%d", time.Now().Unix()))
	opts.SetTLSConfig(tlsConfig)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)

	// Connect to AWS IoT Core
	client := mqtt.NewClient(opts)
	log.Printf("🔌 Connecting to AWS IoT Core...")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("❌ Failed to connect to AWS IoT Core: %v", token.Error())
	}
	log.Printf("✅ Connected to AWS IoT Core")

	// Wait group for graceful shutdown
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	// If duration is set, auto-cancel after duration
	if *duration > 0 {
		go func() {
			time.Sleep(*duration)
			log.Println("⏰ Test duration reached, shutting down...")
			cancel()
		}()
	}

	// Start device goroutines
	for i := 0; i < *numDevices; i++ {
		wg.Add(1)
		deviceID := fmt.Sprintf("watch-%04d", i)
		go publishTelemetryAWS(ctx, &wg, client, *tenantID, deviceID, *interval)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigChan:
		log.Println("🛑 Received interrupt signal...")
	case <-ctx.Done():
		log.Println("🛑 Context cancelled...")
	}

	cancel()
	wg.Wait()
	client.Disconnect(250)

	// Print final metrics
	printFinalMetrics()
	log.Println("✅ Simulator stopped")
}

func newTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	// Load client certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	// Load CA certificate
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	return tlsConfig, nil
}

func publishTelemetryAWS(ctx context.Context, wg *sync.WaitGroup, client mqtt.Client, tenantID, deviceID string, interval time.Duration) {
	defer wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initialize baseline vitals
	baseHR := 70 + rand.Intn(30)
	baseTemp := 36.5 + rand.Float64()
	baseSpO2 := 95 + rand.Intn(5)
	steps := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			startTime := time.Now()

			// Generate telemetry
			telemetry := TelemetryAWS{
				TenantID:  tenantID,
				DeviceID:  deviceID,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Metrics: MetricsAWS{
					HeartRate: baseHR + rand.Intn(21) - 10,
					TempC:     baseTemp + (rand.Float64()*0.4 - 0.2),
					SpO2:      baseSpO2 + rand.Intn(3) - 1,
					Steps:     steps + rand.Intn(50),
				},
				BatteryPct: 100 - rand.Intn(30),
				FWVersion:  "1.3.2",
			}
			steps = telemetry.Metrics.Steps

			// Occasionally simulate anomalies (10% chance)
			if rand.Float32() < 0.1 {
				telemetry.Metrics.HeartRate = 150 + rand.Intn(30)
				telemetry.Metrics.TempC = 38.0 + rand.Float64()
			}

			// Publish to AWS IoT Core
			topic := fmt.Sprintf("tenants/%s/devices/%s/telemetry", tenantID, deviceID)
			payload, _ := json.Marshal(telemetry)

			token := client.Publish(topic, 1, false, payload)
			token.Wait()

			latencyMs := time.Since(startTime).Milliseconds()
			success := token.Error() == nil

			// Record metrics
			globalMetricsAWS.mu.Lock()
			if success {
				globalMetricsAWS.publishCount++
			} else {
				globalMetricsAWS.publishErrors++
			}
			globalMetricsAWS.mu.Unlock()

			if !success {
				log.Printf("❌ [%s] Publish error: %v", deviceID, token.Error())
			} else if rand.Float32() < 0.05 { // Log 5% of successful publishes
				log.Printf("📤 [%s] HR:%d Temp:%.1f SpO2:%d (latency: %dms)",
					deviceID,
					telemetry.Metrics.HeartRate,
					telemetry.Metrics.TempC,
					telemetry.Metrics.SpO2,
					latencyMs,
				)
			}
		}
	}
}

func metricsReporterAWS() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		globalMetricsAWS.mu.RLock()
		elapsed := time.Since(globalMetricsAWS.startTime).Seconds()
		throughput := float64(globalMetricsAWS.publishCount) / elapsed
		log.Printf("📊 Throughput: %.1f msg/s | Published: %d | Errors: %d",
			throughput,
			globalMetricsAWS.publishCount,
			globalMetricsAWS.publishErrors,
		)
		globalMetricsAWS.mu.RUnlock()
	}
}

func printFinalMetrics() {
	globalMetricsAWS.mu.RLock()
	defer globalMetricsAWS.mu.RUnlock()

	elapsed := time.Since(globalMetricsAWS.startTime).Seconds()
	throughput := float64(globalMetricsAWS.publishCount) / elapsed

	fmt.Println("\n" + "============================================================")
	fmt.Println("AWS SIMULATOR METRICS")
	fmt.Println("============================================================")
	fmt.Printf("Total Published:     %d messages\n", globalMetricsAWS.publishCount)
	fmt.Printf("Total Errors:        %d\n", globalMetricsAWS.publishErrors)
	fmt.Printf("Throughput:          %.2f msg/sec\n", throughput)
	fmt.Printf("Elapsed Time:        %.2f sec\n", elapsed)
	fmt.Println("============================================================")
}