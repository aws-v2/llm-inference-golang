package model

import "time"

type TrainingJob struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Instance    string    `json:"instance"`
	InputPath   string    `json:"input_path"`
	OutputPath  string    `json:"output_path"`
	Progress    int       `json:"progress"`
	CreatedAt   time.Time `json:"created_at"`
}