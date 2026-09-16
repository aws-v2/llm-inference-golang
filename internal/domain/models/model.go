package model

import "time"

type Status string

const (
	StatusPendingUpload Status = "PENDING_UPLOAD"
	StatusReady         Status = "READY"
)


type RegisterRequest struct {
	Name string `json:"name"`
	OwnerID string `json:"owner_id"`
	IsPublic bool `json:"is_public"`
	Sha256 string `json:"sha256"`

}

type Model struct {
	ID        string `json:"id"`
	OwnerID   string `json:"owner_id"`
	Name      string `json:"name"`
	IsPublic bool `json:"is_public"`
	Sha256 string `json:"sha256"`
	CreatedAt time.Time `json:"created_at"`
}