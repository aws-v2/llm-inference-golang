package ws

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Manager routes VM events to frontend subscribers per run ID.
type Manager struct {
	mu        sync.RWMutex
	vmConns   map[string]*websocket.Conn
	frontends map[string]map[*websocket.Conn]struct{}
}

// NewManager creates an initialized Manager.
func NewManager() *Manager {
	return &Manager{
		vmConns:   make(map[string]*websocket.Conn),
		frontends: make(map[string]map[*websocket.Conn]struct{}),
	}
}

// RegisterVM associates a VM websocket with a runID. If a previous VM
// connection exists it will be closed and replaced.
func (m *Manager) RegisterVM(runID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if prev, ok := m.vmConns[runID]; ok {
		_ = prev.Close()
	}
	m.vmConns[runID] = conn
	log.Printf("[WS Manager] Registered VM for run %s", runID)
}

// UnregisterVM removes and closes the VM connection for runID.
func (m *Manager) UnregisterVM(runID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.vmConns[runID]; ok {
		_ = c.Close()
		delete(m.vmConns, runID)
	}
	log.Printf("[WS Manager] Unregistered VM for run %s", runID)
}

// RegisterFrontend registers a frontend websocket to receive events for runID.
func (m *Manager) RegisterFrontend(runID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	set, ok := m.frontends[runID]
	if !ok {
		set = make(map[*websocket.Conn]struct{})
		m.frontends[runID] = set
	}
	set[conn] = struct{}{}
	log.Printf("[WS Manager] Registered frontend for run %s (total: %d)", runID, len(set))
}

// UnregisterFrontend removes a frontend websocket from the run's subscriber set.
func (m *Manager) UnregisterFrontend(runID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if set, ok := m.frontends[runID]; ok {
		if _, exists := set[conn]; exists {
			delete(set, conn)
			_ = conn.Close()
		}
		if len(set) == 0 {
			delete(m.frontends, runID)
		}
	}
	log.Printf("[WS Manager] Unregistered frontend for run %s", runID)
}

// BroadcastToRun sends message to all frontend subscribers of runID.
// If a write fails the connection is removed.
func (m *Manager) BroadcastToRun(runID string, message []byte) {
	m.mu.RLock()
	set, ok := m.frontends[runID]
	m.mu.RUnlock()

	if !ok || len(set) == 0 {
		// no subscribers; nothing to do
		fmt.Println("no subscribers")
		return
	}

	// Iterate over a snapshot to avoid holding the RLock during network writes.
	conns := make([]*websocket.Conn, 0, len(set))
	m.mu.RLock()
	for c := range set {
		conns = append(conns, c)
	}
	m.mu.RUnlock()

	for _, c := range conns {
		// protect each write with a deadline
		_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("[WS Manager] write error for run %s: %v; removing connection", runID, err)
			m.UnregisterFrontend(runID, c)
		}
	}
}
