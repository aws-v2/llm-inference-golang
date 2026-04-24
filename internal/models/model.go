package model

import "time"

type Status string

const (
	StatusPendingUpload Status = "PENDING_UPLOAD"
	StatusReady         Status = "READY"
)

type Model struct {
	ID        string
	Name      string
	FilePath  string
	OwnerID   string

	Status    Status

	CreatedAt time.Time

	Temperature float64
	MaxTokens   int
}