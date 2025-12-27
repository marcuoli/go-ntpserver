package ntpserver

import (
	"encoding/binary"
	"time"
)

const (
	PacketSize = 48

	ModeClient = 3
	ModeServer = 4
)

// Timestamp is the 64-bit NTP timestamp (32-bit seconds, 32-bit fraction).
type Timestamp uint64

const ntpEpochOffset = 2208988800

func timeToTimestamp(t time.Time) Timestamp {
	t = t.UTC()
	unixSeconds := t.Unix()
	seconds := uint64(int64(ntpEpochOffset) + unixSeconds)
	fraction := uint64(t.Nanosecond()) * (1 << 32) / 1_000_000_000
	return Timestamp((seconds << 32) | (fraction & 0xffffffff))
}

// Packet is the base NTPv4 (RFC 5905) header.
// Extension fields are intentionally not parsed in this initial version.
type Packet struct {
	LI      uint8
	VN      uint8
	Mode    uint8
	Stratum uint8
	Poll    int8
	Prec    int8

	RootDelay      uint32
	RootDispersion uint32
	RefID          uint32

	Reference Timestamp
	Originate Timestamp
	Receive   Timestamp
	Transmit  Timestamp
}

func ParsePacket(b []byte) (Packet, bool) {
	if len(b) < PacketSize {
		return Packet{}, false
	}
	first := b[0]
	p := Packet{
		LI:      (first >> 6) & 0x3,
		VN:      (first >> 3) & 0x7,
		Mode:    first & 0x7,
		Stratum: b[1],
		Poll:    int8(b[2]),
		Prec:    int8(b[3]),

		RootDelay:      binary.BigEndian.Uint32(b[4:8]),
		RootDispersion: binary.BigEndian.Uint32(b[8:12]),
		RefID:          binary.BigEndian.Uint32(b[12:16]),
		Reference:      Timestamp(binary.BigEndian.Uint64(b[16:24])),
		Originate:      Timestamp(binary.BigEndian.Uint64(b[24:32])),
		Receive:        Timestamp(binary.BigEndian.Uint64(b[32:40])),
		Transmit:       Timestamp(binary.BigEndian.Uint64(b[40:48])),
	}
	return p, true
}

func (p Packet) Marshal() []byte {
	b := make([]byte, PacketSize)
	b[0] = ((p.LI & 0x3) << 6) | ((p.VN & 0x7) << 3) | (p.Mode & 0x7)
	b[1] = p.Stratum
	b[2] = byte(p.Poll)
	b[3] = byte(p.Prec)
	binary.BigEndian.PutUint32(b[4:8], p.RootDelay)
	binary.BigEndian.PutUint32(b[8:12], p.RootDispersion)
	binary.BigEndian.PutUint32(b[12:16], p.RefID)
	binary.BigEndian.PutUint64(b[16:24], uint64(p.Reference))
	binary.BigEndian.PutUint64(b[24:32], uint64(p.Originate))
	binary.BigEndian.PutUint64(b[32:40], uint64(p.Receive))
	binary.BigEndian.PutUint64(b[40:48], uint64(p.Transmit))
	return b
}

func refIDFromASCII4(s string) uint32 {
	var buf [4]byte
	copy(buf[:], []byte(s))
	return binary.BigEndian.Uint32(buf[:])
}

func BuildResponse(req Packet, cfg responseConfig, receivedAt time.Time, transmittedAt time.Time) Packet {
	vn := req.VN
	if vn == 0 {
		vn = 4
	}

	resp := Packet{
		LI:      cfg.LeapIndicator,
		VN:      vn,
		Mode:    ModeServer,
		Stratum: cfg.Stratum,
		Poll:    req.Poll,
		Prec:    cfg.Precision,

		RootDelay:      cfg.RootDelay,
		RootDispersion: cfg.RootDispersion,
		RefID:          cfg.RefID,

		Reference: timeToTimestamp(cfg.ReferenceTime),
		Originate: req.Transmit,
		Receive:   timeToTimestamp(receivedAt),
		Transmit:  timeToTimestamp(transmittedAt),
	}
	return resp
}

type responseConfig struct {
	LeapIndicator uint8
	Stratum       uint8
	Precision     int8
	RootDelay     uint32
	RootDispersion uint32
	RefID         uint32
	ReferenceTime time.Time
}
