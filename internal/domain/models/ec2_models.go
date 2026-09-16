package model

import (
	"encoding/json"
	"time"
)

type Manifest struct {
	ID         string            `json:"id"           gorm:"primaryKey"`
	ProjID     string            `json:"game_id"      gorm:"index"`
	Name       string            `json:"name"`
	CreatedAt  time.Time         `json:"created_at"`
	Parameters map[string]string `json:"parameters"`
}

 
type VMSpecs struct {
	CPU     int `json:"cpu" binding:"required"`
	RAM     int `json:"ram" binding:"required"` // MB
	Storage int `json:"storage,omitempty"`      // GB, optional override of image default
}

type EC2ProvisionRequest struct {
	UserID     string          `json:"userID"`
	Profile    string          `json:"profile" binding:"required"`
	Name       string          `json:"name" binding:"required"`
	ResourceID string          `json:"resource_id"`
	Specs      VMSpecs         `json:"specs" binding:"required"`
	SessionID  string          `json:"session_id"`
	Assets     []Asset         `json:"assets,omitempty"`
	Config     json.RawMessage `json:"config,omitempty"` // profile-specific config, opaque to the core service
}


type SageMakerVM struct {
	VMIP string `json:"vm_ip"`
	VMID string `json:"vm_id"`
		Code int `json:"code"`


}
