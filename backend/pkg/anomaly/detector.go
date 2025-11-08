package anomaly

import "fmt"

// AnomalyResult represents detection result
type AnomalyResult struct {
	IsAnomaly   bool
	AnomalyType string
	Reason      string
}

// SimpleDetector implements basic rule-based anomaly detection
type SimpleDetector struct {
	// Thresholds
	TachycardiaThreshold float64 // BPM
	FeverThreshold       float64 // Celsius
	LowSpO2Threshold     int     // Percentage
}

// NewSimpleDetector creates a detector with default thresholds
func NewSimpleDetector() *SimpleDetector {
	return &SimpleDetector{
		TachycardiaThreshold: 150.0,
		FeverThreshold:       38.0,
		LowSpO2Threshold:     90,
	}
}

// Detect checks telemetry for anomalies
func (d *SimpleDetector) Detect(hr int, tempC float64, spo2 int) AnomalyResult {
	// Check tachycardia (high heart rate)
	if float64(hr) > d.TachycardiaThreshold {
		return AnomalyResult{
			IsAnomaly:   true,
			AnomalyType: "tachycardia",
			Reason:      fmt.Sprintf("Heart rate %d exceeds threshold %.0f", hr, d.TachycardiaThreshold),
		}
	}

	// Check fever
	if tempC >= d.FeverThreshold {
		return AnomalyResult{
			IsAnomaly:   true,
			AnomalyType: "fever",
			Reason:      fmt.Sprintf("Temperature %.1f°C exceeds threshold %.1f°C", tempC, d.FeverThreshold),
		}
	}

	// Check low oxygen
	if spo2 < d.LowSpO2Threshold {
		return AnomalyResult{
			IsAnomaly:   true,
			AnomalyType: "hypoxia",
			Reason:      fmt.Sprintf("SpO2 %d%% below threshold %d%%", spo2, d.LowSpO2Threshold),
		}
	}

	return AnomalyResult{IsAnomaly: false}
}