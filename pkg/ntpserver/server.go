package ntpserver

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

var ErrAlreadyRunning = errors.New("ntpserver: already running")

type Config struct {
	ListenAddr string

	// Clock defaults to a system UTC clock.
	Clock Clock

	// Stratum defaults to 2. If you want to indicate "unsynchronized", use 16.
	Stratum uint8

	// RefID defaults to "LOCL".
	RefID uint32

	// LeapIndicator defaults to 0 (no warning).
	LeapIndicator uint8

	// Precision defaults to -20 (~1 microsecond).
	Precision int8

	// RootDelay and RootDispersion are optional fixed-point values.
	RootDelay uint32
	RootDispersion uint32

	// RateLimitPerSecond enables a basic per-IP token bucket limiter.
	// Set to 0 to disable.
	RateLimitPerSecond float64
	RateLimitBurst     int

	// EventBuffer is the buffer size per subscriber.
	EventBuffer int
	// HistorySize is how many recent events are kept.
	HistorySize int

	// Hook is called after parsing and basic checks, before responding.
	// If it returns a non-empty string, the request is dropped.
	Hook PacketHook

	// Logger for debug/info messages. If nil, no logging is performed.
	Logger *log.Logger

	// Debug enables verbose debug logging.
	Debug bool
}

func (c Config) normalize() Config {
	out := c
	if out.ListenAddr == "" {
		out.ListenAddr = "0.0.0.0:123"
	}
	if out.Clock == nil {
		out.Clock = systemClock{}
	}
	if out.Stratum == 0 {
		out.Stratum = 2
	}
	if out.RefID == 0 {
		out.RefID = refIDFromASCII4("LOCL")
	}
	if out.Precision == 0 {
		out.Precision = -20
	}
	if out.EventBuffer <= 0 {
		out.EventBuffer = 128
	}
	if out.HistorySize <= 0 {
		out.HistorySize = 500
	}
	if out.RateLimitBurst <= 0 {
		out.RateLimitBurst = 5
	}
	return out
}

type Server struct {
	cfg Config

	mu      sync.RWMutex
	conn    *net.UDPConn
	running bool

	hub     *eventHub
	metrics *metrics
	limiter *limiter

	wg sync.WaitGroup
	stopOnce sync.Once
	stopCh chan struct{}
}

func New(cfg Config) *Server {
	cfg = cfg.normalize()
	return &Server{
		cfg:     cfg,
		hub:     newEventHub(cfg.HistorySize),
		metrics: newMetrics(),
		limiter: newLimiter(cfg.RateLimitPerSecond, cfg.RateLimitBurst),
		stopCh:  make(chan struct{}),
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrAlreadyRunning
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.stopOnce = sync.Once{}
	s.mu.Unlock()

	udpAddr, err := net.ResolveUDPAddr("udp", s.cfg.ListenAddr)
	if err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return err
	}

	s.mu.Lock()
	s.conn = conn
	s.metrics.reset(time.Now().UTC())
	s.mu.Unlock()

	if s.cfg.Logger != nil {
		s.cfg.Logger.Printf("[INFO] NTP server started on %s (stratum %d)", s.cfg.ListenAddr, s.cfg.Stratum)
	}

	s.wg.Add(1)
	go s.serveLoop(ctx)
	return nil
}

// Addr returns the current bound local address if running, otherwise the configured ListenAddr.
func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.conn != nil {
		return s.conn.LocalAddr().String()
	}
	return s.cfg.ListenAddr
}

func (s *Server) Stop() error {
	var conn *net.UDPConn
	s.stopOnce.Do(func() {
		s.mu.Lock()
		conn = s.conn
		s.conn = nil
		s.running = false
		close(s.stopCh)
		s.mu.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
	})

	s.wg.Wait()
	return nil
}

func (s *Server) Subscribe() (<-chan RequestEvent, func()) {
	return s.hub.subscribe(s.cfg.EventBuffer)
}

func (s *Server) History() []RequestEvent {
	return s.hub.snapshotHistory()
}

func (s *Server) Metrics() MetricsSnapshot {
	return s.metrics.snapshot()
}

func (s *Server) serveLoop(ctx context.Context) {
	defer s.wg.Done()

	buf := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			_ = s.Stop()
			return
		case <-s.stopCh:
			return
		default:
		}

		s.mu.RLock()
		conn := s.conn
		s.mu.RUnlock()
		if conn == nil {
			return
		}

		_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return
		}

		receivedAt := s.cfg.Clock.Now()
		start := time.Now()

		clientIP := ""
		clientPort := 0
		clientAddr := ""
		if raddr != nil {
			clientIP = raddr.IP.String()
			clientPort = raddr.Port
			clientAddr = raddr.String()
		}

		s.metrics.incRequest(clientIP, receivedAt)

		if s.cfg.Debug && s.cfg.Logger != nil {
			s.cfg.Logger.Printf("[DEBUG] NTP request from %s:%d", clientIP, clientPort)
		} else if s.cfg.Logger != nil {
			s.cfg.Logger.Printf("[INFO] NTP request from %s", clientIP)
		}

		ev := RequestEvent{
			At:         receivedAt,
			ClientAddr: clientAddr,
			ClientIP:   clientIP,
			ClientPort: clientPort,
			Responded:  false,
		}

		if !s.limiter.allow(clientIP, time.Now()) {
			ev.PacketValid = true
			ev.Error = "rate_limited"
			ev.ProcessingUSec = time.Since(start).Microseconds()
			s.metrics.incError()
			s.hub.publish(ev)
			continue
		}

		req, ok := ParsePacket(buf[:n])
		ev.PacketValid = ok
		ev.Version = req.VN
		ev.Mode = req.Mode

		if !ok || req.Mode != ModeClient {
			ev.Error = "invalid_request"
			ev.ProcessingUSec = time.Since(start).Microseconds()
			s.metrics.incError()
			s.hub.publish(ev)
			continue
		}

		if s.cfg.Hook != nil {
			dropReason := s.cfg.Hook(req, RequestMeta{ReceivedAt: receivedAt, ClientIP: clientIP, ClientPort: clientPort, RawLen: n})
			if dropReason != "" {
				ev.Error = dropReason
				ev.ProcessingUSec = time.Since(start).Microseconds()
				s.metrics.incError()
				s.hub.publish(ev)
				continue
			}
		}

		now := s.cfg.Clock.Now()
		resp := BuildResponse(req, responseConfig{
			LeapIndicator:  s.cfg.LeapIndicator,
			Stratum:        s.cfg.Stratum,
			Precision:      s.cfg.Precision,
			RootDelay:      s.cfg.RootDelay,
			RootDispersion: s.cfg.RootDispersion,
			RefID:          s.cfg.RefID,
			ReferenceTime:  now,
		}, receivedAt, now)

		out := resp.Marshal()
		_, werr := conn.WriteToUDP(out, raddr)
		if werr != nil {
			ev.Error = werr.Error()
			ev.ProcessingUSec = time.Since(start).Microseconds()
			s.metrics.incError()
			s.hub.publish(ev)
			continue
		}

		s.metrics.incResponse()
		ev.Responded = true
		ev.ProcessingUSec = time.Since(start).Microseconds()
		s.hub.publish(ev)
	}
}
