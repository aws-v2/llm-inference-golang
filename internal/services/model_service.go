package service

import (
	"errors"
	model "llm-inference-service/internal/models"
	"llm-inference-service/internal/repository"
	"log"
	"time"

	"github.com/google/uuid"
)

type ModelService struct {
	store repository.ModelRepository // depend on interface, not concrete type
}

func NewModelService(store repository.ModelRepository) *ModelService {
	return &ModelService{store: store}
}

func (s *ModelService) Register(name string, ownerID string) (model.Model, error) {
	m := model.Model{
		ID:        uuid.New().String(),
		Name:      name,
		OwnerID:   ownerID,
		Status:    model.StatusPendingUpload,
		CreatedAt: time.Now(),
	}
	if err := s.store.Save(m); err != nil {
		return model.Model{}, err
	}
	return m, nil
}

func (s *ModelService) GetByOwner(ownerID string) ([]model.Model, error) {
	return s.store.GetByOwner(ownerID)
}

func (s *ModelService) GetByID(id, ownerID string) (model.Model, error) {
	m, err := s.store.Get(id)
	if err != nil {
		return model.Model{}, errors.New("model not found")
	}
	if m.OwnerID != ownerID {
		return model.Model{}, errors.New("model not found")
	}
	return m, nil
}

func (s *ModelService) UpdateConfig(id, ownerID string, temp float64, maxTokens int) error {
	m, err := s.store.Get(id)
	if err != nil || m.OwnerID != ownerID {
		return errors.New("model not found")
	}

	m.Temperature = temp
	m.MaxTokens = maxTokens

	return s.store.Save(m)
}


func (s *ModelService) Deploy(modelID string, ownerID string) (model.Model, error) {
	log.Println("Deploying model", modelID, "by owner", ownerID)
	return s.store.DeployModel(modelID, ownerID)
}
