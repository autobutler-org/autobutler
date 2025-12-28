package networking

import (
	"fmt"
	"sync"
	"time"
)

type Metrics struct {
	mu                sync.RWMutex
	uptime            time.Time
	throughputDown    float64
	throughputUp      float64
	latency           float64
	activeClients     int
	encryptedSessions int
	totalSessions     int
	blockedRequests   int
	lastDeviceName    string
	lastDeviceTime    time.Time
}

type MetricsSnapshot struct {
	Uptime            time.Duration `json:"uptime"`
	ThroughputDown    float64       `json:"throughput_down"`
	ThroughputUp      float64       `json:"throughput_up"`
	Latency           float64       `json:"latency"`
	ActiveClients     int           `json:"active_clients"`
	EncryptedSessions int           `json:"encrypted_sessions"`
	TotalSessions     int           `json:"total_sessions"`
	BlockedRequests   int           `json:"blocked_requests"`
	LastDeviceName    string        `json:"last_device_name"`
	LastDeviceTime    string        `json:"last_device_time"`
}

func NewMetrics() *Metrics {
	return &Metrics{
		uptime: time.Now(),
	}
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lastDeviceTime := ""
	if !m.lastDeviceTime.IsZero() {
		duration := time.Since(m.lastDeviceTime)
		if duration < time.Minute {
			lastDeviceTime = "just now"
		} else if duration < time.Hour {
			mins := int(duration.Minutes())
			lastDeviceTime = fmt.Sprintf("%d min ago", mins)
		} else if duration < 24*time.Hour {
			hours := int(duration.Hours())
			lastDeviceTime = fmt.Sprintf("%d hr ago", hours)
		} else {
			days := int(duration.Hours() / 24)
			lastDeviceTime = fmt.Sprintf("%d days ago", days)
		}
	}

	return MetricsSnapshot{
		Uptime:            time.Since(m.uptime),
		ThroughputDown:    m.throughputDown,
		ThroughputUp:      m.throughputUp,
		Latency:           m.latency,
		ActiveClients:     m.activeClients,
		EncryptedSessions: m.encryptedSessions,
		TotalSessions:     m.totalSessions,
		BlockedRequests:   m.blockedRequests,
		LastDeviceName:    m.lastDeviceName,
		LastDeviceTime:    lastDeviceTime,
	}
}

func (m *Metrics) UpdateThroughput(down, up float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.throughputDown = down
	m.throughputUp = up
}

func (m *Metrics) UpdateLatency(latency float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latency = latency
}

func (m *Metrics) UpdateClients(active int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeClients = active
}

func (m *Metrics) UpdateSessions(encrypted, total int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.encryptedSessions = encrypted
	m.totalSessions = total
}

func (m *Metrics) IncrementBlockedRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blockedRequests++
}

func (m *Metrics) RecordDevice(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastDeviceName = name
	m.lastDeviceTime = time.Now()
}
