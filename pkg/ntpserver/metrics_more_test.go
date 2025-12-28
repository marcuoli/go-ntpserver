package ntpserver

import (
	"testing"
	"time"
)

func TestMetrics_TopClientsSortedAndLimited(t *testing.T) {
	m := newMetrics()
	started := time.Unix(1, 0).UTC()
	m.reset(started)

	// Create 12 unique clients; two of them have higher counts.
	at := time.Unix(2, 0).UTC()
	for i := 0; i < 10; i++ {
		ip := "10.0.0." + string(rune('0'+i))
		m.incRequest(ip, at)
	}
	m.incRequest("1.1.1.1", at)
	m.incRequest("1.1.1.1", at)
	m.incRequest("2.2.2.2", at)
	m.incRequest("2.2.2.2", at)
	m.incRequest("2.2.2.2", at)

	s := m.snapshot()
	if !s.StartedAt.Equal(started) {
		t.Fatalf("startedAt: got=%v want=%v", s.StartedAt, started)
	}
	if s.TotalRequests != 15 {
		t.Fatalf("TotalRequests: got=%d want=%d", s.TotalRequests, 15)
	}
	if s.UniqueClients != 12 {
		t.Fatalf("UniqueClients: got=%d want=%d", s.UniqueClients, 12)
	}
	if len(s.TopClients) != 10 {
		t.Fatalf("TopClients len: got=%d want=%d", len(s.TopClients), 10)
	}
	if s.TopClients[0].ClientIP != "2.2.2.2" || s.TopClients[0].Count != 3 {
		t.Fatalf("top[0]: got=%+v", s.TopClients[0])
	}
	if s.TopClients[1].ClientIP != "1.1.1.1" || s.TopClients[1].Count != 2 {
		t.Fatalf("top[1]: got=%+v", s.TopClients[1])
	}
	if s.LastRequestIP == "" {
		t.Fatalf("LastRequestIP expected non-empty")
	}
	if s.LastRequestAt.IsZero() {
		t.Fatalf("LastRequestAt expected non-zero")
	}
}
