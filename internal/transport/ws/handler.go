package ws

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

type Handler struct {
	mgr *Manager
}

func NewHandler(mgr *Manager) *Handler {
	return &Handler{mgr: mgr}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type EventMessage struct {
	RunID    string            `json:"run_id"`
	Step     string            `json:"step"`
	Event    string            `json:"event"`
	Message  string            `json:"message"`
	Metadata map[string]string `json:"metadata"`
}

// VMEvents handles websocket connections from VM runtimes connecting to /events/{run_id}
func (h *Handler) VMEvents(w http.ResponseWriter, r *http.Request) {
	log.Printf("[WS VM] upgrade succes")

	runID := chi.URLParam(r, "run_id")
	if runID == "" {
		http.Error(w, "missing run_id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS VM] upgrade failed: %v", err)
		return
	}

	h.mgr.RegisterVM(runID, conn)

	// Read loop: forward incoming messages to frontend subscribers.
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[WS VM] read closed for run %s: %v", runID, err)
			break
		}

 

		// Broadcast exactly what we received.
		h.mgr.BroadcastToRun(runID, msg)

	}

	h.mgr.UnregisterVM(runID)
}

// FrontendWS handles websocket connections from browser clients: /ws/runs/{run_id}
func (h *Handler) FrontendWS(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "run_id")
	if runID == "" {
		http.Error(w, "missing run_id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS FE] upgrade failed: %v", err)
		return
	}

	// Register subscriber
	h.mgr.RegisterFrontend(runID, conn)

	// Block and detect client disconnects. We simply read messages and ignore them,
	// using read errors as a signal to unregister.
	conn.SetReadDeadline(time.Time{})
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	h.mgr.UnregisterFrontend(runID, conn)
}
