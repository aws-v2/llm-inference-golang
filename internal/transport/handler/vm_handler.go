package handler

import (
	"fmt"
	"log"
	"net/http"

	"llm-inference-service/internal/sse"

	"github.com/go-chi/chi/v5"
)

// VMHandler streams EC2 instance lifecycle events to the client via SSE.
type VMHandler struct {
	hub *sse.Hub
}

func NewVMHandler(hub *sse.Hub) *VMHandler {
	return &VMHandler{hub: hub}
}

// StreamEvents handles GET /api/v1/llm/vm/events/{sessionID}
//
// The frontend opens this endpoint before (or immediately after) triggering
// node execution. It keeps the connection alive and receives one SSE event
// per EC2 lifecycle notification until the session channel is closed or the
// client disconnects.
func (h *VMHandler) StreamEvents(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	if sessionID == "" {
		http.Error(w, "missing sessionID", http.StatusBadRequest)
		return
	}

	// Verify the client can flush — required for SSE.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	// Send initial connection acknowledgement.
	fmt.Fprintf(w, "event: connected\ndata: {\"session_id\":%q}\n\n", sessionID)
	flusher.Flush()

	log.Printf("[SSE] client connected for session %s", sessionID)

	ch := h.hub.Register(sessionID)
	defer h.hub.Unregister(sessionID)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			log.Printf("[SSE] client disconnected for session %s", sessionID)
			return

		case msg, open := <-ch:
			if !open {
				// Hub closed the channel (e.g. session ended).
				fmt.Fprintf(w, "event: closed\ndata: {}\n\n")
				flusher.Flush()
				return
			}
			// Each message is already a JSON blob; wrap it in an SSE frame.
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}
