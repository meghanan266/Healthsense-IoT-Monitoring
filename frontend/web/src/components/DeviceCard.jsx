import React from 'react';

const DeviceCard = ({ device, isLive = false }) => {
    const getHRColor = (hr) => {
        if (hr > 150) return { bg: '#fee2e2', text: '#dc2626' };
        if (hr > 100) return { bg: '#fef3c7', text: '#d97706' };
        return { bg: '#dcfce7', text: '#16a34a' };
    };

    const getTempColor = (temp) => {
        if (temp >= 38.0) return { bg: '#fee2e2', text: '#dc2626' };
        if (temp >= 37.5) return { bg: '#fef3c7', text: '#d97706' };
        return { bg: '#dcfce7', text: '#16a34a' };
    };

    const getSpO2Color = (spo2) => {
        if (spo2 < 90) return { bg: '#fee2e2', text: '#dc2626' };
        if (spo2 < 95) return { bg: '#fef3c7', text: '#d97706' };
        return { bg: '#dcfce7', text: '#16a34a' };
    };

    const hrColor = getHRColor(device.hr_bpm);
    const tempColor = getTempColor(device.temp_c);
    const spo2Color = getSpO2Color(device.spo2_pct);

    const formatTimestamp = (ts) => {
        if (!ts) return 'N/A';
        const date = new Date(ts);
        return date.toLocaleTimeString();
    };

    const styles = {
        card: {
            backgroundColor: 'white',
            borderRadius: '8px',
            boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
            padding: '1.5rem',
            transition: 'box-shadow 0.2s',
        },
        header: {
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: '1rem',
        },
        deviceId: {
            fontSize: '1.125rem',
            fontWeight: '600',
            color: '#111827',
        },
        liveIndicator: {
            display: 'flex',
            alignItems: 'center',
            fontSize: '0.75rem',
            color: '#16a34a',
        },
        liveDot: {
            width: '8px',
            height: '8px',
            backgroundColor: '#16a34a',
            borderRadius: '50%',
            marginRight: '4px',
            animation: 'pulse 2s infinite',
        },
        vitalsGrid: {
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: '1rem',
        },
        vitalBox: {
            padding: '0.75rem',
            borderRadius: '8px',
        },
        vitalLabel: {
            fontSize: '0.75rem',
            fontWeight: '500',
            marginBottom: '0.25rem',
        },
        vitalValue: {
            fontSize: '1.5rem',
            fontWeight: 'bold',
        },
        vitalUnit: {
            fontSize: '0.75rem',
        },
        footer: {
            marginTop: '1rem',
            paddingTop: '1rem',
            borderTop: '1px solid #e5e7eb',
            display: 'flex',
            justifyContent: 'space-between',
            fontSize: '0.75rem',
            color: '#6b7280',
        },
    };

    return (
        <div style={styles.card}>
            <div style={styles.header}>
                <h3 style={styles.deviceId}>{device.device_id}</h3>
                {isLive && (
                    <span style={styles.liveIndicator}>
                        <span style={styles.liveDot}></span>
                        LIVE
                    </span>
                )}
            </div>

            <div style={styles.vitalsGrid}>
                <div style={{ ...styles.vitalBox, backgroundColor: hrColor.bg, color: hrColor.text }}>
                    <div style={styles.vitalLabel}>Heart Rate</div>
                    <div style={styles.vitalValue}>{device.hr_bpm}</div>
                    <div style={styles.vitalUnit}>bpm</div>
                </div>

                <div style={{ ...styles.vitalBox, backgroundColor: tempColor.bg, color: tempColor.text }}>
                    <div style={styles.vitalLabel}>Temperature</div>
                    <div style={styles.vitalValue}>{device.temp_c?.toFixed(1)}</div>
                    <div style={styles.vitalUnit}>°C</div>
                </div>

                <div style={{ ...styles.vitalBox, backgroundColor: spo2Color.bg, color: spo2Color.text }}>
                    <div style={styles.vitalLabel}>SpO₂</div>
                    <div style={styles.vitalValue}>{device.spo2_pct}</div>
                    <div style={styles.vitalUnit}>%</div>
                </div>

                <div style={{ ...styles.vitalBox, backgroundColor: '#dbeafe', color: '#2563eb' }}>
                    <div style={styles.vitalLabel}>Steps</div>
                    <div style={styles.vitalValue}>{device.steps?.toLocaleString()}</div>
                    <div style={styles.vitalUnit}>steps</div>
                </div>
            </div>

            <div style={styles.footer}>
                <span>Battery: {device.battery_pct}%</span>
                <span>{formatTimestamp(device.timestamp)}</span>
            </div>
        </div>
    );
};

export default DeviceCard;