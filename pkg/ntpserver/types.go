package ntpserver

import "time"

// Clock abstracts a time source for the server.
// This keeps testing easy and allows future replacement with a more accurate clock.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// RequestEvent captures a single UDP request as observed by the server.
// It is meant for logging/monitoring and future integrations.
type RequestEvent struct {
	At             time.Time `json:"at"`
	ClientAddr     string    `json:"client_addr"`
	ClientIP       string    `json:"client_ip"`
	ClientPort     int       `json:"client_port"`
	Version        uint8     `json:"version"`
	Mode           uint8     `json:"mode"`
	PacketValid    bool      `json:"packet_valid"`
	Responded      bool      `json:"responded"`
	Error          string    `json:"error,omitempty"`
	ProcessingUSec int64     `json:"processing_usec"`
}

type ClientCount struct {
	ClientIP string `json:"client_ip"`
	Count    uint64 `json:"count"`
}

type MetricsSnapshot struct {
	StartedAt      time.Time     `json:"started_at"`
	TotalRequests  uint64        `json:"total_requests"`
	TotalResponses uint64        `json:"total_responses"`
	TotalErrors    uint64        `json:"total_errors"`
	LastRequestAt  time.Time     `json:"last_request_at"`
	LastRequestIP  string        `json:"last_request_ip"`
	UniqueClients  int           `json:"unique_clients"`
	TopClients     []ClientCount `json:"top_clients"`
}

// PacketHook can observe requests and influence future policy decisions.
// For now it is called after parsing and before responding.
// If it returns non-empty error text, the request is dropped.
type PacketHook func(req Packet, meta RequestMeta) (dropReason string)

type RequestMeta struct {
	ReceivedAt time.Time
	ClientIP   string
	ClientPort int
	RawLen     int
}
