package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	model "llm-inference-service/internal/domain/models"
	"llm-inference-service/internal/nats"
	"llm-inference-service/internal/repository"
	"llm-inference-service/internal/utils"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TrainingService struct {
	repo            *repository.PostgresRepository
	nc              *nats.Client
	natsPrefix      string
	controlPlaneURL string
}

func NewTrainingService(repo *repository.PostgresRepository, nc *nats.Client, natsPrefix string, controlPlaneURL string) *TrainingService {
	return &TrainingService{repo: repo,
		nc:              nc,
		natsPrefix:      natsPrefix,
		controlPlaneURL: controlPlaneURL,
	}
}

type CreateDefaultBucketRequest struct {
	CorrelationID  string   `json:"correlation_id"`
	UserID         string   `json:"user_id"`
	SessionID      string   `json:"session_id"`
	BucketName     string   `json:"bucket_name"`
	DefaultFolders []string `json:"default_folders"`
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
		CorrelationID:  uuid.New().String(),
		UserID:         ownerID,
		SessionID:      req.SessionID,
		BucketName:     fmt.Sprintf("%s-sg-project", bucketName),
		DefaultFolders: []string{"ingest", "clean", "train", "evaluate", "deploy"},
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

func (s *TrainingService) RemoveFile(fileId, jobID string ) error {

	return s.repo.RemoveFile(fileId,jobID)

}
func (s *TrainingService) AddFileRecord(jobID string, reqs []model.AddFileRequest) error {
	var dbFiles []model.LLMTrainingJobFile

	for _, req := range reqs {

		fileID := fmt.Sprintf("file-%d", time.Now().UnixNano())

		dbFiles = append(dbFiles, model.LLMTrainingJobFile{
			ID:        fileID,
			ProjectID: req.ProjectId,
			Name:      req.FileName,
			Role:      req.Role,
			Size:      req.Size,
			Type:      req.ContentType,
			Path:      "/games/data/s3Key",
			Sha256:    req.Sha256,
		})

	}

	if err := s.repo.AddFileToLLMJob(jobID, dbFiles); err != nil {
		log.Printf("[Service:AddFile] failed to persist file records for job %s: %v", jobID, err)
		return fmt.Errorf("failed to persist file records: %w", err)
	}

	return nil

}
func (s *TrainingService) AddFile(jobID string, reqs []model.AddFileRequest) ([]model.FileUploadResult, error) {

	var results []model.FileUploadResult
	var dbFiles []model.LLMTrainingJobFile

	job, err := s.repo.GetLLMJobByID(jobID)

	if err != nil {
		log.Printf("[Service:GetLLMJobByID] GetLLMJobByID request failed for jobid %s: %v", jobID, err)
		return nil, fmt.Errorf("presign request failed for %s: %w", jobID, err)
	}
	fmt.Printf("The coose bucket is && %s", job.BucketName)

	for _, req := range reqs {
		s3Key := fmt.Sprintf("%s/%s", req.Role, req.FileName)

		respBytes, err := s.nc.Publisher.Request("s3.task.create_presigned_url", model.CreatePresignedUploadURLRequest{
			UserID:     req.OwnerID,
			GameID:     req.ProjectId,
			AssetID:    req.ProjectId,
			AssetType:  "models",
			AssetName:  req.FileName,
			Key:        s3Key,
			Sha256:     req.Sha256,
			BucketName: job.BucketName,
		})
		if err != nil {
			log.Printf("[Service:AddFile] presign request failed for %s: %v", req.FileName, err)
			return nil, fmt.Errorf("presign request failed for %s: %w", req.FileName, err)
		}

		var presignResp struct {
			URL string `json:"upload_url"`
		}

		log.Println("Presign response bytes:", string(respBytes))

		if err := json.Unmarshal(respBytes, &presignResp); err != nil {
			log.Printf("[Service:AddFile] failed to parse presign response for %s: %v", req.FileName, err)
			return nil, fmt.Errorf("failed to parse presign response for %s: %w", req.FileName, err)
		}

		uploadURL := presignResp.URL
		if uploadURL != "" && !strings.HasPrefix(uploadURL, "http://") && !strings.HasPrefix(uploadURL, "https://") {
			uploadURL = s.controlPlaneURL + uploadURL
		}

		fileID := fmt.Sprintf("file-%d", time.Now().UnixNano())

		dbFiles = append(dbFiles, model.LLMTrainingJobFile{
			ID:        fileID,
			ProjectID: req.ProjectId,
			Name:      req.FileName,
			Role:      req.Role,
			Size:      req.Size,
			Type:      req.ContentType,
			Path:      s3Key,
			Sha256:    req.Sha256,
		})

		results = append(results, model.FileUploadResult{
			FileName:  req.FileName,
			UploadURL: uploadURL,
		})
	}

	if err := s.repo.AddFileToLLMJob(jobID, dbFiles); err != nil {
		log.Printf("[Service:AddFile] failed to persist file records for job %s: %v", jobID, err)
		return nil, fmt.Errorf("failed to persist file records: %w", err)
	}

	return results, nil
}

func (s *TrainingService) CreateLLMProject(ownerID string, req model.CreateLLMJobRequest, host string) (model.CreateLLMJobResponse, error) {
	projID := req.ID
	if projID == "" {
		projID = fmt.Sprintf("proj-%d", time.Now().UnixNano()/1e6)
	}

	jobType := req.Type
	if jobType == "" {
		jobType = "llm"
	}
	baseModel := req.BaseModel
	if baseModel == "" {
		baseModel = "Serwin Small 1B"
	}
	trainingMode := req.TrainingMode
	if trainingMode == "" {
		trainingMode = "serwin"
	}
	schedule := req.Schedule
	if schedule == "" {
		schedule = "manual"
	}
	pipeline := req.Pipeline
	if len(pipeline) == 0 {
		pipeline = []string{"ingest", "clean", "train", "evaluate", "deploy"}
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	entryPoint := req.EntryPoint
	if entryPoint == "" {
		entryPoint = "main.py"
	}
	bucketName := sanitizeBucketName(req.Name)
	now := time.Now()

	// check if a project with thsi name exists,
	allJobs, err := s.repo.GetAllLLMJobs(ownerID)
	if err != nil {
		return model.CreateLLMJobResponse{}, fmt.Errorf("Error gettin your project  %w", err)
	}

	for _, oneJob := range allJobs {
		if oneJob.Name == req.Name {
			return model.CreateLLMJobResponse{}, fmt.Errorf("A job with that name already exists   %w", err)

		}
	}

	job := model.LLMTrainingJob{
		ID:                 projID,
		OwnerID:            ownerID,
		Name:               req.Name,
		Description:        req.Description,
		Type:               jobType,
		BaseModel:          baseModel,
		TrainingDataSource: req.TrainingDataSource,
		TrainingDataURL:    req.TrainingDataURL,
		TrainingDataBucket: req.TrainingDataBucket,
		UseValidationData:  req.UseValidationData,
		TrainingMode:       trainingMode,
		EntryPoint:         entryPoint,
		Compute:            req.Compute,
		Schedule:           schedule,
		Pipeline:           pipeline,
		Tags:               tags,
		Status:             "idle",
		BucketName:         bucketName,
		CreatedAt:          now,
		UpdatedAt:          now,
		CallBack:           req.CallBack,
	}

	if host == "" {
		host = "localhost:8080"
	}
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")

	var dbFiles []model.LLMTrainingJobFile
	var presignedURLs []model.PresignedURLInfo

	type DefaultBucketEvent struct {
		CorrelationID  string   `json:"correlation_id"`
		UserID         string   `json:"user_id"`
		SessionID      string   `json:"session_id"`
		BucketName     string   `json:"bucket_name"`
		DefaultFolders []string `json:"default_folders"`
	}
	eventPayload := DefaultBucketEvent{
		CorrelationID: uuid.New().String(),
		UserID:        ownerID,
		SessionID:     uuid.New().String(),
		BucketName:    fmt.Sprintf("%s-sg-bucket", req.Name),
		// DefaultFolders : []string{"outputs"}, donot uncomment this
	}

	defaultBucketResponseBytes, err := s.nc.Publisher.Request("s3.task.create_default_bucket", eventPayload)
	if err != nil {
		return model.CreateLLMJobResponse{}, fmt.Errorf("create_default_bucket request failed: %w", err)
	}
	fmt.Println("che 1")
	type CreateDefaultBucketResponse struct {
		BucketName string `json:"bucket_name"`
		BucketID   string `json:"bucket_id"`
		Created    bool   `json:"created"`
		Error      string `json:"error,omitempty"`
	}
	var defaultBucketResponse CreateDefaultBucketResponse
	if err := json.Unmarshal(defaultBucketResponseBytes, &defaultBucketResponse); err != nil {
		return model.CreateLLMJobResponse{}, fmt.Errorf("unmarshal create_default_bucket resp: %w", err)
	}
	job.BucketID = defaultBucketResponse.BucketID
	job.BucketName = defaultBucketResponse.BucketName
	fmt.Printf("=*=*=> %+v", defaultBucketResponse)
	fmt.Println("che 2")

	for i, f := range req.Files {
		fileID := fmt.Sprintf("file-%s-%d", uuid.New().String()[:8], i)
		fileRole := f.Role
		if fileRole == "" {
			fileRole = "training"
		}
		checksum := CalculateSHA256Bytes([]byte(f.Name))
		s3Key := fmt.Sprintf("%s/%s", f.Role, f.Name)
		targetBucket := fmt.Sprintf("%s-sg-project", bucketName)
		s3Path := fmt.Sprintf("s3://%s/%s", targetBucket, s3Key)

		presignReq := model.CreatePresignedUploadURLRequest{
			UserID:     ownerID,
			GameID:     projID,
			AssetID:    projID,
			AssetName:  req.Name,
			AssetType:  model.AssetTypeScript,
			BucketName: job.BucketName,
			Key:        s3Key,
			Sha256:     checksum,
		}

		var uploadURL string
		if s.nc != nil && s.nc.Publisher != nil {
			respBytes, err := s.nc.Publisher.Request("s3.task.create_presigned_url", presignReq)
			if err == nil {
				var presignResp model.CreatePresigneUploaddURLResponse
				if err := json.Unmarshal(respBytes, &presignResp); err == nil && presignResp.UploadURL != "" {
					uploadURL = presignResp.UploadURL
				} else {
					log.Printf("[CreateLLMProject] error unmarshaling presign resp: %v, raw: %s", err, string(respBytes))
				}
			} else {
				log.Printf("[CreateLLMProject] NATS request s3.task.create_presigned_url failed: %v", err)
			}
		}

		if uploadURL == "" {
			uploadURL = fmt.Sprintf("https://%s.s3.amazonaws.com/%s?token=presigned_%s", targetBucket, s3Key, uuid.New().String()[:8])
		}
		uploadURL = strings.Replace(uploadURL, "/api/v1", "", 1)
		fmt.Println("che 3")

		dbFile := model.LLMTrainingJobFile{
			ID:        fileID,
			ProjectID: projID,
			Name:      f.Name,
			Role:      fileRole,
			Size:      f.Size,
			Type:      f.Type,
			Path:      s3Path,
			CreatedAt: now,
			Sha256:    checksum,
		}
		dbFiles = append(dbFiles, dbFile)

		presignedURLs = append(presignedURLs, model.PresignedURLInfo{
			Name: f.Name,
			Role: fileRole,
			URL:  uploadURL,
		})
	}

	type CreateJobApiKey struct {
		Name             string   `json:"name"`
		Description      string   `json:"description"`
		AllowedActions   []string `json:"allowed_actions"`
		AllowedResources []string `json:"allowed_resources"`
		ExpiresAt        string   `json:"expires_at"`
		UserID           string   `json:"userId"`
	}

	apiKeyReq := CreateJobApiKey{
		Name:             "sg-" + job.Name + "key",
		Description:      "An auto generated api key to execute " + job.Name + " sg project",
		AllowedActions:   []string{},
		AllowedResources: []string{},
		UserID:           ownerID,
		// ExpiresAt: ,
	}
	respBytesApiKey, err := s.nc.Publisher.Request("auth.task.create_api_key", apiKeyReq)

	// the response from this is
	/*
	   @Data
	   @AllArgsConstructor
	   @NoArgsConstructor
	   @ToString
	   public class InstanceTokenResponse {
	         public String token;
	         public String error;
	   }*/

	type ApiKeyResponse struct {
		ApiKey string `json:"token"`
		Erorr  string `json:"error"`
	}
	fmt.Println("che 4")

	var apiKeyResponse ApiKeyResponse
	err = json.Unmarshal(respBytesApiKey, &apiKeyResponse)

	if err != nil {
		fmt.Printf("The error we recieved whn gettin n api key for this job   %+v", apiKeyResponse)
		return model.CreateLLMJobResponse{}, nil
	}

	fmt.Printf("This is the training api key we recieved  %+v", apiKeyResponse)

	job.ApiKey = apiKeyResponse.ApiKey

	if err := s.repo.CreateLLMJob(job, dbFiles); err != nil {
		return model.CreateLLMJobResponse{}, fmt.Errorf("create llm job: %w", err)
	}

	return model.CreateLLMJobResponse{
		ID:            job.ID,
		Name:          job.Name,
		Description:   job.Description,
		Type:          job.Type,
		BaseModel:     job.BaseModel,
		Status:        job.Status,
		CreatedAt:     job.CreatedAt,
		PresignedURLs: presignedURLs,
	}, nil

	//
}

func (s *TrainingService) GetAllLLMProjects(ownerID string) ([]model.LLMTrainingJob, error) {
	return s.repo.GetAllLLMJobs(ownerID)
}

func (s *TrainingService) GetLLMProjectByID(id string) (model.LLMTrainingJob, error) {
	return s.repo.GetLLMJobByID(id)
}

func (s *TrainingService) UpdateLLMProject(id string, req model.LLMTrainingJob) (model.LLMTrainingJob, error) {
	return s.repo.UpdateLLMJob(id, req)
}

func (s *TrainingService) DeleteLLMProject(id string) error {
	return s.repo.DeleteLLMJob(id)
}

func (s *TrainingService) HandleVMReportStatus(
	req model.VMReportStatus,
) error {

	log.Printf(
		"1[VMReportStatus] run=%s step=%s status=%s duration=%f",
		req.RunID,
		req.Step,
		req.Status,
		req.Duration,
	)

	// ---------------------------------------------------------
	// 1. Get the current pipeline job
	// ---------------------------------------------------------

	job, err := s.repo.GetPipelineJobByRunAndStep(
		req.RunID,
		req.Step,
	)
	log.Printf("-------------->> %+v", job)
	originalJobVmID := job.VMID
	originalJobVmIp := job.VMIP
	originalProjectID := job.ProjectID
	if err != nil {
		log.Printf(
			"2.0 failed to get pipeline job: %v",
			err,
		)
		return fmt.Errorf(
			"failed to get pipeline job: %w",
			err,
		)
	}

	log.Printf(
		"2[VMReportStatus] run=%s step=%s status=%s duration=%f with vmid: %s and vmip: %s",
		req.RunID,
		req.Step,
		req.Status,
		req.Duration,
		job.VMID,
		job.VMIP,
	)

	if job == nil {
		log.Printf(
			"2.1[VMReportStatus] run=%s step=%s status=%s duration=%f",
			req.RunID,
			req.Step,
			req.Status,
			req.Duration,
		)
		return fmt.Errorf(
			"pipeline job not found: run=%s step=%s",
			req.RunID,
			req.Step,
		)
	}

	// Send a message to ec2 to release the vm, nomatter whether its success or failure
	// as long as it the last step ie deploy
	if req.Step == "deploy" {

		err = utils.ReleaseSageMakerVM(s.nc, job.VMID)

	}

	// ---------------------------------------------------------
	// 2. VM reports failure
	// ---------------------------------------------------------

	if req.Status == "failed" {

		log.Printf(
			"[VMReportStatus] step failed: run=%s step=%s",
			req.RunID,
			req.Step,
		)

		// Mark the current job failed.
		err = s.repo.MarkPipelineJobFailed(
			job.ID,
			req.Duration,
		)

		if err != nil {
			return fmt.Errorf(
				"failed to mark pipeline job failed: %w",
				err,
			)
		}

		// Mark entire pipeline failed.
		err = s.repo.MarkPipelineFailed(
			req.RunID,
		)

		if err != nil {
			return fmt.Errorf(
				"failed to mark pipeline failed: %w",
				err,
			)
		}

		// releasevm

		// err = .releaseSageMakerVM()
		err = utils.ReleaseSageMakerVM(s.nc, job.VMID)

		// callthe projects callback

		return nil
	}
	log.Printf(
		"3[VMReportStatus] run=%s step=%s status=%s duration=%f",
		req.RunID,
		req.Step,
		req.Status,
		req.Duration,
	)
	// ---------------------------------------------------------
	// 3. VM reports success
	// ---------------------------------------------------------

	if req.Status != "success" {
		return fmt.Errorf(
			"unknown VM status: %s",
			req.Status,
		)
	}
	// Mark current step completed.
	err = s.repo.MarkPipelineJobCompleted(
		job.ID,
		req.Duration,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to mark pipeline job completed: %w",
			err,
		)
	}

	log.Printf(
		"4[VMReportStatus] run=%s step=%s status=%s duration=%f",
		req.RunID,
		req.Step,
		req.Status,
		req.Duration,
	)

	// ---------------------------------------------------------
	// 4. Determine next step
	// ---------------------------------------------------------
	log.Printf("-----------------------------------------%s", req.Step)
	nextStep := ""

	switch req.Step {

	case "ingest":
		nextStep = "train"

	case "train":
		nextStep = "validate"

	case "validate":
		nextStep = "deploy"

	case "deploy":

		// Pipeline is completely finished.
		err = s.repo.MarkPipelineCompleted(
			req.RunID,
		)

		if err != nil {
			return fmt.Errorf(
				"failed to mark pipeline completed: %w",
				err,
			)
		}

		log.Printf(
			"[VMReportStatus] pipeline completed: %s",
			req.RunID,
		)

		return nil

	default:
		return fmt.Errorf(
			"unknown pipeline step: %s",
			req.Step,
		)
	}

	// ---------------------------------------------------------
	// 5. Create next pipeline job
	// ---------------------------------------------------------

	nextJob, err := s.repo.AddPipelineJobComplete(
		model.AddPipelineJobRequest{
			RunID:     req.RunID,
			Step:      nextStep,
			Status:    "queued",
			VMID:      originalJobVmID,
			VMIP:      originalJobVmIp,
			ProjectID: originalProjectID,
		},
	)

	//  err = s.repo.UpdatePipelineJobVM(
	// 	job.ID,
	// 	originalJobVmID,
	// 	originalJobVmIp,
	// )

	log.Printf("next Step {%s} nect job %+v", nextStep, nextJob)

	if err != nil {
		log.Printf("next Step {%s} failed err %v", nextStep, err)
		return fmt.Errorf(
			"failed to create next pipeline job: %w",
			err,
		)
	}

	err = s.repo.UpdatePipelineJobVM(
		nextJob.ID,
		nextJob.VMID,
		job.VMIP,
	)

	if err != nil {
		log.Println(
			"[orchestrator] failed to save VM info:",
			err,
		)
	}

	log.Printf(
		"[VMReportStatus] next step queued: run=%s step=%s",
		req.RunID,
		nextStep,
	)

	// ---------------------------------------------------------
	// 6. Reuse the same VM
	// ---------------------------------------------------------

	vmIP := job.VMIP

	if vmIP == "" {
		return fmt.Errorf(
			"VM IP missing for job %s",
			job.ID,
		)
	}

	// ---------------------------------------------------------
	// 7. Mark next job running BEFORE calling VM
	// ---------------------------------------------------------

	err = s.repo.MarkPipelineJobRunning(
		nextJob.ID,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to mark next job running: %w",
			err,
		)
	}

	log.Printf(
		"[VMReportStatus] started next step: run=%s step=%s vm=%s",
		req.RunID,
		nextStep,
		vmIP,
	)

	// ---------------------------------------------------------
	// 8. Start next step on VM
	// ---------------------------------------------------------

	switch nextStep {

	case "train":

		err = s.startTrain(
			vmIP,
			req.RunID,
		)

	case "validate":

		err = s.startValidate(
			vmIP,
			req.RunID,
		)

	case "deploy":

		err = s.startDeploy(
			vmIP,
			req.RunID,
			job.ProjectID,
		)
	}

	if err != nil {

		// The VM could not even be instructed to start
		// the next step, so fail the pipeline.

		_ = s.repo.MarkPipelineJobFailed(
			nextJob.ID,
			0,
		)

		_ = s.repo.MarkPipelineFailed(
			req.RunID,
		)

		return fmt.Errorf(
			"failed to start %s: %w",
			nextStep,
			err,
		)
	}

	return nil
}

func (s *TrainingService) startTrain(
	vmIP string,
	runID string,
) error {

	project, err := s.repo.GetLLMProjectByRunID(runID)
	if err != nil {
		log.Printf("Starting training, gettin projrct err %v", err)
		return err
	}

	fmt.Printf("starting trainig* %+v\n", project)

	req := map[string]interface{}{
		"run_id":        runID,
		"control_plane": fmt.Sprintf("%s/api/v1/llm", s.controlPlaneURL),
		"entry_point":   project.EntryPoint,
		"api_key":       project.ApiKey,
	}

	_, err = s.callVM(
		vmIP,
		"/train",
		req,
	)

	return err
}

func (s *TrainingService) startValidate(
	vmIP string,
	runID string,
) error {

	project, err := s.repo.GetLLMProjectByRunID(runID)
	if err != nil {
		log.Printf("Starting validation, gettin projrct err %v", err)
		return err
	}

	fmt.Printf("starting validation")

	req := map[string]interface{}{
		"run_id":        runID,
		"control_plane": fmt.Sprintf("%s/api/v1/llm", s.controlPlaneURL),
		"api_key":       project.ApiKey,
	}

	_, err = s.callVM(
		vmIP,
		"/validate",
		req,
	)

	return err
}

type DeployResponse struct {
	Status         string             `json:"status"`
	RunID          string             `json:"run_id"`
	DeploymentMode string             `json:"deployment_mode"`
	Artifacts      []DeployedArtifact `json:"artifacts"`
}

type DeployedArtifact struct {
	FileName string `json:"file_name"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
}

func (s *TrainingService) startDeploy(
	vmIP string,
	runID, jobid string,
) error {

	project, err := s.repo.GetLLMProjectByRunID(runID)
	if err != nil {
		log.Printf("Starting deployment, gettin projrct err %v", err)
		return err
	}

	fmt.Printf("starting deployment")

	req := map[string]interface{}{
		"run_id":          runID,
		"job_id":          jobid,
		"deployment_mode": "artifact",
		"control_plane":   fmt.Sprintf("%s/api/v1/llm", s.controlPlaneURL),
		"api_key":         project.ApiKey,
	}

	body, err := s.callVM(
		vmIP,
		"/deploy",
		req,
	)
	if err != nil {
		return err
	}

	var deployResponse DeployResponse

	if err := json.Unmarshal(body, &deployResponse); err != nil {
		return fmt.Errorf(
			"failed to parse VM deploy response: %w",
			err,
		)
	}

	fmt.Printf(
		"deployment response: %+v\n",
		deployResponse,
	)

	for _, artifact := range deployResponse.Artifacts {
		fmt.Printf(
			"deployed artifact: %s sha256=%s size=%d\n for project callback: %s",
			artifact.FileName,
			artifact.SHA256,
			artifact.Size,
			project.CallBack,
		)
	}

	if project.CallBack != "" {
		err = s.notifyCallback(
			project.CallBack,
			runID,
			project.ID,
			deployResponse.Artifacts,
		)
		if err != nil {
			return fmt.Errorf("failed to notify callback: %w", err)
		}
	}

	return nil
}

func (s *TrainingService) notifyCallback(
	callbackURL string,
	runID string,
	projectID string,
	artifacts []DeployedArtifact,
) error {

	payload := map[string]interface{}{
		"event":      "artifact_ready",
		"run_id":     runID,
		"project_id": projectID,
		"artifacts":  artifacts,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal callback payload: %w", err)
	}

	log.Printf(
		"[Callback] POST %s",
		callbackURL,
	)

	req, err := http.NewRequest(
		http.MethodPost,
		callbackURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed to create callback request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("callback request failed: %w", err)
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read callback response: %w", err)
	}

	fmt.Printf(
		"[Callback] Response status=%d body=%s\n",
		resp.StatusCode,
		string(responseBody),
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"callback returned HTTP %d",
			resp.StatusCode,
		)
	}

	return nil
}

func (s *TrainingService) callVM(
	vmIP string,
	endpoint string,
	payload interface{},
) ([]byte, error) {

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"http://%s:9034%s",
		vmIP,
		endpoint,
	)

	log.Printf(
		"[VM] POST %s",
		url,
	)

	req, err := http.NewRequest(
		http.MethodPost,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	bodyResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read VM response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"VM returned HTTP %d: %s",
			resp.StatusCode,
			string(bodyResp),
		)
	}

	return bodyResp, nil
}

func (s *TrainingService) RunPipeline(project model.LLMTrainingJob) (model.TrainingQueue, error) {
	runID := uuid.New().String()

	req := model.AddTrainingJobRequest{
		ProjectID: project.ID,
		Status:    "queued",
	}

	trainingQueue, err := s.repo.AddTrainingJob(runID, req)
	if err != nil {
		return model.TrainingQueue{}, err
	}
	trainingQueue.WSUrl = "http://localhost:8080/api/v1/llm/ws/runs/" + trainingQueue.RunID

	return trainingQueue, nil
}

func CalculateSHA256Bytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
