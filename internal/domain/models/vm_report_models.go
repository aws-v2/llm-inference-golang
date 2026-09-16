package model


type VMReportStatus struct{
    RunID string    `json:"run_id"`
    Step string    `json:"step"`
    Status string    `json:"status"`
    Duration float64    `json:"duration_seconds"`
}


type Artifact struct{
 Name   string `json:"name"`
 Url   string `json:"url"`
}


type OrchestratorData struct {
	RunID           string   `json:"run_id"`
	BaseModel       Artifact   `json:"base_model"`
	TrainingFiles   []Artifact `json:"training_data"`
	ValidationFiles []Artifact `json:"validation_data"`
	CodeFiles       []Artifact `json:"code"`
	ControlPlane string `json:"control_plane"`
	ApiKey string `json:"api_key"`
}
