package repository

import (
	"database/sql"
	model "llm-inference-service/internal/models"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresTrainingRepo(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

type Repository interface {
	Save(m model.TrainingJob) error
	Get(id string) (model.TrainingJob, error)
	GetByOwner(ownerID string) ([]model.TrainingJob, error)
	UpdateStatus(id string, status model.Status) error
}

func (r *PostgresRepository) Create(job model.TrainingJob) error {
	query := `
		INSERT INTO training_jobs (
			id, owner_id, name, status, instance,
			input_path, output_path, progress, created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`

	_, err := r.db.Exec(
		query,
		job.ID,
		job.OwnerID,
		job.Name,
		job.Status,
		job.Instance,
		job.InputPath,
		job.OutputPath,
		job.Progress,
		job.CreatedAt,
	)

	return err
}


func (r *PostgresRepository) GetByID(jobID string, ownerID string) (model.TrainingJob, error) {

	query := `
		SELECT id, owner_id, name, status, instance,
		       input_path, output_path, progress, created_at
		FROM training_jobs
		WHERE id = $1 AND owner_id = $2
	`

	var job model.TrainingJob

	err := r.db.QueryRow(query, jobID, ownerID).Scan(
		&job.ID,
		&job.OwnerID,
		&job.Name,
		&job.Status,
		&job.Instance,
		&job.InputPath,
		&job.OutputPath,
		&job.Progress,
		&job.CreatedAt,
	)

	return job, err
}


func (r *PostgresRepository) GetAllByOwner(ownerID string) ([]model.TrainingJob, error) {

	query := `
		SELECT id, owner_id, name, status, instance,
		       input_path, output_path, progress, created_at
		FROM training_jobs
		WHERE owner_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.TrainingJob

	for rows.Next() {
		var j model.TrainingJob

		if err := rows.Scan(
			&j.ID,
			&j.OwnerID,
			&j.Name,
			&j.Status,
			&j.Instance,
			&j.InputPath,
			&j.OutputPath,
			&j.Progress,
			&j.CreatedAt,
		); err != nil {
			return nil, err
		}

		jobs = append(jobs, j)
	}

	return jobs, nil
}
 