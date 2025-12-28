package ntpserver

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestServer_Start_ErrAlreadyRunning(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := New(Config{ListenAddr: "127.0.0.1:0", Network: "udp4"})
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	err := srv.Start(ctx)
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("expected ErrAlreadyRunning, got=%v", err)
	}
}

func TestServer_Stop_Idempotent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := New(Config{ListenAddr: "127.0.0.1:0", Network: "udp4"})
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := srv.Stop(); err != nil {
		t.Fatalf("stop1: %v", err)
	}
	if err := srv.Stop(); err != nil {
		t.Fatalf("stop2: %v", err)
	}
}

func TestServer_HookDrop_NoResponseAndErrorEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := New(Config{
		ListenAddr: "127.0.0.1:0",
		Network:    "udp4",
		Hook: func(Packet, RequestMeta) string {
			return "blocked"
		},
		RateLimitPerSecond: 0,
		RateLimitBurst:     0,
	})

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	evCh, unsub := srv.Subscribe()
	defer unsub()

	raddr, err := net.ResolveUDPAddr("udp", srv.Addr())
	if err != nil {
		t.Fatalf("resolve server addr: %v", err)
	}
	c, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = c.Close() }()

	now := time.Now().UTC()
	req := Packet{VN: 4, Mode: ModeClient, Poll: 6, Transmit: timeToTimestamp(now)}
	if _, err := c.Write(req.Marshal()); err != nil {
		t.Fatalf("write: %v", err)
	}

	_ = c.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
	buf := make([]byte, 1024)
	_, rerr := c.Read(buf)
	if rerr == nil {
		t.Fatalf("expected no response")
	}
	if ne, ok := rerr.(net.Error); ok {
		if !ne.Timeout() {
			t.Fatalf("expected timeout, got=%v", rerr)
		}
	}

	select {
	case ev := <-evCh:
		if ev.Error != "blocked" {
			t.Fatalf("event error: got=%q want=%q", ev.Error, "blocked")
		}
		if ev.Responded {
			t.Fatalf("expected Responded=false")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for event")
	}

	m := srv.Metrics()
	if m.TotalRequests == 0 {
		t.Fatalf("expected requests > 0")
	}
	if m.TotalResponses != 0 {
		t.Fatalf("expected responses = 0, got=%d", m.TotalResponses)
	}
	if m.TotalErrors == 0 {
		t.Fatalf("expected errors > 0")
	}
}

func TestServer_InvalidPacket_NoResponseAndInvalidEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := New(Config{ListenAddr: "127.0.0.1:0", Network: "udp4", RateLimitPerSecond: 0, RateLimitBurst: 0})
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	evCh, unsub := srv.Subscribe()
	defer unsub()

	raddr, err := net.ResolveUDPAddr("udp", srv.Addr())
	if err != nil {
		t.Fatalf("resolve server addr: %v", err)
	}
	c, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = c.Close() }()

	if _, err := c.Write([]byte{0x00, 0x01, 0x02}); err != nil {
		t.Fatalf("write: %v", err)
	}

	_ = c.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
	buf := make([]byte, 1024)
	_, rerr := c.Read(buf)
	if rerr == nil {
		t.Fatalf("expected no response")
	}
	if ne, ok := rerr.(net.Error); ok {
		if !ne.Timeout() {
			t.Fatalf("expected timeout, got=%v", rerr)
		}
	}

	select {
	case ev := <-evCh:
		if ev.Error != "invalid_request" {
			t.Fatalf("event error: got=%q want=%q", ev.Error, "invalid_request")
		}
		if ev.PacketValid {
			t.Fatalf("expected PacketValid=false")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for event")
	}
}
