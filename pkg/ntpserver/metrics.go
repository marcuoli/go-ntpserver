package ntpserver

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type metrics struct {
	startedAt atomic.Value // time.Time

	totalRequests  atomic.Uint64
	totalResponses atomic.Uint64
	totalErrors    atomic.Uint64

	lastRequestAt atomic.Value // time.Time
	lastRequestIP atomic.Value // string

	mu      sync.Mutex
	byIP    map[string]uint64
}

func newMetrics() *metrics {
	m := &metrics{byIP: make(map[string]uint64)}
	m.startedAt.Store(time.Time{})
	m.lastRequestAt.Store(time.Time{})
	m.lastRequestIP.Store("")
	return m
}

func (m *metrics) reset(startedAt time.Time) {
	m.totalRequests.Store(0)
	m.totalResponses.Store(0)
	m.totalErrors.Store(0)
	m.startedAt.Store(startedAt)
	m.lastRequestAt.Store(time.Time{})
	m.lastRequestIP.Store("")
	m.mu.Lock()
	m.byIP = make(map[string]uint64)
	m.mu.Unlock()
}

func (m *metrics) incRequest(ip string, at time.Time) {
	m.totalRequests.Add(1)
	m.lastRequestAt.Store(at)
	m.lastRequestIP.Store(ip)
	if ip == "" {
		return
	}
	m.mu.Lock()
	m.byIP[ip]++
	m.mu.Unlock()
}

func (m *metrics) incResponse() {
	m.totalResponses.Add(1)
}

func (m *metrics) incError() {
	m.totalErrors.Add(1)
}

func (m *metrics) snapshot() MetricsSnapshot {
	startedAt, _ := m.startedAt.Load().(time.Time)
	lastAt, _ := m.lastRequestAt.Load().(time.Time)
	lastIP, _ := m.lastRequestIP.Load().(string)

	m.mu.Lock()
	counts := make([]ClientCount, 0, len(m.byIP))
	for ip, c := range m.byIP {
		counts = append(counts, ClientCount{ClientIP: ip, Count: c})
	}
	unique := len(m.byIP)
	m.mu.Unlock()

	sort.Slice(counts, func(i, j int) bool {
		if counts[i].Count == counts[j].Count {
			return counts[i].ClientIP < counts[j].ClientIP
		}
		return counts[i].Count > counts[j].Count
	})
	if len(counts) > 10 {
		counts = counts[:10]
	}

	return MetricsSnapshot{
		StartedAt:      startedAt,
		TotalRequests:  m.totalRequests.Load(),
		TotalResponses: m.totalResponses.Load(),
		TotalErrors:    m.totalErrors.Load(),
		LastRequestAt:  lastAt,
		LastRequestIP:  lastIP,
		UniqueClients:  unique,
		TopClients:     counts,
	}
}
