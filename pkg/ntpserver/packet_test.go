package ntpserver

import (
	"testing"
	"time"
)

func TestTimeToTimestamp_UnixEpoch(t *testing.T) {
	ts := timeToTimestamp(time.Unix(0, 0).UTC())
	expected := Timestamp(uint64(ntpEpochOffset) << 32)
	if ts != expected {
		t.Fatalf("unexpected timestamp: got=%d want=%d", ts, expected)
	}
}

func TestPacket_MarshalParse_RoundTrip(t *testing.T) {
	p := Packet{
		LI:      0,
		VN:      4,
		Mode:    ModeClient,
		Stratum: 0,
		Poll:    6,
		Prec:    -20,

		RootDelay:      0x01020304,
		RootDispersion: 0x05060708,
		RefID:          refIDFromASCII4("TEST"),
		Reference:      Timestamp(0x1111111122222222),
		Originate:      Timestamp(0x3333333344444444),
		Receive:        Timestamp(0x5555555566666666),
		Transmit:       Timestamp(0x7777777788888888),
	}

	b := p.Marshal()
	if len(b) != PacketSize {
		t.Fatalf("unexpected marshal size: got=%d want=%d", len(b), PacketSize)
	}

	p2, ok := ParsePacket(b)
	if !ok {
		t.Fatalf("expected parse ok")
	}
	if p2 != p {
		t.Fatalf("packet mismatch after roundtrip:\n got=%+v\nwant=%+v", p2, p)
	}
}

func TestBuildResponse_BasicFields(t *testing.T) {
	reqTx := timeToTimestamp(time.Date(2023, 1, 2, 3, 4, 5, 6_000_000, time.UTC))
	req := Packet{VN: 4, Mode: ModeClient, Poll: 6, Transmit: reqTx}

	receivedAt := time.Date(2023, 1, 2, 3, 4, 5, 7_000_000, time.UTC)
	now := time.Date(2023, 1, 2, 3, 4, 5, 8_000_000, time.UTC)

	resp := BuildResponse(req, responseConfig{
		LeapIndicator:  0,
		Stratum:        2,
		Precision:      -20,
		RootDelay:      0,
		RootDispersion: 0,
		RefID:          refIDFromASCII4("LOCL"),
		ReferenceTime:  now,
	}, receivedAt, now)

	if resp.Mode != ModeServer {
		t.Fatalf("unexpected mode: got=%d want=%d", resp.Mode, ModeServer)
	}
	if resp.VN != 4 {
		t.Fatalf("unexpected version: got=%d want=%d", resp.VN, 4)
	}
	if resp.Stratum != 2 {
		t.Fatalf("unexpected stratum: got=%d want=%d", resp.Stratum, 2)
	}
	if resp.Originate != reqTx {
		t.Fatalf("unexpected originate: got=%d want=%d", resp.Originate, reqTx)
	}
	if resp.Receive != timeToTimestamp(receivedAt) {
		t.Fatalf("unexpected receive timestamp")
	}
	if resp.Transmit != timeToTimestamp(now) {
		t.Fatalf("unexpected transmit timestamp")
	}
}
