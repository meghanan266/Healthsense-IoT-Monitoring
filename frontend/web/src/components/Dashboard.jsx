import React, { useEffect, useState } from 'react';
import { getDevices } from '../services/api';
import { useWebSocket } from '../hooks/useWebSocket';
import DeviceCard from './DeviceCard';

const Dashboard = () => {
    const [devices, setDevices] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [usePolling, setUsePolling] = useState(false);

    const { isConnected, lastMessage, latency } = useWebSocket('acme-clinic');

    useEffect(() => {
        fetchDevices();
    }, []);

    useEffect(() => {
        if (usePolling) {
            const interval = setInterval(fetchDevices, 2000);
            return () => clearInterval(interval);
        }
    }, [usePolling]);

    useEffect(() => {
        if (!usePolling && lastMessage && lastMessage.device_id) {
            updateDeviceData(lastMessage);
        }
    }, [lastMessage, usePolling]);

    const fetchDevices = async () => {
        try {
            const response = await getDevices('acme-clinic');
            setDevices(response.data.devices || []);
            setLoading(false);
        } catch (err) {
            console.error('Failed to fetch devices:', err);
            setError(err.message);
            setLoading(false);
        }
    };

    const updateDeviceData = (message) => {
        setDevices((prev) =>
            prev.map((device) =>
                device.device_id === message.device_id
                    ? { ...device, ...message.data, timestamp: message.timestamp }
                    : device
            )
        );
    };

    const styles = {
        container: {
            minHeight: '100vh',
            backgroundColor: '#f9fafb',
            fontFamily: 'system-ui, -apple-system, sans-serif',
        },
        header: {
            backgroundColor: 'white',
            boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
            padding: '1.5rem',
        },
        headerContent: {
            maxWidth: '1280px',
            margin: '0 auto',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
        },
        title: {
            fontSize: '2rem',
            fontWeight: 'bold',
            color: '#111827',
            margin: 0,
        },
        subtitle: {
            fontSize: '0.875rem',
            color: '#6b7280',
            marginTop: '0.25rem',
        },
        statusBar: {
            display: 'flex',
            alignItems: 'center',
            gap: '1rem',
        },
        statusDot: {
            width: '12px',
            height: '12px',
            borderRadius: '50%',
            backgroundColor: isConnected ? '#10b981' : '#ef4444',
        },
        main: {
            maxWidth: '1280px',
            margin: '0 auto',
            padding: '2rem 1.5rem',
        },
        statsGrid: {
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
            gap: '1rem',
            marginBottom: '2rem',
        },
        statCard: {
            backgroundColor: 'white',
            borderRadius: '8px',
            boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
            padding: '1rem',
        },
        statLabel: {
            fontSize: '0.875rem',
            color: '#6b7280',
            marginBottom: '0.5rem',
        },
        statValue: {
            fontSize: '1.875rem',
            fontWeight: 'bold',
        },
        devicesGrid: {
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
            gap: '1.5rem',
        },
        toggleButton: {
            padding: '0.5rem 1rem',
            borderRadius: '6px',
            border: '1px solid #d1d5db',
            backgroundColor: usePolling ? '#ef4444' : '#10b981',
            color: 'white',
            cursor: 'pointer',
            fontSize: '0.875rem',
            fontWeight: '500',
        },
    };

    if (loading) {
        return (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100vh' }}>
                <div style={{ fontSize: '1.5rem', color: '#6b7280' }}>Loading devices...</div>
            </div>
        );
    }

    if (error) {
        return (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100vh' }}>
                <div style={{ fontSize: '1.5rem', color: '#ef4444' }}>Error: {error}</div>
            </div>
        );
    }

    return (
        <div style={styles.container}>
            <header style={styles.header}>
                <div style={styles.headerContent}>
                    <div>
                        <h1 style={styles.title}>HealthSense Dashboard</h1>
                        <p style={styles.subtitle}>Real-time IoT Health Monitoring</p>
                    </div>

                    <div style={styles.statusBar}>
                        <button
                            onClick={() => setUsePolling(!usePolling)}
                            style={styles.toggleButton}
                        >
                            {usePolling ? 'Polling Mode' : 'WebSocket Mode'}
                        </button>

                        {!usePolling && (
                            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                                <span style={styles.statusDot}></span>
                                <span style={{ fontSize: '0.875rem', color: '#6b7280' }}>
                                    {isConnected ? 'Connected' : 'Disconnected'}
                                    {isConnected && latency > 0 && ` (${latency}ms)`}
                                </span>
                            </div>
                        )}
                    </div>
                </div>
            </header>

            <main style={styles.main}>
                <div style={styles.statsGrid}>
                    <div style={styles.statCard}>
                        <div style={styles.statLabel}>Total Devices</div>
                        <div style={{ ...styles.statValue, color: '#111827' }}>{devices.length}</div>
                    </div>
                    <div style={styles.statCard}>
                        <div style={styles.statLabel}>Active</div>
                        <div style={{ ...styles.statValue, color: '#10b981' }}>{devices.length}</div>
                    </div>
                    <div style={styles.statCard}>
                        <div style={styles.statLabel}>Mode</div>
                        <div style={{ ...styles.statValue, color: '#3b82f6' }}>
                            {usePolling ? 'Polling' : 'WebSocket'}
                        </div>
                    </div>
                    <div style={styles.statCard}>
                        <div style={styles.statLabel}>Latency</div>
                        <div style={{ ...styles.statValue, color: '#8b5cf6' }}>
                            {latency > 0 ? `${latency}ms` : 'N/A'}
                        </div>
                    </div>
                </div>

                <div style={styles.devicesGrid}>
                    {devices.map((device) => (
                        <DeviceCard
                            key={device.device_id}
                            device={device}
                            isLive={!usePolling && isConnected}
                        />
                    ))}
                </div>

                {devices.length === 0 && (
                    <div style={{ textAlign: 'center', padding: '3rem' }}>
                        <p style={{ fontSize: '1.125rem', color: '#6b7280' }}>No devices found</p>
                        <p style={{ fontSize: '0.875rem', color: '#9ca3af', marginTop: '0.5rem' }}>
                            Make sure the simulator is running
                        </p>
                    </div>
                )}
            </main>
        </div>
    );
};

export default Dashboard;