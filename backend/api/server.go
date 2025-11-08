package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/meghanan266/healthsense/backend/pkg/cache"
	"github.com/meghanan266/healthsense/backend/pkg/db"
)

type Server struct {
	router      *gin.Engine
	ddbClient   *db.DynamoDBClient
	redisClient *cache.RedisClient
	wsHub       *WSHub
}

// NewServer creates and configures the API server
func NewServer(ddbClient *db.DynamoDBClient, redisClient *cache.RedisClient) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// CORS configuration for local development
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	wsHub := NewWSHub()
	go wsHub.Run()

	server := &Server{
		router:      router,
		ddbClient:   ddbClient,
		redisClient: redisClient,
		wsHub:       wsHub,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.handleHealth)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Public routes (no auth for now - will add JWT later)
		v1.GET("/devices", s.handleGetDevices)
		v1.GET("/devices/:deviceId/latest", s.handleGetLatestTelemetry)
		v1.GET("/devices/:deviceId/timeseries", s.handleGetTimeseries)
		
		// WebSocket endpoint
		v1.GET("/ws", s.handleWebSocket)

		// Internal broadcast endpoint (for consumer)
		v1.POST("/internal/broadcast", s.handleInternalBroadcast)
	}
}

// Internal endpoint for consumer to push telemetry for WebSocket broadcast
func (s *Server) handleInternalBroadcast(c *gin.Context) {
	var msg WSMessage
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}
	
	// Broadcast to all connected WebSocket clients
	s.wsHub.Broadcast(msg)
	
	c.JSON(http.StatusOK, gin.H{"status": "broadcasted"})
}

// Start runs the HTTP server
func (s *Server) Start(addr string) error {
	log.Printf("API Server starting on %s", addr)
	return s.router.Run(addr)
}

// Health check handler
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "healthsense-api",
	})
}

// Get all devices with latest telemetry
func (s *Server) handleGetDevices(c *gin.Context) {
	tenantID := c.DefaultQuery("tenant_id", "acme-clinic")
	
	// For now, we'll hardcode device IDs (in production, these would come from a device registry)
	deviceIDs := []string{"watch-0000", "watch-0001", "watch-0002", "watch-0003", "watch-0004"}
	
	ctx := context.Background()
	devices := make([]gin.H, 0)
	
	for _, deviceID := range deviceIDs {
		latest, err := s.redisClient.GetLatest(ctx, tenantID, deviceID)
		if err != nil {
			log.Printf("Failed to get latest for %s: %v", deviceID, err)
			continue
		}
		
		devices = append(devices, gin.H{
			"device_id":   latest.DeviceID,
			"timestamp":   latest.Timestamp,
			"hr_bpm":      latest.HeartRate,
			"temp_c":      latest.TempC,
			"spo2_pct":    latest.SpO2,
			"steps":       latest.Steps,
			"battery_pct": latest.BatteryPct,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"count":     len(devices),
		"devices":   devices,
	})
}

// Get latest telemetry for a specific device
func (s *Server) handleGetLatestTelemetry(c *gin.Context) {
	deviceID := c.Param("deviceId")
	tenantID := c.DefaultQuery("tenant_id", "acme-clinic")
	
	ctx := context.Background()
	latest, err := s.redisClient.GetLatest(ctx, tenantID, deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found or no recent data"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"device_id":   latest.DeviceID,
		"timestamp":   latest.Timestamp,
		"hr_bpm":      latest.HeartRate,
		"temp_c":      latest.TempC,
		"spo2_pct":    latest.SpO2,
		"steps":       latest.Steps,
		"battery_pct": latest.BatteryPct,
	})
}

// Get timeseries data from DynamoDB
func (s *Server) handleGetTimeseries(c *gin.Context) {
	deviceID := c.Param("deviceId")
	tenantID := c.DefaultQuery("tenant_id", "acme-clinic")
	
	// For now, return a simple response
	// In production, this would query DynamoDB with time range filters
	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"tenant_id": tenantID,
		"message":   "Timeseries endpoint - DynamoDB query to be implemented",
		"data":      []gin.H{},
	})
}