package ntpserver

import "sync"

type eventHub struct {
	mu          sync.RWMutex
	subscribers map[chan RequestEvent]struct{}
	history     []RequestEvent
	maxHistory  int
}

func newEventHub(maxHistory int) *eventHub {
	if maxHistory <= 0 {
		maxHistory = 500
	}
	return &eventHub{
		subscribers: make(map[chan RequestEvent]struct{}),
		maxHistory:  maxHistory,
	}
}

func (h *eventHub) publish(ev RequestEvent) {
	h.mu.Lock()
	if h.maxHistory > 0 {
		h.history = append(h.history, ev)
		if len(h.history) > h.maxHistory {
			copy(h.history, h.history[len(h.history)-h.maxHistory:])
			h.history = h.history[:h.maxHistory]
		}
	}
	for ch := range h.subscribers {
		select {
		case ch <- ev:
		default:
			// Drop if subscriber is slow.
		}
	}
	h.mu.Unlock()
}

func (h *eventHub) subscribe(buffer int) (<-chan RequestEvent, func()) {
	if buffer <= 0 {
		buffer = 128
	}
	ch := make(chan RequestEvent, buffer)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if _, ok := h.subscribers[ch]; ok {
			delete(h.subscribers, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
	return ch, cancel
}

func (h *eventHub) snapshotHistory() []RequestEvent {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]RequestEvent, len(h.history))
	copy(out, h.history)
	return out
}
