package model

import "time"



type AddPipelineJobRequest struct {
	RunID string
	Step  string
	Status string
		VMID   string `json:"vm_id,omitempty"`
	VMIP   string `json:"vm_ip,omitempty"`
	ProjectID string `json:"project_id"`
}

type PipelineJob struct {
	ID     string `json:"id"`
	RunID  string `json:"run_id"`
	Step   string `json:"step"`
	Status string `json:"status"`
	VMID   string `json:"vm_id,omitempty"`
	ProjectID   string `json:"project_id"`
	VMIP   string `json:"vm_ip,omitempty"`
}

type QueuedRun struct {
	ID        string   `json:"id"`
	ProjectID string   `json:"project_id"`
	Status    string   `json:"status"`
	Pipeline  []string `json:"pipeline"`
}

type TrainingJobFile struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	Size      int64  `json:"size"`
	Type      string `json:"type"`
	Path      string `json:"path"`
	Sha256 string `json:"sha256,omitempty"`

}


type AddTrainingJobRequest struct {
	ProjectID string
	Status    string
}


type TrainingQueue struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
	WSUrl string `json:"ws_url"`
}


type TrainingJobDto struct {
	ID          string         `json:"id"`
	OwnerID     string         `json:"owner_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Nodes       []PipelineNode `json:"nodes"`
	Edges       []PipelineEdge `json:"edges"`
	Pipeline    []string       `json:"pipeline"`
	Status      string         `json:"status"`
	Progress    float64        `json:"progress"`
	CreatedAt   time.Time      `json:"created_at"`
	Tags        []string       `json:"tags"`
	SessionID   string         `json:"session_id"`
}
type TrainingJob struct {
	ID          string         `json:"id"`
	OwnerID     string         `json:"owner_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Nodes       []PipelineNode `json:"nodes"`
	Edges       []PipelineEdge `json:"edges"`
	Pipeline    []string       `json:"pipeline"`
	Status      string         `json:"status"`
	BucketID    string         `json:"bucket_id"`

	Progress   float64   `json:"progress"`
	CreatedAt  time.Time `json:"created_at"`
	Tags       []string  `json:"tags"`
	SessionID  string    `json:"session_id"`
	BucketName string    `json:"bucket_name"`
}
type FileUploadResult struct {
	FileName  string `json:"file_name"`
	UploadURL string `json:"upload_url"`
}
type LLMTrainingJobFile struct {
	ID        string    `json:"id,omitempty"`
	ProjectID string    `json:"project_id,omitempty"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Size      int64     `json:"size"`
	Type      string    `json:"type"`
	Path      string    `json:"path,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Sha256 string `json:"sha256,omitempty"`
}



type LLMTrainingJob struct {
	ID                 string               `json:"id"`
	OwnerID            string               `json:"owner_id,omitempty"`
	Name               string               `json:"name"`
	Description        string               `json:"description"`
	Type               string               `json:"type"`
	BaseModel          string               `json:"base_model"`
	TrainingDataSource string               `json:"training_data_source"`
	TrainingDataURL    string               `json:"training_data_url"`
	TrainingDataBucket string               `json:"training_data_bucket"`
	UseValidationData  bool                 `json:"use_validation_data"`
	TrainingMode       string               `json:"training_mode"`
	EntryPoint         string               `json:"entry_point"`
	Compute            string               `json:"compute"`
	Schedule           string               `json:"schedule"`
	ScheduleIntervalSeconds *int64          `json:"schedule_interval_seconds,omitempty"`
	NextRunAt          *time.Time           `json:"next_run_at,omitempty"`
	LastRunAt          *time.Time           `json:"last_run_at,omitempty"`
	ScheduleEnabled    bool                 `json:"schedule_enabled,omitempty"`
	Pipeline           []string             `json:"pipeline"`
	Tags               []string             `json:"tags"`
	Status             string               `json:"status"`
	Nodes              []PipelineNode       `json:"nodes,omitempty"`
	Edges              []PipelineEdge       `json:"edges,omitempty"`
	Progress           float64              `json:"progress,omitempty"`
	SessionID          string               `json:"session_id,omitempty"`
	BucketName         string               `json:"bucket_name,omitempty"`
	BucketID           string               `json:"bucket_id,omitempty"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
	Files              []LLMTrainingJobFile `json:"files,omitempty"`
	Runs               []interface{}        `json:"runs,omitempty"`
	PresignedURLs      []PresignedURLInfo   `json:"presigned_urls,omitempty"`
	Bucket      string   `json:"bucket"`
	ApiKey      string   `json:"api_key"`
	CallBack string `json:"callback"`

}

type CreateLLMJobRequest struct {
	ID                 string               `json:"id"`
	Name               string               `json:"name"`
	Description        string               `json:"description"`
	Type               string               `json:"type"`
	BaseModel          string               `json:"base_model"`
	TrainingDataSource string               `json:"training_data_source"`
	TrainingDataURL    string               `json:"training_data_url"`
	TrainingDataBucket string               `json:"training_data_bucket"`
	UseValidationData  bool                 `json:"use_validation_data"`
	TrainingMode       string               `json:"training_mode"`
	EntryPoint         string               `json:"entry_point"`
	Compute            string               `json:"compute"`
	Schedule           string               `json:"schedule"`
	Pipeline           []string             `json:"pipeline"`
	Tags               []string             `json:"tags"`
	Files              []LLMTrainingJobFile `json:"files"`
	CallBack string `json:"callback"`
}

type CreateLLMJobResponse struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Description   string             `json:"description"`
	Type          string             `json:"type"`
	BaseModel     string             `json:"base_model"`
	Status        string             `json:"status"`
	CreatedAt     time.Time          `json:"created_at"`
	PresignedURLs []PresignedURLInfo `json:"presigned_urls"`
}


type PipelineScript struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	RouteTo string `json:"route_to"`
    FileSha256    string `json:"sha256"`
}
 

 type PipelineNode struct {
	ID         string           `json:"id"`
	Type       string           `json:"type"`
	Label      string           `json:"label"`
	Scripts    []PipelineScript `json:"scripts"`
	Schedule   string           `json:"schedule"`
	Cascade    bool             `json:"cascade"`
	X          int              `json:"x"`
	Y          int              `json:"y"`
	VMNode     string           `json:"vmMode"`
	Vm         string           `json:"vm"`
	DestBucket string           `json:"destBucket"`
	SrcBucket string           `json:"srcBucket"`
	Status     string           `json:"status"`               // pending|running|completed|failed
	StartedAt  *time.Time       `json:"started_at,omitempty"`
	FinishedAt *time.Time       `json:"finished_at,omitempty"`
	Error      string           `json:"error,omitempty"`
}


type PipelineEdge struct {
	ID         string `json:"id"`
	FromNodeId string `json:"fromNodeId"`
	ToNodeId   string `json:"toNodeId"`
}

var (
	InestNode  = "ingest"
	CleanNode  = "clean"
	TrainNode  = "train"
	EvalNode   = "evaluate"
	DeployNode = "deploy"
	CustomNode = "custom"
	GateNode   = "gate"
)
 
 















 

// S3 types






type NodeExecutionResult struct {
	NodeID     string     `json:"node_id"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	SessionID  string     `json:"session_id"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Error      string     `json:"error,omitempty"`
}