import { useEffect, useRef, useState } from 'react';
import { connectWebSocket } from '../services/api';

export const useWebSocket = (tenantId = 'acme-clinic') => {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState(null);
  const [latency, setLatency] = useState(0);
  const wsRef = useRef(null);
  const reconnectTimeoutRef = useRef(null);

  useEffect(() => {
    const connect = () => {
      try {
        const ws = connectWebSocket(tenantId);
        wsRef.current = ws;

        ws.onopen = () => {
          console.log('WebSocket connected');
          setIsConnected(true);
          
          // Subscribe to all devices (you can customize this)
          for (let i = 0; i < 5; i++) {
            ws.send(JSON.stringify({
              type: 'subscribe',
              device_id: `watch-000${i}`,
              tenant_id: tenantId,
            }));
          }
        };

        ws.onmessage = (event) => {
          const receiveTime = Date.now();
          try {
            const message = JSON.parse(event.data);
            setLastMessage(message);
            
            // Calculate latency if timestamp is present
            if (message.timestamp) {
              const sentTime = new Date(message.timestamp).getTime();
              setLatency(receiveTime - sentTime);
            }
          } catch (error) {
            console.error('Failed to parse WebSocket message:', error);
          }
        };

        ws.onerror = (error) => {
          console.error('WebSocket error:', error);
        };

        ws.onclose = () => {
          console.log('WebSocket disconnected');
          setIsConnected(false);
          
          // Attempt to reconnect after 3 seconds
          reconnectTimeoutRef.current = setTimeout(() => {
            console.log('Attempting to reconnect...');
            connect();
          }, 3000);
        };
      } catch (error) {
        console.error('Failed to connect WebSocket:', error);
      }
    };

    connect();

    // Cleanup
    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [tenantId]);

  return { isConnected, lastMessage, latency };
};