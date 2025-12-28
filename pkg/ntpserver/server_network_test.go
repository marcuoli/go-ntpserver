package ntpserver

import (
	"context"
	"strings"
	"testing"
)

func TestServer_Start_udp4_BindsIPv4(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := New(Config{ListenAddr: "127.0.0.1:0", Network: "udp4"})
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	addr := srv.Addr()
	if !strings.Contains(addr, "127.0.0.1") {
		t.Fatalf("expected IPv4 addr, got=%q", addr)
	}
}

func TestServer_Start_udp6_BindsIPv6OrSkips(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := New(Config{ListenAddr: "[::1]:0", Network: "udp6"})
	if err := srv.Start(ctx); err != nil {
		t.Skipf("ipv6 not available on this system: %v", err)
		return
	}
	defer func() { _ = srv.Stop() }()

	addr := srv.Addr()
	if !strings.Contains(addr, "::1") {
		t.Fatalf("expected IPv6 addr, got=%q", addr)
	}
}
