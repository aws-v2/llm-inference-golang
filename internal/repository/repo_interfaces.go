package repository

import (
	model "llm-inference-service/internal/domain/models"
	"time"
)

type TrainingJobFilesRepository interface {
	AddFileToLLMJob(jobID string, files []model.LLMTrainingJobFile) error
	GetProjectFiles(projectID string) ([]model.TrainingJobFile, error)
}


// type ModelRepository interface {
// 	Save(m model.TrainingJob) error
// 	Get(id string) (model.TrainingJob, error)
// 	GetByOwner(ownerID string) ([]model.TrainingJob, error)
// 	UpdateStatus(id string, status model.Status) error
// }


type OrchestratorRepository interface {
 CountRunningPipelines() (int, error) 
 GetQueuedRun() (*model.QueuedRun, error) 
 MarkPipelineRunning(runID string) error 
 AddPipelineJob(req model.AddPipelineJobRequest,) (model.PipelineJob, error) 
 GetProjectFiles(projectID string,) ([]model.TrainingJobFile, error)
 GetLLMJobByID(id string) (model.LLMTrainingJob, error)
 UpdateJobScheduleInfo(jobID string, intervalSeconds int64, nextRunAt time.Time, lastRunAt time.Time) error
 HasActiveOrQueuedRun(projectID string) (bool, error)
 EnqueuePipelineRun(projectID string) (string, error)
 GetDueScheduledJobs() ([]model.LLMTrainingJob, error)
}


/*
	llmTrainingJob, err := o.repo.GetLLMJobByID(projectId)
	model, err := o.modelRepo.Get(llmTrainingJob.BaseModel)


*/