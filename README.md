# HealthSense - Real-Time IoT Health Monitoring

## Local System Architecture
```
Simulator (MQTT) → Mosquitto → Consumer → DynamoDB Local + Redis
                                     ↓
                                  API Server (REST + WebSocket)
                                     ↓
                                React Dashboard
```

## Running Locally

### Prerequisites
- Docker Desktop
- Go 1.25+
- Node.js 22+
- AWS CLI

### Start Services

1. **Start Docker containers:**
```bash
   docker-compose up -d
```

2. **Start Simulator (Terminal 1):**
```bash
   cd backend/cmd/simulator
   go run main.go -devices 5 -interval 2s
```

3. **Start Consumer (Terminal 2):**
```bash
   cd backend/cmd/consumer
   go run main.go
```

4. **Start API Server (Terminal 3):**
```bash
   cd backend/cmd/api
   go run main.go
```

5. **Start Frontend (Terminal 4):**
```bash
   cd frontend/web
   npm run dev
```

6. **Open Dashboard:**
   http://localhost:5173

## Features Implemented

- MQTT telemetry simulation (5 devices, 2s interval)
- Real-time anomaly detection (tachycardia, fever, hypoxia)
- DynamoDB storage with TTL
- Redis caching for fast queries
- WebSocket real-time updates
- Color-coded dashboard with live indicators
- Polling vs WebSocket toggle

## Anomaly Thresholds

- Tachycardia: HR > 150 bpm
- Fever: Temp ≥ 38.0°C
- Hypoxia: SpO2 < 90%

## API Endpoints

- `GET /health` - Health check
- `GET /api/v1/devices` - List all devices with latest data
- `GET /api/v1/devices/:id/latest` - Get specific device latest
- `GET /api/v1/ws` - WebSocket endpoint for live updates
- `POST /api/v1/internal/broadcast` - Internal broadcast endpoint

## Next: AWS Deployment

See `docs/aws-deployment.md`