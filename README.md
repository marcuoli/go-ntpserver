# go-ntpserver

A small, pure-Go NTPv4 (RFC 5905) UDP server library with a thin CLI.

This project intentionally starts as a simple SNTP-style responder (stateless, UDP, client-mode requests) while providing hooks for future improvements (rate limiting, policy, Kiss-o'-Death, extension fields, NTS).

## Install

```bash
go get github.com/marcuoli/go-ntpserver
```

## Quick start (library)

```go
srv := ntpserver.New(ntpserver.Config{ListenAddr: "0.0.0.0:123"})
if err := srv.Start(context.Background()); err != nil {
    panic(err)
}
defer srv.Stop()
```

## CLI

```bash
go run ./cmd/ntpserver -listen 0.0.0.0:123
```

## Protocol

- Core protocol: RFC 5905 (NTPv4)
- This server implements a minimal, working subset (SNTP-style responder).

Note: There is no RFC for a "multithreaded" NTP server; concurrency is an implementation detail.

## Versioning

This library uses semantic versioning.

- Default bump for routine changes (tests, bugfixes, small improvements): increment **patch** ($0.3.x \rightarrow 0.3.(x+1)$)
- Increment **minor** only when adding backwards-compatible features that materially change behavior or public API
- Increment **major** only for breaking changes

The source of truth lives in `pkg/ntpserver/version.go`.
