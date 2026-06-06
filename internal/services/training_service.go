package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	model "llm-inference-service/internal/models"
	"llm-inference-service/internal/nats"
	"llm-inference-service/internal/repository"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TrainingService struct {
	repo *repository.PostgresRepository
	nc   *nats.Client
}
type AssetType string

const (
	AssetTypeGame     AssetType = "game"
	AssetTypeTemplate AssetType = "template"
	AssetTypeAgent    AssetType = "agent"
	AssetTypeScript   AssetType = "script"
)

type createPresignedURLRequest struct {
	UserID     string    `json:"user_id"`
	GameID     string    `json:"game_id,omitempty"`
	AssetID    string    `json:"asset_id"`
	AssetName  string    `json:"asset_name"`
	AssetType  AssetType `json:"asset_type"` // "game" | "template"
	Sha256     string    `json:"sha256"`
	Key        string    `json:"key"`
	BucketName string    `json:"bucket_name"` // this is the nameof the job/game/render job this asset belongs to like kalshi or ruto tracker

}
type NodeExecutionResult struct {
	NodeID     string     `json:"node_id"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	SessionID  string     `json:"session_id"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Error      string     `json:"error,omitempty"`
}
type ArchiveByPrefixInputMessage struct {
	BucketID    string `json:"bucket_id"`
	Prefix      string `json:"prefix" binding:"required"`
	ArchiveName string `json:"archive_name" binding:"required"`
	Format      string `json:"format"` // zip or tar
	UserID      string `json:"user_id"`
}

type createPresignDownloadURLRequest struct {
	UserID        string `json:"user_id"`
	CorrelationID string `json:"correlation_id"`
	FileSha256    string `json:"sha256"`
	AssetID       string `json:"asset_id"`
}
type createPresignDownloadURLResponse struct {
	URL      string `json:"url"`
	BucketID string `json:"bucket_id"`

	FileCount int `json:"file_count"` // zip or tar
}

func NewTrainingService(repo *repository.PostgresRepository, nc *nats.Client) *TrainingService {
	return &TrainingService{repo: repo,
		nc: nc,
	}
}

type CreateDefaultBucketRequest struct {
	CorrelationID string `json:"correlation_id"`
	UserID        string `json:"user_id"`
	SessionID     string `json:"session_id"`
	BucketName    string `json:"bucket_name"`
}

type CreateDefaultBucketResponse struct {
	BucketName string `json:"bucket_name"`
	Created    bool   `json:"created"`
	Error      string `json:"error,omitempty"`
	BucketID   string `json:"bucket_id"`
}

func (s *TrainingService) CreateJob(ownerID string, req model.TrainingJobDto) (model.TrainingJob, error) {
	nodes := req.Nodes
	edges := req.Edges

	if len(nodes) == 0 {
		nodes = buildDefaultNodes(req.Pipeline)
	}
	if len(edges) == 0 {
		edges = buildDefaultEdges(nodes)
	}

	// sanitize project name → bucket name
	// "Test One" → "test_one_sg_bucket"
	bucketName := sanitizeBucketName(req.Name)

	// create default bucket via NATS
	bucketReq := CreateDefaultBucketRequest{
		CorrelationID: uuid.New().String(),
		UserID:        ownerID,
		SessionID:     req.SessionID,
		BucketName:    fmt.Sprintf("%s-sg-project", bucketName),
	}

	respBytes, err := s.nc.Publisher.Request("s3.task.create_default_bucket", bucketReq)
	if err != nil {
		return model.TrainingJob{}, fmt.Errorf("bucket creation failed: %w", err)
	}

	var bucketResp CreateDefaultBucketResponse
	if err := json.Unmarshal(respBytes, &bucketResp); err != nil {
		return model.TrainingJob{}, fmt.Errorf("unmarshal bucket resp: %w", err)
	}
	if bucketResp.Error != "" {
		return model.TrainingJob{}, fmt.Errorf("bucket service error: %s", bucketResp.Error)
	}

	// stamp each node with its src/dest folders under the single bucket
	for i := range nodes {
		nodes[i].SrcBucket = fmt.Sprintf("%s/%s_src", bucketName, nodes[i].Type)
		nodes[i].DestBucket = fmt.Sprintf("%s/%s_dest", bucketName, nodes[i].Type)
	}

	// wire cascade: next node's SrcBucket = prev node's DestBucket
	for i := 1; i < len(nodes); i++ {
		nodes[i].SrcBucket = nodes[i-1].DestBucket
	}

	job := model.TrainingJob{
		ID:          uuid.New().String(),
		OwnerID:     ownerID,
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
		SessionID:   req.SessionID,
		Status:      "Initializing",
		Progress:    0,
		Nodes:       nodes,
		Edges:       edges,
		BucketName:  bucketName,
		BucketID:    bucketResp.BucketID,
		CreatedAt:   time.Now(),
	}

	err = s.repo.Create(job)
	return job, err
}

func sanitizeBucketName(name string) string {
	// lowercase, replace spaces/special chars with underscores, append suffix
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	// strip anything not alphanumeric or underscore
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	// return fmt.Sprintf("%s-scripst",req.AssetName) ,req.Key, nil

	log.Printf("This is the bucket name %s", b.String())
	return b.String()
}

var defaultPipelineStages = []string{"ingest", "clean", "train", "evaluate", "deploy"}

func buildDefaultNodes(stages []string) []model.PipelineNode {
	if len(stages) == 0 {
		stages = defaultPipelineStages
	}
	nodes := make([]model.PipelineNode, len(stages))
	for i, stage := range stages {
		nodes[i] = model.PipelineNode{
			ID:         fmt.Sprintf("node-%d", i),
			Type:       stage,
			Label:      stage,
			Scripts:    []model.PipelineScript{},
			X:          60 + (i * 210),
			Y:          120,
			VMNode:     "shared",
			Vm:         "shared-01",
			DestBucket: fmt.Sprintf("s3://%s/", stage),
			Schedule:   "1m",
			Cascade:    stage != "deploy",
		}
	}
	return nodes
}

func buildDefaultEdges(nodes []model.PipelineNode) []model.PipelineEdge {
	if len(nodes) < 2 {
		return []model.PipelineEdge{}
	}
	edges := make([]model.PipelineEdge, len(nodes)-1)
	for i := 0; i < len(nodes)-1; i++ {
		edges[i] = model.PipelineEdge{
			ID:         fmt.Sprintf("edge-%d", i),
			FromNodeId: nodes[i].ID,
			ToNodeId:   nodes[i+1].ID,
		}
	}
	return edges
}

func (s *TrainingService) GetAll(ownerID string) ([]model.TrainingJob, error) {
	return s.repo.GetAllByOwner(ownerID)
}

func (s *TrainingService) GetByID(jobID string, ownerID string) (model.TrainingJob, error) {
	return s.repo.GetByID(jobID, ownerID)
}

func (s *TrainingService) UpdateJob(req model.TrainingJobDto) (model.TrainingJob, error) {
	return s.repo.UpdateJob(req)
}

func sha256File(file multipart.File) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

type createPresignedURLResponse struct {
	UploadURL string `json:"upload_url"`
}
type ScriptUploadResponse struct {
	UploadURL string            `json:"upload_url"`
	Job       model.TrainingJob `json:"job"`
}

func (s *TrainingService) GetScriptUploadURL(ownerID, jobID, nodeID, nodeType, filename, contentType, routeTo string, file multipart.File) (ScriptUploadResponse, error) {
	job, err := s.repo.GetByID(jobID, ownerID)
	if err != nil {
		return ScriptUploadResponse{}, fmt.Errorf("job not found: %w", err)
	}

	checksum, err := sha256File(file)
	if err != nil {
		return ScriptUploadResponse{}, fmt.Errorf("hash file: %w", err)
	}

	s3Key := fmt.Sprintf("%s-node/scripts", nodeType)

	// build script record with s3 path
	script := model.PipelineScript{
		ID:         fmt.Sprintf("script-%s", uuid.New().String()[:8]),
		Name:       filename,
		Path:       fmt.Sprintf("s3://%s/%s", job.BucketName, s3Key),
		RouteTo:    routeTo,
		FileSha256: checksum,
	}

	presignReq := createPresignedURLRequest{
		UserID:     job.OwnerID,
		GameID:     job.ID,
		AssetID:    script.ID,
		AssetName:  job.Name,
		Key:        s3Key,
		Sha256:     checksum,
		AssetType:  AssetTypeScript,
		BucketName: fmt.Sprintf("%s-sg-project", job.BucketName),
	}

	respBytes, err := s.nc.Publisher.Request("s3.task.create_presigned_url", presignReq)
	if err != nil {
		return ScriptUploadResponse{}, fmt.Errorf("presign request failed: %w", err)
	}

	var presignResp createPresignedURLResponse
	if err := json.Unmarshal(respBytes, &presignResp); err != nil {
		return ScriptUploadResponse{}, fmt.Errorf("unmarshal presign resp: %w", err)
	}
	if presignResp.UploadURL == "" {
		return ScriptUploadResponse{}, fmt.Errorf("empty upload URL from s3 service")
	}

	// inject into node
	for i := range job.Nodes {
		if job.Nodes[i].ID == nodeID {
			job.Nodes[i].Scripts = append(job.Nodes[i].Scripts, script)
			break
		}
	}

	// save to db
	updatedJob, err := s.repo.UpdateJob(model.TrainingJobDto{
		ID:          job.ID,
		OwnerID:     job.OwnerID,
		Name:        job.Name,
		Description: job.Description,
		Nodes:       job.Nodes,
		Edges:       job.Edges,
		Pipeline:    job.Pipeline,
		Tags:        job.Tags,
		Status:      job.Status,
		Progress:    job.Progress,
	})
	if err != nil {
		return ScriptUploadResponse{}, fmt.Errorf("save script to db: %w", err)
	}

	return ScriptUploadResponse{
		UploadURL: presignResp.UploadURL,
		Job:       updatedJob,
	}, nil
}

func (s *TrainingService) AddScriptByPath(ownerID, jobID, nodeID string, script model.PipelineScript) (model.TrainingJob, error) {
	job, err := s.repo.GetByID(jobID, ownerID)
	if err != nil {
		return model.TrainingJob{}, fmt.Errorf("job not found: %w", err)
	}

	for i := range job.Nodes {
		if job.Nodes[i].ID == nodeID {
			job.Nodes[i].Scripts = append(job.Nodes[i].Scripts, script)
			break
		}
	}

	return s.repo.UpdateJob(model.TrainingJobDto{
		ID:          job.ID,
		OwnerID:     job.OwnerID,
		Name:        job.Name,
		Description: job.Description,
		Nodes:       job.Nodes,
		Edges:       job.Edges,
		Pipeline:    job.Pipeline,
		Tags:        job.Tags,
		Status:      job.Status,
		Progress:    job.Progress,
	})
}

func (s *TrainingService) ExecuteNode(ownerID, jobID, nodeID string, scriptID string, sessionID string) (NodeExecutionResult, error) {
	// 1. fetch the job to validate ownership + get node
	job, err := s.repo.GetByID(jobID, ownerID)
	if err != nil {
		return NodeExecutionResult{}, fmt.Errorf("job not found: %w", err)
	}
	// 2. find the node
	var targetNode *model.PipelineNode
	for i := range job.Nodes {
		if job.Nodes[i].ID == nodeID {
			targetNode = &job.Nodes[i]
			break
		}
	}
	if targetNode == nil {
		return NodeExecutionResult{}, fmt.Errorf("node %s not found in job %s", nodeID, jobID)
	}
	// 3. mark node as running
	startedAt := time.Now()
	err = s.repo.UpdateNodeStatus(jobID, nodeID, "running", startedAt, nil, "")
	if err != nil {
		return NodeExecutionResult{}, fmt.Errorf("failed to mark node running: %w", err)
	}
	// 4. execute — decide between full node or single script
	var execErr error
	if scriptID == "" {
		log.Printf("[EXECUTION] of all scripts ina nodes")
		// Execute all scripts in the node (Standard behavior)
		execErr = s.runAllNodeScripts(targetNode, job, sessionID)
	} else {
		log.Printf("[EXECUTION] of a single script in a node")

		// Execute only the specific script provided
		execErr = s.runSingleScript(targetNode, scriptID, job, sessionID)
	}
	// 5. mark node complete or failed
	finishedAt := time.Now()
	status := "completed"
	errMsg := ""
	if execErr != nil {
		status = "failed"
		errMsg = execErr.Error()
		log.Printf("node %s (script: %s) failed: %s", nodeID, scriptID, errMsg)
	}
	err = s.repo.UpdateNodeStatus(jobID, nodeID, status, startedAt, &finishedAt, errMsg)
	if err != nil {
		return NodeExecutionResult{}, fmt.Errorf("failed to update node status: %w", err)
	}
	// 6. trigger next node ONLY if full node execution was successful
	// (We usually don't cascade if only a single manual script was run)
	if execErr == nil && scriptID == "" && targetNode.Cascade {
		go s.triggerNextNode(ownerID, job, nodeID, sessionID)
	}
	return NodeExecutionResult{
		NodeID:     nodeID,
		Status:     status,
		StartedAt:  startedAt,
		FinishedAt: &finishedAt,
		SessionID:sessionID,
		Error:      errMsg,
	}, nil
}

// runSingleScript looks for a specific script within the node and executes it
func (s *TrainingService) runSingleScript(node *model.PipelineNode, scriptID string, job model.TrainingJob, sessionID string) error {
	var targetScript *model.PipelineScript
	for _, sc := range node.Scripts {
		if sc.ID == scriptID {
			targetScript = &sc
			break
		}
	}
	if targetScript == nil {
		return fmt.Errorf("script %s not found in node %s", scriptID, node.ID)
	}
	log.Printf("[SINGLE SCRIPT MODE] node: %s, running script: %s path: %s", node.ID, targetScript.Name, targetScript.Path)

	// Delegate to a shared execution helper

	log.Printf(">>>> EXECUTING: %s (Location: %s)", targetScript.Name, targetScript.Path)

	// TODO: Provision VM, sync S3 data, and run the script
	// ... we will solve this in the next step ...

	var req createPresignDownloadURLRequest

	req.AssetID = targetScript.ID
	req.FileSha256 = targetScript.FileSha256
	req.UserID = job.OwnerID
	req.CorrelationID = uuid.New().String()

	respBytes, err := s.nc.Publisher.Request("s3.task.create_presign_download_url", req)
	if err != nil {
		return fmt.Errorf("presign request failed: %w", err)
	}

	var presignResp createPresignDownloadURLResponse
	if err := json.Unmarshal(respBytes, &presignResp); err != nil {
		return fmt.Errorf("unmarshal presign resp: %w", err)
	}

	log.Printf(">>>> PRESIGNED_DOWNLOAD_URLS: %s )", presignResp)

	payload := EC2ProvisionRequest{
		Profile:   "ai-worker",
		Specs:     map[string]int{"cpu": 2, "ram": 4096},
		SessionID: sessionID,
		UserID:    job.OwnerID,
		Manifest: AIManifest{
			ProjID:    job.ID,
			ProjName:  job.Name,
			CreatedAt: job.CreatedAt,
		},
	}

	ec2RespBytes, err := s.nc.Publisher.Request("ec2.task.provision", payload)

	if err != nil {
		return fmt.Errorf("presign request failed: %w, %s", err, string(ec2RespBytes))
	}

	return nil

}

type AIManifest struct {
	ID         string            `json:"id"           gorm:"primaryKey"`
	ProjID     string            `json:"game_id"      gorm:"index"`
	ProjName   string            `json:"name"`
	CreatedAt  time.Time         `json:"created_at"`
	Parameters map[string]string `json:"parameters"`
}
type EC2ProvisionRequest struct {
	Profile    string            `json:"profile"`
	Specs      map[string]int    `json:"specs"`
	Parameters map[string]string `json:"parameters"`
	UserID     string            `json:"user_id"`
	StorageARN string            `json:"storage_arn"`
	Manifest   AIManifest        `json:"manifest"`
	SessionID  string            `json:"session_id"`
}

func (s *TrainingService) runAllNodeScripts(node *model.PipelineNode, job model.TrainingJob, sessionID string) error {
	if len(node.Scripts) == 0 {
		log.Printf("node %s has no scripts, skipping execution", node.ID)
		return nil
	}
	var req ArchiveByPrefixInputMessage
	req.BucketID = job.BucketID
	req.UserID = job.OwnerID
	s3Prefix := fmt.Sprintf("%s-node/scripts", node.Type)

	req.Prefix = s3Prefix //fmt.Sprintf("%s/%s", projectName, node.Label)
	req.ArchiveName = fmt.Sprintf("%s-archive-%s", node.Label, time.Now().Format("2006-01-02_15-04-05"))
	req.Format = "zip"

	log.Printf("[ArchiveByPrefix] raw response**: %s", job.OwnerID)

	respBytes, err := s.nc.Publisher.Request("s3.task.create_zip_download_url", req)
	if err != nil {
		return fmt.Errorf("presign request failed: %w", err)
	}

	// Log raw response

	var presignResp createPresignDownloadURLResponse
	if err := json.Unmarshal(respBytes, &presignResp); err != nil {
		return fmt.Errorf("unmarshal presign resp: %w", err)
	}

	log.Printf("[ArchiveByPrefix] Files returned: %d files, with this download url %s", presignResp.FileCount, presignResp.URL)

	payload := EC2ProvisionRequest{
		Profile:   "ai-worker",
		Specs:     map[string]int{"cpu": 2, "ram": 4096},
		SessionID: sessionID,
		UserID:    job.OwnerID,
		// Manifest: AIManifest{
		// 	ProjID:   job.ID,
		// 	ProjName: job.Name,
		// 	CreatedAt: job.CreatedAt,
		// },
	}
	// data, err := json.Marshal(payload)
	// 	log.Printf("PROVISION_GAME_PAYLOAD_MARSHAL_FAILED %s", data.)

	if err != nil {
		log.Printf("PROVISION_GAME_PAYLOAD_MARSHAL_FAILED")
		return fmt.Errorf("failed to marshal provision payload: %w", err)
	}
	ec2RespBytes, err := s.nc.Publisher.Request("ec2.task.provision", payload)

	if err != nil {
		return fmt.Errorf("presign request failed: %w, %s", err, string(ec2RespBytes))
	}
	return nil
}

func (s *TrainingService) triggerNextNode(ownerID string, job model.TrainingJob, currentNodeID, sessionID string) {
	// find the edge where fromNodeId == currentNodeID
	for _, edge := range job.Edges {
		if edge.FromNodeId == currentNodeID {
			log.Printf("cascading to next node: %s", edge.ToNodeId)
			_, err := s.ExecuteNode(ownerID, job.ID, edge.ToNodeId, sessionID, "")
			if err != nil {
				log.Printf("cascade execute failed: %s", err)
			}
			return
		}
	}
	// no next node — pipeline complete
	log.Printf("pipeline %s completed", job.ID)
	_ = s.repo.UpdateJobStatus(job.ID, "completed")
}
