package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	model "llm-inference-service/internal/models"
	"log"
	"time"
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
	nodesJSON, err := json.Marshal(job.Nodes)
	if err != nil {
		return fmt.Errorf("marshal nodes: %w", err)
	}
	edgesJSON, err := json.Marshal(job.Edges)
	if err != nil {
		return fmt.Errorf("marshal edges: %w", err)
	}
	pipelineJSON, err := json.Marshal(job.Pipeline)
	if err != nil {
		return fmt.Errorf("marshal pipeline: %w", err)
	}
	tagsJSON, err := json.Marshal(job.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	query := `
		INSERT INTO training_jobs (
			id, owner_id, name, status, description,
			nodes, edges, pipeline, tags,
			progress, created_at, session_id, bucket_name
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`

	_, err = r.db.Exec(query,
		job.ID,
		job.OwnerID,
		job.Name,
		job.Status,
		job.Description,
		nodesJSON,
		edgesJSON,
		pipelineJSON,
		tagsJSON,
		job.Progress,
		job.CreatedAt,
		job.SessionID,
		job.BucketName,
	)

	return err
}
func (r *PostgresRepository) GetByID(jobID string, ownerID string) (model.TrainingJob, error) {
	query := `
		SELECT id, owner_id, name, status, description,
		       progress, created_at, session_id,
		       nodes, edges, pipeline, tags, bucket_name
		FROM training_jobs
		WHERE id = $1 AND owner_id = $2
	`

	var job model.TrainingJob
	var nodesJSON, edgesJSON, pipelineJSON, tagsJSON []byte

	err := r.db.QueryRow(query, jobID, ownerID).Scan(
		&job.ID, &job.OwnerID, &job.Name, &job.Status, &job.Description,
		&job.Progress, &job.CreatedAt, &job.SessionID,
		&nodesJSON, &edgesJSON, &pipelineJSON, &tagsJSON, &job.BucketName,
	)
	if err != nil {
		return job, err
	}

	json.Unmarshal(nodesJSON,    &job.Nodes)
	json.Unmarshal(edgesJSON,    &job.Edges)
	json.Unmarshal(pipelineJSON, &job.Pipeline)
	json.Unmarshal(tagsJSON,     &job.Tags)

	return job, nil
}
func (r *PostgresRepository) UpdateNodeStatus(jobID, nodeID, status string, startedAt time.Time, finishedAt *time.Time, errMsg string) error {
	// fetch current nodes
	var nodesJSON []byte
	err := r.db.QueryRow(`SELECT nodes FROM training_jobs WHERE id = $1`, jobID).Scan(&nodesJSON)
	if err != nil {
		return fmt.Errorf("fetch nodes: %w", err)
	}

	var nodes []model.PipelineNode
	if err := json.Unmarshal(nodesJSON, &nodes); err != nil {
		return fmt.Errorf("unmarshal nodes: %w", err)
	}

	// update target node
	for i := range nodes {
		if nodes[i].ID == nodeID {
			nodes[i].Status     = status
			nodes[i].StartedAt  = &startedAt
			nodes[i].FinishedAt = finishedAt
			nodes[i].Error      = errMsg
			break
		}
	}

	updated, err := json.Marshal(nodes)
	if err != nil {
		return fmt.Errorf("marshal nodes: %w", err)
	}

	_, err = r.db.Exec(`UPDATE training_jobs SET nodes = $1 WHERE id = $2`, updated, jobID)
	return err
}

func (r *PostgresRepository) UpdateJobStatus(jobID, status string) error {
	_, err := r.db.Exec(`UPDATE training_jobs SET status = $1 WHERE id = $2`, status, jobID)
	return err
}
func (r *PostgresRepository) UpdateJob(req model.TrainingJobDto) (model.TrainingJob, error) {
    nodesJSON, err := json.Marshal(req.Nodes)
    if err != nil {
        return model.TrainingJob{}, fmt.Errorf("marshal nodes: %w", err)
    }
    edgesJSON, err := json.Marshal(req.Edges)
    if err != nil {
        return model.TrainingJob{}, fmt.Errorf("marshal edges: %w", err)
    }
    pipelineJSON, err := json.Marshal(req.Pipeline)
    if err != nil {
        return model.TrainingJob{}, fmt.Errorf("marshal pipeline: %w", err)
    }
    tagsJSON, err := json.Marshal(req.Tags)
    if err != nil {
        return model.TrainingJob{}, fmt.Errorf("marshal tags: %w", err)
    }

    query := `
        UPDATE training_jobs SET
            name        = COALESCE(NULLIF($1, ''), name),
            description = COALESCE(NULLIF($2, ''), description),
            nodes       = $3,
            edges       = $4,
            pipeline    = $5,
            tags        = $6,
            status      = COALESCE(NULLIF($7, ''), status),
            progress    = $8
        WHERE id = $9
        RETURNING id, owner_id, name, description, nodes, edges, pipeline, tags, status, progress, created_at, session_id
    `

    var job model.TrainingJob
    var retNodes, retEdges, retPipeline, retTags []byte

    err = r.db.QueryRow(query,
        req.Name,
        req.Description,
        nodesJSON,
        edgesJSON,
        pipelineJSON,
        tagsJSON,
        req.Status,
        req.Progress,
        req.ID,
    ).Scan(
        &job.ID,
        &job.OwnerID,
        &job.Name,
        &job.Description,
        &retNodes,
        &retEdges,
        &retPipeline,
        &retTags,
        &job.Status,
        &job.Progress,
        &job.CreatedAt,
        &job.SessionID,
    )
    if err != nil {
        return model.TrainingJob{}, fmt.Errorf("update job: %w", err)
    }

    json.Unmarshal(retNodes,    &job.Nodes)
    json.Unmarshal(retEdges,    &job.Edges)
    json.Unmarshal(retPipeline, &job.Pipeline)
    json.Unmarshal(retTags,     &job.Tags)

    return job, nil
}
func (r *PostgresRepository) GetAllByOwner(ownerID string) ([]model.TrainingJob, error) {
	query := `
		SELECT id, owner_id, name, status, description,
		       session_id, progress, created_at,
		       nodes, edges, pipeline, tags
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
		var nodesJSON, edgesJSON, pipelineJSON, tagsJSON []byte

		if err := rows.Scan(
			&j.ID, &j.OwnerID, &j.Name, &j.Status, &j.Description,
			&j.SessionID, &j.Progress, &j.CreatedAt,
			&nodesJSON, &edgesJSON, &pipelineJSON, &tagsJSON,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(nodesJSON,    &j.Nodes)
		json.Unmarshal(edgesJSON,    &j.Edges)
		json.Unmarshal(pipelineJSON, &j.Pipeline)
		json.Unmarshal(tagsJSON,     &j.Tags)

		jobs = append(jobs, j)
	}
log.Printf("the jobs are: %v", jobs)
	return jobs, nil
}