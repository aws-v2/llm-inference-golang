package models

type InferenceRequest struct {
	ModelID string                 `json:"model_id"`
	Prompt  string                 `json:"prompt"`
	Params  map[string]interface{} `json:"params"`
}