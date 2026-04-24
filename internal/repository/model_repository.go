package repository

import (
	"database/sql"
	"fmt"
	model "llm-inference-service/internal/models"
	"time"

	"github.com/google/uuid"
)

type ModelRepository interface {
	Save(m model.Model) error
	Get(id string) (model.Model, error)
	GetByOwner(ownerID string) ([]model.Model, error)
	UpdateStatus(id string, status model.Status) error
	DeployModel(modelID string, ownerID string) (model.Model, error)
}

type PostgresModelRepository struct {
	db *sql.DB
}

func NewPostgresModelRepository(db *sql.DB) *PostgresModelRepository {
	return &PostgresModelRepository{db: db}
}

func (r *PostgresModelRepository) Save(m model.Model) error {
	query := `
		INSERT INTO models (
			id, name, file_path, owner_id, status,
			temperature, max_tokens, created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`
	_, err := r.db.Exec(
		query,
		m.ID,
		m.Name,
		m.FilePath,
		m.OwnerID,
		m.Status,
		m.Temperature,
		m.MaxTokens,
		m.CreatedAt,
	)
	return err
}

func (r *PostgresModelRepository) Get(id string) (model.Model, error) {
	query := `
		SELECT id, name, file_path, owner_id, status,
		       temperature, max_tokens, created_at
		FROM models
		WHERE id = $1
	`
	var m model.Model
	err := r.db.QueryRow(query, id).Scan(
		&m.ID,
		&m.Name,
		&m.FilePath,
		&m.OwnerID,
		&m.Status,
		&m.Temperature,
		&m.MaxTokens,
		&m.CreatedAt,
	)
	return m, err
}

func (r *PostgresModelRepository) GetByOwner(ownerID string) ([]model.Model, error) {
	query := `
		SELECT id, name, file_path, owner_id, status,
		       temperature, max_tokens, created_at
		FROM models
		WHERE owner_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []model.Model
	for rows.Next() {
		var m model.Model
		if err := rows.Scan(
			&m.ID,
			&m.Name,
			&m.FilePath,
			&m.OwnerID,
			&m.Status,
			&m.Temperature,
			&m.MaxTokens,
			&m.CreatedAt,
		); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

func (r *PostgresModelRepository) UpdateStatus(id string, status model.Status) error {
	query := `
		UPDATE models
		SET status = $1
		WHERE id = $2
	`
	_, err := r.db.Exec(query, status, id)
	return err
}


func (r *PostgresModelRepository) DeployModel(modelID string, ownerID string) (model.Model, error) {

    // validate before touching the DB
    if modelID == "" {
        return model.Model{}, fmt.Errorf("modelID is required")
    }
    if ownerID == "" {
        return model.Model{}, fmt.Errorf("ownerID is required")
    }

    tx, err := r.db.Begin()
	if err != nil {
		return model.Model{}, err
	}
	defer tx.Rollback()

	// 1. Get training job
	var job model.TrainingJob

	queryJob := `
		SELECT id, name, owner_id, status, input_path, output_path
		FROM training_jobs
		WHERE id = $1 AND owner_id = $2
	`

	err = tx.QueryRow(queryJob, modelID, ownerID).Scan(
		&job.ID,
		&job.Name,
		&job.OwnerID,
		&job.Status,
		&job.InputPath,
		&job.OutputPath,
	)

	if err != nil {
		return model.Model{}, err
	}

	// 2. Must be completed
	if job.Status != "Completed" {
		return model.Model{}, fmt.Errorf("job not completed")
	}

	// 3. Create model from job
	newModel := model.Model{
		ID:         uuid.NewString(),
		Name:       job.Name + "-deployed",
		FilePath:   job.OutputPath,
		OwnerID:    ownerID,
		Status:     "Ready",
		CreatedAt:  time.Now(),
	}

	queryModel := `
		INSERT INTO models (id, name, file_path, owner_id, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)
	`

	_, err = tx.Exec(
		queryModel,
		newModel.ID,
		newModel.Name,
		newModel.FilePath,
		newModel.OwnerID,
		newModel.Status,
		newModel.CreatedAt,
	)

	if err != nil {
		return model.Model{}, err
	}

	// 4. mark training job as deployed
	_, err = tx.Exec(`
		UPDATE training_jobs
		SET status = 'Deployed'
		WHERE id = $1
	`, modelID)

	if err != nil {
		return model.Model{}, err
	}

	err = tx.Commit()
	if err != nil {
		return model.Model{}, err
	}

	return newModel, nil
}