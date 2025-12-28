package ntpserver

import "testing"

func TestConfig_normalize_Defaults(t *testing.T) {
	cfg := Config{}.normalize()
	if cfg.ListenAddr != "0.0.0.0:123" {
		t.Fatalf("ListenAddr default: got=%q want=%q", cfg.ListenAddr, "0.0.0.0:123")
	}
	if cfg.Network != "udp" {
		t.Fatalf("Network default: got=%q want=%q", cfg.Network, "udp")
	}
	if cfg.Clock == nil {
		t.Fatalf("Clock default: expected non-nil")
	}
	if cfg.Stratum != 2 {
		t.Fatalf("Stratum default: got=%d want=%d", cfg.Stratum, 2)
	}
	if cfg.RefID == 0 {
		t.Fatalf("RefID default: expected non-zero")
	}
	if cfg.Precision != -20 {
		t.Fatalf("Precision default: got=%d want=%d", cfg.Precision, -20)
	}
	if cfg.EventBuffer != 128 {
		t.Fatalf("EventBuffer default: got=%d want=%d", cfg.EventBuffer, 128)
	}
	if cfg.HistorySize != 500 {
		t.Fatalf("HistorySize default: got=%d want=%d", cfg.HistorySize, 500)
	}
	if cfg.RateLimitBurst != 5 {
		t.Fatalf("RateLimitBurst default: got=%d want=%d", cfg.RateLimitBurst, 5)
	}
}
