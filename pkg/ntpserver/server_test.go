package ntpserver

import (
	"context"
	"net"
	"testing"
	"time"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func TestServer_RespondsToClientRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	srv := New(Config{
		ListenAddr: "127.0.0.1:0",
		Clock:      fixedClock{t: now},
		Stratum:    2,
		// Disable rate limiting to keep test deterministic.
		RateLimitPerSecond: 0,
		RateLimitBurst:     0,
	})

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	raddr, err := net.ResolveUDPAddr("udp", srv.Addr())
	if err != nil {
		t.Fatalf("resolve server addr: %v", err)
	}

	c, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = c.Close() }()

	req := Packet{VN: 4, Mode: ModeClient, Poll: 6, Transmit: timeToTimestamp(now)}
	if _, err := c.Write(req.Marshal()); err != nil {
		t.Fatalf("write: %v", err)
	}

	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	resp, ok := ParsePacket(buf[:n])
	if !ok {
		t.Fatalf("expected parse ok")
	}
	if resp.Mode != ModeServer {
		t.Fatalf("unexpected mode: got=%d want=%d", resp.Mode, ModeServer)
	}
	if resp.VN != 4 {
		t.Fatalf("unexpected version: got=%d want=%d", resp.VN, 4)
	}
	if resp.Stratum != 2 {
		t.Fatalf("unexpected stratum: got=%d want=%d", resp.Stratum, 2)
	}
	if resp.Originate != req.Transmit {
		t.Fatalf("unexpected originate: got=%d want=%d", resp.Originate, req.Transmit)
	}

	m := srv.Metrics()
	if m.TotalRequests == 0 {
		t.Fatalf("expected requests > 0")
	}
	if m.TotalResponses == 0 {
		t.Fatalf("expected responses > 0")
	}
}
