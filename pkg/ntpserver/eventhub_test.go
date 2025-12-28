package ntpserver

import (
	"testing"
	"time"
)

func TestEventHub_HistoryTruncates(t *testing.T) {
	h := newEventHub(2)
	h.publish(RequestEvent{At: time.Unix(1, 0), ClientIP: "1.1.1.1"})
	h.publish(RequestEvent{At: time.Unix(2, 0), ClientIP: "2.2.2.2"})
	h.publish(RequestEvent{At: time.Unix(3, 0), ClientIP: "3.3.3.3"})

	hist := h.snapshotHistory()
	if len(hist) != 2 {
		t.Fatalf("history len: got=%d want=%d", len(hist), 2)
	}
	if hist[0].ClientIP != "2.2.2.2" || hist[1].ClientIP != "3.3.3.3" {
		t.Fatalf("history order: got=%v", []string{hist[0].ClientIP, hist[1].ClientIP})
	}
}

func TestEventHub_SubscribeUnsubscribeCloses(t *testing.T) {
	h := newEventHub(10)
	ch, cancel := h.subscribe(1)
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatalf("expected channel closed")
		}
	default:
		// channel may close slightly later, but cancel closes under lock.
		_, ok := <-ch
		if ok {
			t.Fatalf("expected channel closed")
		}
	}
}
