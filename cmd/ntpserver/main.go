package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marcuoli/go-ntpserver/pkg/ntpserver"
)

func main() {
	listen := flag.String("listen", "0.0.0.0:123", "UDP listen address (host:port)")
	stratum := flag.Int("stratum", 2, "NTP stratum (use 16 for unsynchronized)")
	rate := flag.Float64("rate", 0, "Per-IP request rate limit (requests/sec), 0=disabled")
	burst := flag.Int("burst", 5, "Per-IP rate limit burst")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := ntpserver.New(ntpserver.Config{
		ListenAddr:         *listen,
		Stratum:            uint8(*stratum),
		RateLimitPerSecond: *rate,
		RateLimitBurst:     *burst,
		Hook: func(req ntpserver.Packet, meta ntpserver.RequestMeta) (dropReason string) {
			_ = req
			_ = meta
			return ""
		},
	})

	if err := srv.Start(ctx); err != nil {
		log.Printf("failed to start: %v", err)
		os.Exit(1)
	}
	defer func() { _ = srv.Stop() }()

	log.Printf("%s listening on udp://%s", ntpserver.VersionInfo(), *listen)

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Println("stopping...")
			return
		case <-ticker.C:
			m := srv.Metrics()
			log.Printf("requests=%d responses=%d errors=%d unique_clients=%d last_ip=%s", m.TotalRequests, m.TotalResponses, m.TotalErrors, m.UniqueClients, m.LastRequestIP)
		}
	}
}
