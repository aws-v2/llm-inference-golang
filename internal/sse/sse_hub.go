package sse

import (
	"sync"
)

// Hub is a thread-safe registry that maps sessionID → event channel.
// Each connected SSE client owns one channel in the registry.
type Hub struct {
	mu       sync.RWMutex
	sessions map[string]chan []byte
}

func NewHub() *Hub {
	return &Hub{
		sessions: make(map[string]chan []byte),
	}
}

// Register creates a buffered channel for the given sessionID and returns it.
// If a channel already exists for that session it is returned as-is.
func (h *Hub) Register(sessionID string) <-chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()

	if ch, ok := h.sessions[sessionID]; ok {
		return ch
	}
	ch := make(chan []byte, 64)
	h.sessions[sessionID] = ch
	return ch
}

// Send delivers a raw event payload to the session's channel.
// It is non-blocking: if the buffer is full the message is dropped.
func (h *Hub) Send(sessionID string, data []byte) {
	h.mu.RLock()
	ch, ok := h.sessions[sessionID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case ch <- data:
	default:
		// buffer full — drop
	}
}

// Unregister closes and removes the channel for sessionID.
func (h *Hub) Unregister(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if ch, ok := h.sessions[sessionID]; ok {
		close(ch)
		delete(h.sessions, sessionID)
	}
}
