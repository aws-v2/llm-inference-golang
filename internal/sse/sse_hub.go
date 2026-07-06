package sse

import (
	"log"
	"sync"
	"time"
)

// Hub is a thread-safe registry that maps sessionID → event channel.
// It also maintains a short-lived buffer for "pending" events that arrive
// before the client has finished connecting.
type Hub struct {
	mu       sync.RWMutex
	sessions map[string]chan []byte
	// pending stores up to 10 recent events per sessionID that haven't registered yet.
	pending map[string][][]byte
	// timestamps tracks when each session was last seen (for cleanup).
	timestamps map[string]time.Time
}

func NewHub() *Hub {
	h := &Hub{
		sessions:   make(map[string]chan []byte),
		pending:    make(map[string][][]byte),
		timestamps: make(map[string]time.Time),
	}
	// Background cleanup for stale pending data.
	go h.cleanupLoop()
	return h
}

// Register creates an event channel for the given sessionID.
// If any events arrived before registration, they are replayed immediately.
func (h *Hub) Register(sessionID string) <-chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()

	log.Printf("[HUB] Registering session %s", sessionID)

	ch, exists := h.sessions[sessionID]
	if !exists {
		ch = make(chan []byte, 64)
		h.sessions[sessionID] = ch
	}

	// Replay pending events if any exist.
	if events, ok := h.pending[sessionID]; ok {
		log.Printf("[HUB] Replaying %d pending events for session %s", len(events), sessionID)
		for _, data := range events {
			select {
			case ch <- data:
			default:
				// if channel is somehow full during replay, skip
			}
		}
		delete(h.pending, sessionID)
	}

	h.timestamps[sessionID] = time.Now()
	return ch
}

// Send delivers an event. If the session isn't registered yet,
// the event is buffered in the 'pending' map for later replay.
func (h *Hub) Send(sessionID string, data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch, ok := h.sessions[sessionID]



	
	if !ok {
		// Not registered yet — buffer it.
		h.pending[sessionID] = append(h.pending[sessionID], data)
		// Cap pending buffer to 10 events per session.
		if len(h.pending[sessionID]) > 10 {
			h.pending[sessionID] = h.pending[sessionID][1:]
		}
		h.timestamps[sessionID] = time.Now()
		log.Printf("[HUB] Buffered event for unregistered session %s (total pending: %d)", sessionID, len(h.pending[sessionID]))
		return
	}

	select {
	case ch <- data:
		h.timestamps[sessionID] = time.Now()
	default:
		log.Printf("[HUB] Dropped event for session %s (buffer full)", sessionID)
	}
}

// Unregister closes and removes the channel for sessionID.
func (h *Hub) Unregister(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if ch, ok := h.sessions[sessionID]; ok {
		close(ch)
		delete(h.sessions, sessionID)
		delete(h.timestamps, sessionID)
		log.Printf("[HUB] Unregistered session %s", sessionID)
	}
}

func (h *Hub) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.Lock()
		now := time.Now()
		for sid, ts := range h.timestamps {
			// If a pending session hasn't connected for 2 minutes, clear it.
			if _, active := h.sessions[sid]; !active && now.Sub(ts) > 2*time.Minute {
				delete(h.pending, sid)
				delete(h.timestamps, sid)
				log.Printf("[HUB] Cleaned up stale pending session %s", sid)
			}
		}
		h.mu.Unlock()
	}
}

