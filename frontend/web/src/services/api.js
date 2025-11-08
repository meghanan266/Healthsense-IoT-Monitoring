import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';
const WS_BASE_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/api/v1';

// Create axios instance
const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// API methods
export const healthCheck = () => api.get('/health');

export const getDevices = (tenantId = 'acme-clinic') => 
  api.get('/devices', { params: { tenant_id: tenantId } });

export const getDeviceLatest = (deviceId, tenantId = 'acme-clinic') =>
  api.get(`/devices/${deviceId}/latest`, { params: { tenant_id: tenantId } });

export const getDeviceTimeseries = (deviceId, params = {}) =>
  api.get(`/devices/${deviceId}/timeseries`, { params });

// WebSocket connection
export const connectWebSocket = (tenantId = 'acme-clinic') => {
  const ws = new WebSocket(`${WS_BASE_URL}/ws?tenant_id=${tenantId}`);
  return ws;
};

export default api;