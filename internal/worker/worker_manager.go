package worker

import "log"

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) StartWorker(modelID, path string) {
	// Later: docker run python-layer with env vars
	log.Printf("Starting worker for model %s at %s\n", modelID, path)
}