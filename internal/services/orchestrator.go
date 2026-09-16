package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	// "net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	model "llm-inference-service/internal/domain/models"
	"llm-inference-service/internal/nats"
	"llm-inference-service/internal/repository"
	"llm-inference-service/internal/transport/ws"
	"llm-inference-service/internal/utils"

	// "llm-inference-service/internal/utils"

	"github.com/google/uuid"
)

type OrchestratorService struct {
	repo         *repository.PostgresRepository
	nc           *nats.Client
	modelRepo    repository.ModelRepository
	controlPlane string
	maxPipelines int
	mgr          *ws.Manager
	natsPrefix   string
}

func NewOrchestratorService(
	repo *repository.PostgresRepository,
	modelRepo repository.ModelRepository,
	nc *nats.Client,
	natsPrefix string,
	controlPlane string,
	maxPipelines int,
	mgr *ws.Manager,
) *OrchestratorService {

	return &OrchestratorService{
		repo:         repo,
		nc:           nc,
		natsPrefix:   natsPrefix,
		modelRepo:    modelRepo,
		controlPlane: controlPlane,
		maxPipelines: maxPipelines,
		mgr:          mgr,
	}
}

// s.nc.Publisher.Request

func (o *OrchestratorService) Start() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// release all vms before starting he jobs loop
	// select vm_id distinct from pipeline_jobs where status='failed'

	vmIds, err := o.repo.FailedJobs()
	if err != nil {
		log.Println("[orchestrator] failed to get failed jobs:", err)

	}

	for _, vmdId := range vmIds {
		log.Printf("\n[orchestrator] releasing vm: %s before job loop\n", vmdId)

		utils.ReleaseSageMakerVM(o.nc, vmdId)
	}

	for range ticker.C {
		o.checkDueScheduledJobs()

		// --------------------------------------------------
		// 1. Check how many pipelines are currently running
		// --------------------------------------------------

		running, err := o.repo.CountRunningPipelines()
		if err != nil {
			log.Println("[orchestrator] failed to count running pipelines:", err)
			continue
		}
		if running >= o.maxPipelines {
			continue
		}

		log.Printf("current running pipelines: %v", running)

		// --------------------------------------------------
		// 2. Get oldest queued pipeline
		// --------------------------------------------------

		run, err := o.repo.GetQueuedRun()
		if err != nil {
			log.Println("[orchestrator] failed to get queued run:", err)
			continue
		}

		if run == nil {
			continue
		}

		log.Println("[orchestrator] found queued run:", run.ID)

		// --------------------------------------------------
		// 6. Provision EC2 VM
		//step 6 is up here because if we have no free sagemaker vm then dont do anything
		// just return the details to th euser
		// --------------------------------------------------

		vm, err := o.provisionSageMakerVM(run.ID, o.controlPlane)
		if err != nil {

			log.Println(
				"[orchestrator] failed to provision VM:",
				err,
			)
			// data, _ := json.Marshal(err)
			// o.mgr.BroadcastToRun(run.ID, data)

			continue
		}

		log.Println(
			"[orchestrator] VM provisioned:",
			vm.VMID,
			vm.VMIP,
		)

		// --------------------------------------------------
		// 3. Claim the pipeline
		// --------------------------------------------------

		err = o.repo.MarkPipelineRunning(run.ID)
		if err != nil {
			log.Println(
				"[orchestrator] failed to mark run running:",
				err,
			)
			continue
		}

		log.Println(
			"[orchestrator] pipeline marked running:",
			run.ID,
		)

		// --------------------------------------------------
		// 4. For now, the first step MUST be ingest
		// --------------------------------------------------

		if len(run.Pipeline) == 0 {
			log.Println(
				"[orchestrator] pipeline has no steps:",
				run.ID,
			)
			continue
		}

		step := run.Pipeline[0]

		if step != "ingest" {
			log.Println(
				"[orchestrator] expected first step to be ingest, got:",
				step,
			)
			continue
		}

		// --------------------------------------------------
		// 5. Create pipeline job
		// --------------------------------------------------

		job, err := o.repo.AddPipelineJob(
			model.AddPipelineJobRequest{
				RunID:     run.ID,
				Step:      "ingest",
				Status:    "queued",
				ProjectID: run.ProjectID,
			},
		)

		if err != nil {
			log.Println(
				"[orchestrator] failed to create ingest job:",
				err,
			)
			continue
		}

		log.Println(
			"[orchestrator] created ingest job:",
			job.ID,
		)
 
		// --------------------------------------------------
		// 7. Save VM information against pipeline job
		// --------------------------------------------------

		err = o.repo.UpdatePipelineJobVM(
			job.ID,
			vm.VMID,
			vm.VMIP,
		)

		if err != nil {
			log.Println(
				"[orchestrator] failed to save VM info:",
				err,
			)
			continue
		}

		// --------------------------------------------------
		// 8. Get project files
		// --------------------------------------------------

		files, err := o.repo.GetProjectFiles(run.ProjectID)
		if err != nil {
			log.Println(
				"[orchestrator] failed to get project files:",
				err,
			)
			continue
		}

		baseModel, err := o.GetBaseModel(run.ProjectID)
		if err != nil {
			log.Println(
				"[orchestrator] failed to get project basemodel:",
				err,
			)
			continue
		}

		// --------------------------------------------------
		// 9. Construct ingest data
		// --------------------------------------------------

		ingestData := o.buildOrchestratorData(
			run.ID,
			files,
			baseModel,
			run.ProjectID,
		)

		// --------------------------------------------------
		// 10. Send ingest command to VM
		// --------------------------------------------------

		err = o.sendIngest(
			vm.VMIP,
			ingestData,
		)

		if err != nil {
			log.Println(
				"[orchestrator] failed to send ingest command:",
				err,
			)
			continue
		}

		log.Println(
			"[orchestrator] ingest started:",
			run.ID,
		)

		_, _ = o.ScheduleNextRun(run.ProjectID)
	}
}

func parseScheduleInterval(scheduleStr string) (time.Duration, error) {
	s := strings.TrimSpace(strings.ToLower(scheduleStr))
	if s == "" || s == "manual" || s == "0" || s == "none" || s == "disabled" {
		return 0, nil
	}

	// 1. Check for pure integer/float number, e.g. "10", "15", "60" -> interpreted as minutes
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		if val <= 0 {
			return 0, nil
		}
		return time.Duration(val * float64(time.Minute)), nil
	}

	// 2. Check for keywords
	switch s {
	case "hourly":
		return time.Hour, nil
	case "daily":
		return 24 * time.Hour, nil
	case "weekly":
		return 7 * 24 * time.Hour, nil
	}

	// 3. Check for standard Go duration string, e.g. "10m", "15m", "1h", "30s"
	dur, err := time.ParseDuration(s)
	if err == nil {
		if dur <= 0 {
			return 0, nil
		}
		return dur, nil
	}

	// 4. Try parsing patterns like "15 minutes", "15 mins", "2 hours"
	re := regexp.MustCompile(`^(\d+)\s*(m|min|minute|minutes|h|hr|hour|hours|s|sec|second|seconds)?$`)
	matches := re.FindStringSubmatch(s)
	if len(matches) >= 2 {
		val, parseErr := strconv.Atoi(matches[1])
		if parseErr == nil && val > 0 {
			unit := "m"
			if len(matches) >= 3 && matches[2] != "" {
				unit = matches[2]
			}
			switch {
			case strings.HasPrefix(unit, "h"):
				return time.Duration(val) * time.Hour, nil
			case strings.HasPrefix(unit, "s"):
				return time.Duration(val) * time.Second, nil
			default:
				return time.Duration(val) * time.Minute, nil
			}
		}
	}

	return 0, fmt.Errorf("unrecognized schedule format: %s", scheduleStr)
}

func (o *OrchestratorService) ScheduleNextRun(project_id string) (*model.SageMakerVM, error) {
	job, err := o.repo.GetLLMJobByID(project_id)
	if err != nil {
		log.Printf("[orchestrator] failed to get job for scheduling (project_id: %s): %v", project_id, err)
		return nil, err
	}

	interval, err := parseScheduleInterval(job.Schedule)
	if err != nil {
		log.Printf("[orchestrator] invalid schedule format '%s' for project %s: %v", job.Schedule, project_id, err)
		return nil, err
	}

	if interval <= 0 {
		log.Printf("[orchestrator] project %s schedule is manual ('%s'), skipping next schedule", project_id, job.Schedule)
		return nil, nil
	}

	now := time.Now()
	nextRunAt := now.Add(interval)
	intervalSeconds := int64(interval.Seconds())

	log.Printf("[orchestrator] project %s scheduled for next run in %v at %s (interval: %ds)", project_id, interval, nextRunAt.Format(time.RFC3339), intervalSeconds)

	err = o.repo.UpdateJobScheduleInfo(project_id, intervalSeconds, nextRunAt, now)
	if err != nil {
		log.Printf("[orchestrator] failed to update schedule info in DB for project %s: %v", project_id, err)
		return nil, err
	}

	time.AfterFunc(interval, func() {
		log.Printf("[orchestrator] timer fired for scheduled project %s, enqueuing pipeline run", project_id)
		hasActive, err := o.repo.HasActiveOrQueuedRun(project_id)
		if err != nil {
			log.Printf("[orchestrator] failed to check active runs for project %s: %v", project_id, err)
			return
		}
		if hasActive {
			log.Printf("[orchestrator] project %s already has an active or queued run, skipping duplicate enqueue", project_id)
			return
		}

		runID, err := o.repo.EnqueuePipelineRun(project_id)
		if err != nil {
			log.Printf("[orchestrator] failed to enqueue scheduled run for project %s: %v", project_id, err)
		} else {
			log.Printf("[orchestrator] successfully enqueued scheduled run %s for project %s", runID, project_id)
		}
	})

	return nil, nil
}

func (o *OrchestratorService) checkDueScheduledJobs() {
	dueJobs, err := o.repo.GetDueScheduledJobs()
	if err != nil {
		log.Println("[orchestrator] failed to fetch due scheduled jobs:", err)
		return
	}

	for _, job := range dueJobs {
		hasActive, err := o.repo.HasActiveOrQueuedRun(job.ID)
		if err != nil {
			log.Printf("[orchestrator] failed to check active run for due job %s: %v", job.ID, err)
			continue
		}

		interval, err := parseScheduleInterval(job.Schedule)
		if err != nil || interval <= 0 {
			continue
		}

		now := time.Now()
		nextRunAt := now.Add(interval)
		intervalSec := int64(interval.Seconds())
		_ = o.repo.UpdateJobScheduleInfo(job.ID, intervalSec, nextRunAt, now)

		if !hasActive {
			runID, err := o.repo.EnqueuePipelineRun(job.ID)
			if err != nil {
				log.Printf("[orchestrator] failed to enqueue due run for project %s: %v", job.ID, err)
			} else {
				log.Printf("[orchestrator] enqueued due run %s for project %s", runID, job.ID)
			}
		}
	}
}

// func (o *OrchestratorService) releaseSageMakerVM(vmId string) error {

// 	subject := "ec2.task.release_lambda"
// 	type ReleaseVMReq struct {
// 		VMID string `json:"vm_id"`
// 	}

// 	req := ReleaseVMReq{
// 		VMID: vmId,
// 	}

// 	log.Printf("Thisis ht ereq being sent ;  %+v", req)

// 	respBytes, err := o.nc.Publisher.Request(subject, req)

// 	if err != nil {
// 		fmt.Printf("\nError sending reelease vmevent  %v", err)
// 		return fmt.Errorf("Error sending reelease vmevent  %v", err)
// 	}

// 	type RealeaseResponse struct {
// 		Code int `json:"code"`
// 	}

// 	var releaseVmresp RealeaseResponse
// 	err = json.Unmarshal(respBytes, &releaseVmresp)

// 	if err != nil {
// 		fmt.Printf("\nError unmarshalling  reelease vmevent response   %v", err)
// 		return fmt.Errorf("Error unmarshalling  reelease vmevent response  %v", err)

// 	}
// 	if releaseVmresp.Code != 200 {
// 		fmt.Printf("\nError in EC@ Couldnot release the vm:    %v", releaseVmresp.Code)
// 		return fmt.Errorf("Error in EC@ Couldnot release the vm:    %v", releaseVmresp.Code)

// 	} else {
// 		fmt.Printf("\n EC2  released the vm succesfully:    %v", releaseVmresp.Code)
// 	}

// 	return nil

// }
func (o *OrchestratorService) provisionSageMakerVM(runId string, controlPlane string) (*model.SageMakerVM, error) {

	payload := map[string]interface{}{
		"profile": "sagemaker",
	}

	subject := fmt.Sprintf(
		"ec2.task.provision",
	)
	log.Println("[orchestrator] failed to fetch due scheduled jobs:", subject)

	// Adapt this one call to the exact API
	// of your internal nats.Client.

	response, err := o.nc.Publisher.Request(
		subject,
		payload,
	)

	if err != nil {
		return nil, err
	}

	var vm model.SageMakerVM

	err = json.Unmarshal(response, &vm)
	if err != nil {
		return nil, err
	}
	fmt.Printf("THe returned vm details: %+v", vm)

	if vm.Code == 400 {
		fmt.Printf("All sagemaker vms already in use, standby ")
		return nil, fmt.Errorf(
			"All sagemaker vms already in use, standby ",
		)

	}
	if vm.VMID == "" || vm.VMIP == "" {
		return nil, fmt.Errorf(
			"EC2 returned invalid VM information",
		)
	}
	// url , _ := url.Parse(o.controlPlane)
	// hostname:=url.Hostname()
	// vm.VMIP =  hostname

	// before we mark the pipeline job as runing we ping the vm, and the vm should reply with code
	// 200, ie the controlplane must be able to rech the vm and the vmmust be able to reach the control plane
	pingJob, err := o.repo.GetLLMProjectByRunID(runId)
	if err != nil {
		log.Println(
			"[orchestrator] failed to get job by run id fo pinging",
			err,
		)
		err = utils.ReleaseSageMakerVM(o.nc, vm.VMID)

		return nil, fmt.Errorf("[orchestrator] failed to get job by run id fo pinging: %v", err)
	}

	errr := o.sendPingToVm(vm.VMIP, pingJob.ApiKey, runId, controlPlane)

	if errr != nil {
		log.Println(
			"[orchestrator] failed to send ping: %v",
			errr,
		)
		errrr := utils.ReleaseSageMakerVM(o.nc, vm.VMID)

		return nil, fmt.Errorf("[orchestrator] Sending ping to vm failed: %v", errrr)

	}

	return &vm, nil
}

func (o *OrchestratorService) GetBaseModel(projectId string) (string, error) {
	llmTrainingJob, err := o.repo.GetLLMJobByID(projectId)
	if err != nil {
		log.Println(
			"[orchestrator] failed to job by id:",
			err,
		)
		return "", err
	}

	model, err := o.modelRepo.Get(llmTrainingJob.BaseModel)
	fmt.Printf("The details used for presigned download **basemodel filesha :%s for this userid %s", model.Sha256, llmTrainingJob.OwnerID)

	s3downloadUrlSubject := fmt.Sprintf("s3.task.create_presign_download_url")
	type CreatePresignDownloadURLRequest struct {
		UserID        string `json:"user_id"`
		CorrelationID string `json:"correlation_id"`
		FileSha256    string `json:"sha256"`
		AssetID       string `json:"asset_id"`
		FileCount     int    `json:"file_count"`
	}
	type CreatePresignDownloadURLResponse struct {
		URL      string `json:"url"`
		BucketID string `json:"bucket_id"`
		Sha256   string `json:"sha256"`

		FileCount int `json:"file_count"` // zip or tar
	}

	req := CreatePresignDownloadURLRequest{
		UserID:        llmTrainingJob.OwnerID,
		CorrelationID: uuid.New().String(),
		FileSha256:    model.Sha256,
		AssetID:       model.ID,
		FileCount:     1,
	}

	respBytes, err := o.nc.Publisher.Request(s3downloadUrlSubject, req)
	if err != nil {
		return "", fmt.Errorf("presign request failed: %w", err)
	}

	var presignResp CreatePresignDownloadURLResponse
	if err := json.Unmarshal(respBytes, &presignResp); err != nil {
		return "", fmt.Errorf("unmarshal presign resp: %w", err)
	}

	return presignResp.URL, nil

}

func (o *OrchestratorService) buildOrchestratorData(
	runID string,
	files []model.TrainingJobFile,
	baseModel string,
	projectId string,
) model.OrchestratorData {

	data := model.OrchestratorData{
		RunID: runID,
		BaseModel: model.Artifact{
			Name: "Base Model",
			Url:  fmt.Sprintf("%s%s", o.controlPlane, baseModel),
		},

		TrainingFiles:   []model.Artifact{},
		ValidationFiles: []model.Artifact{},
		CodeFiles:       []model.Artifact{},
	}
	data.ControlPlane = fmt.Sprintf("%s/api/v1/llm", o.controlPlane)

	job, err := o.repo.GetLLMJobByID(projectId)
	if err != nil {
		log.Printf("Error gettin job for project id: %s", projectId)
	}

	data.ApiKey = job.ApiKey

	for _, file := range files {
		req := model.CreatePresignDownloadURLRequest{
			UserID:        job.OwnerID,
			CorrelationID: uuid.New().String(),
			FileSha256:    file.Sha256,
			AssetID:       projectId,
			FileCount:     1,
		}

		log.Printf("the request sent is %+v", req)

		respBytes, err := o.nc.Publisher.Request("s3.task.create_presign_download_url", req)

		if err != nil {
			log.Printf("Error gettin Presigned download url for : %s", file.Name)

			return model.OrchestratorData{}
		}
		fmt.Printf("The details used for presigned download basemodel filesha :%s for this userid %s and this api-key %s", file.Sha256, job.OwnerID, job.ApiKey)

		var presignResp model.CreatePresignDownloadURLResponse
		if err := json.Unmarshal(respBytes, &presignResp); err != nil {
			log.Printf("Error unmarshalling Presigned download url for : %s", file.Name)
			return model.OrchestratorData{}

		}

		switch file.Role {

		case "training":

			// For now use the S3 path directly.
			// Later this can become a presigned URL.
			data.TrainingFiles = append(
				data.TrainingFiles,
				model.Artifact{
					Url:  fmt.Sprintf("%s%s", o.controlPlane, presignResp.URL),
					Name: file.Name,
				},
				// file.Path,
			)

		case "validation":

			data.ValidationFiles = append(
				data.ValidationFiles,
				model.Artifact{
					Url: fmt.Sprintf("%s%s", o.controlPlane, presignResp.URL),

					Name: file.Name,
				},
			)

		case "code":

			data.CodeFiles = append(
				data.CodeFiles,
				model.Artifact{
					Url: fmt.Sprintf("%s%s", o.controlPlane, presignResp.URL),

					Name: file.Name,
				},
			)
		}
	}

	return data
}

func (o *OrchestratorService) sendPingToVm(
	vmIP string, apiKey string, runId string, controlPlane string,
) error {
	url := fmt.Sprintf(
		"http://%s:9034/ping",
		vmIP,
	)

	log.Println(
		"[orchestrator] pinging vm before sending ingest",
		url,
	)

	type PingData struct {
		RunID        string `json:"run_id"`
		ApiKey       string `json:"api_key"`
		ControlPlane string `json:"control_plane"`
	}
	pingData := PingData{
		ApiKey:       apiKey,
		RunID:        runId,
		ControlPlane: controlPlane,
	}
	pyaload, err := json.Marshal(pingData)

	if err != nil {
		log.Println("Ping marshaller error")
		return err
	}
	req, err := http.NewRequest(
		http.MethodPost,
		url,

		bytes.NewReader(pyaload),
	)
if err != nil {
    return fmt.Errorf("failed to build ping request: %w", err)
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
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"ingest returned HTTP %d",
			resp.StatusCode,
		)
	}

	bodyResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read VM response: %w", err)
	}

	type PingResponse struct {
		Status string `json:"status"`
	}
	var pingResponse PingResponse
	if err := json.Unmarshal(bodyResp, &pingResponse); err != nil {
		return fmt.Errorf(
			"failed to parse VM deploy response: %w",
			err,
		)
	}

	fmt.Printf("\nthe ping reponse: %+v\n", pingResponse)
	if pingResponse.Status != "success" {
		return fmt.Errorf("Ping response didnot return succes ")
	}

	return nil
}

func (o *OrchestratorService) sendIngest(
	vmIP string,
	data model.OrchestratorData,
) error {

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"http://%s:9034/ingest",
		vmIP,
	)

	log.Println(
		"[orchestrator] sending ingest to:",
		url,
	)

	// We'll implement this using net/http.
	// The request should return quickly with 202
	// once the VM has accepted the ingest job.

	req, err := http.NewRequest(
		http.MethodPost,
		url,
		bytes.NewReader(payload),
	)

	if err != nil {
		return err
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
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"ingest returned HTTP %d",
			resp.StatusCode,
		)
	}

	return nil
}
