package service

import (
	model "llm-inference-service/internal/models"
	"llm-inference-service/internal/repository"
	"time"

	"github.com/google/uuid"
)

type TrainingService struct {
	repo *repository.PostgresRepository
}

func NewTrainingService(repo *repository.PostgresRepository) *TrainingService {
	return &TrainingService{repo: repo}
}

func (s *TrainingService) CreateJob(ownerID string, req model.TrainingJob) (model.TrainingJob, error) {

	job := model.TrainingJob{
		ID:         uuid.NewString(),
		OwnerID:    ownerID,
		Name:       req.Name,
		Status:     "Initializing",
		Instance:   req.Instance,
		InputPath:  req.InputPath,
		OutputPath: req.OutputPath,
		Progress:   0,
		CreatedAt:  time.Now(),
	}

	err := s.repo.Create(job)
	return job, err
}

func (s *TrainingService) GetAll(ownerID string) ([]model.TrainingJob, error) {
	return s.repo.GetAllByOwner(ownerID)
}

func (s *TrainingService) GetByID(jobID string, ownerID string) (model.TrainingJob, error) {
	return s.repo.GetByID(jobID, ownerID)
}