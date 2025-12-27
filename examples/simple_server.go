//go:build ignore

package main

import (
	"context"
	"log"
	"time"

	"github.com/marcuoli/go-ntpserver/pkg/ntpserver"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := ntpserver.New(ntpserver.Config{ListenAddr: "127.0.0.1:1123"})
	if err := srv.Start(ctx); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = srv.Stop() }()

	log.Println("running for 30s on udp://127.0.0.1:1123")
	time.Sleep(30 * time.Second)
}
