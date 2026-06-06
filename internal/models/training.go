package model

import (
	"time"
)

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
    BucketID string `json:"bucket_id"`

	Progress    float64        `json:"progress"`
	CreatedAt   time.Time      `json:"created_at"`
	Tags        []string       `json:"tags"`
	SessionID   string         `json:"session_id"`
	BucketName   string         `json:"bucket_name"`
}

type PipelineScript struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	RouteTo string `json:"route_to"`
    FileSha256    string `json:"sha256"`
}

// type PipelineNode struct {
// 	ID       string           `json:"id"`
// 	Type     string           `json:"type"`  //: 'ingest' | 'clean' | 'train' | 'evaluate' | 'deploy' | 'custom' | 'gate'
// 	Label    string           `json:"label"` // ?: string; x: number; y: number
// 	Scripts  []PipelineScript `json:"scripts"`
// 	Schedule string           `json:"schedule"`
// 	Cascade  bool             `json:"cascade"`
// 	X int `json:"x"`
// 	Y int `json:"y"`
// 	VMNode string `json:"vmMode"`
// 	Vm string `json:"vm"`
// 	DestBucket string `json:"destBucket"`
// }


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

// // ─── Palette config ───────────────────────────────────────────────────────────
// const paletteTypes = [
//   { type:'ingest',   label:'ingest',  icon:'⬇', desc:'Pull data from sources' },
//   { type:'clean',    label:'clean',   icon:'✦', desc:'Normalize & deduplicate' },
//   { type:'train',    label:'train',   icon:'◈', desc:'Distributed model training' },
//   { type:'evaluate', label:'eval',    icon:'◉', desc:'Score & compare models' },
//   { type:'deploy',   label:'deploy',  icon:'↗', desc:'Expose inference endpoint' },
//   { type:'custom',   label:'script',  icon:'⟨⟩', desc:'Custom script node' },
//   { type:'gate',     label:'gate',    icon:'⑂', desc:'Condition gate' },
// ]
