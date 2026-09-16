package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	model "llm-inference-service/internal/domain/models"
	"log"
	"time"

	"github.com/google/uuid"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresTrainingRepo(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
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
			progress, created_at, session_id, bucket_name, bucket_id
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13, $14)
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
		job.BucketID,
	)

	return err
}
func (r *PostgresRepository) GetByID(jobID string, ownerID string) (model.TrainingJob, error) {
	query := `
		SELECT id, owner_id, name, status, description,
		       progress, created_at, session_id,
		       nodes, edges, pipeline, tags, bucket_name,bucket_id
		FROM training_jobs
		WHERE id = $1 AND owner_id = $2
	`

	var job model.TrainingJob
	var nodesJSON, edgesJSON, pipelineJSON, tagsJSON []byte

	err := r.db.QueryRow(query, jobID, ownerID).Scan(
		&job.ID, &job.OwnerID, &job.Name, &job.Status, &job.Description,
		&job.Progress, &job.CreatedAt, &job.SessionID,
		&nodesJSON, &edgesJSON, &pipelineJSON, &tagsJSON, &job.BucketName, &job.BucketID,
	)
	if err != nil {
		return job, err
	}

	json.Unmarshal(nodesJSON, &job.Nodes)
	json.Unmarshal(edgesJSON, &job.Edges)
	json.Unmarshal(pipelineJSON, &job.Pipeline)
	json.Unmarshal(tagsJSON, &job.Tags)

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
			nodes[i].Status = status
			nodes[i].StartedAt = &startedAt
			nodes[i].FinishedAt = finishedAt
			nodes[i].Error = errMsg
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

	json.Unmarshal(retNodes, &job.Nodes)
	json.Unmarshal(retEdges, &job.Edges)
	json.Unmarshal(retPipeline, &job.Pipeline)
	json.Unmarshal(retTags, &job.Tags)

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

		json.Unmarshal(nodesJSON, &j.Nodes)
		json.Unmarshal(edgesJSON, &j.Edges)
		json.Unmarshal(pipelineJSON, &j.Pipeline)
		json.Unmarshal(tagsJSON, &j.Tags)

		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (r *PostgresRepository) RemoveFile(fileId, jobID string,) error {
	fileQuery :=`delete from llm_training_job_files where id=$1 and project_id=$2`
		_, err := r.db.Exec(fileQuery,
			fileId,jobID,
		)
		if err != nil {
			return fmt.Errorf("delete llm_training_job_files (%s): %w", fileId, err)
		}

		return nil

}
func (r *PostgresRepository) AddFileToLLMJob(jobID string, files []model.LLMTrainingJobFile) error {
	now := time.Now()

	fileQuery := `
		INSERT INTO llm_training_job_files (
			id, project_id, name, role, size, type, path, created_at, sha256
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	for _, f := range files {
		fileCreatedAt := f.CreatedAt
		if fileCreatedAt.IsZero() {
			fileCreatedAt = now
		}
		_, err := r.db.Exec(fileQuery,
			f.ID, jobID, f.Name, f.Role, f.Size, f.Type, f.Path, fileCreatedAt, f.Sha256,
		)
		if err != nil {
			return fmt.Errorf("insert llm_training_job_files (%s): %w", f.Name, err)
		}
	}

	return nil
}
func (r *PostgresRepository) CreateLLMJob(job model.LLMTrainingJob, files []model.LLMTrainingJobFile) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	pipelineJSON, err := json.Marshal(job.Pipeline)
	if err != nil {
		return fmt.Errorf("marshal pipeline: %w", err)
	}
	tagsJSON, err := json.Marshal(job.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	nodesJSON, err := json.Marshal(job.Nodes)
	if err != nil {
		return fmt.Errorf("marshal nodes: %w", err)
	}
	edgesJSON, err := json.Marshal(job.Edges)
	if err != nil {
		return fmt.Errorf("marshal edges: %w", err)
	}

	jobQuery := `
		INSERT INTO llm_training_jobs (
			id, name, description, type, base_model,
			training_data_source, training_data_url, training_data_bucket,
			use_validation_data, training_mode, entry_point, compute,
			schedule, pipeline, tags, status, owner_id, nodes, edges,
			progress, session_id, bucket_name, bucket_id, created_at, updated_at, api_key, callback_url
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27
		)
	`

	now := time.Now()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = now
	}

	_, err = tx.Exec(jobQuery,
		job.ID, job.Name, job.Description, job.Type, job.BaseModel,
		job.TrainingDataSource, job.TrainingDataURL, job.TrainingDataBucket,
		job.UseValidationData, job.TrainingMode, job.EntryPoint, job.Compute,
		job.Schedule, pipelineJSON, tagsJSON, job.Status, job.OwnerID, nodesJSON, edgesJSON,
		job.Progress, job.SessionID, job.BucketName, job.BucketID, job.CreatedAt, job.UpdatedAt, job.ApiKey, job.CallBack,
	)
	if err != nil {
		return fmt.Errorf("insert llm_training_jobs: %w", err)
	}

	fileQuery := `
		INSERT INTO llm_training_job_files (
			id, project_id, name, role, size, type, path, created_at, sha256
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	for _, f := range files {
		fileCreatedAt := f.CreatedAt
		if fileCreatedAt.IsZero() {
			fileCreatedAt = now
		}
		_, err := tx.Exec(fileQuery,
			f.ID, job.ID, f.Name, f.Role, f.Size, f.Type, f.Path, fileCreatedAt, f.Sha256,
		)
		if err != nil {
			return fmt.Errorf("insert llm_training_job_files (%s): %w", f.Name, err)
		}
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetAllLLMJobs(ownerID string) ([]model.LLMTrainingJob, error) {
	query := `
		SELECT id, name, description, type, base_model,
		       training_data_source, training_data_url, training_data_bucket,
		       use_validation_data, training_mode, entry_point, compute,
		       schedule, pipeline, tags, status, owner_id,
		       progress, session_id, bucket_name, bucket_id, created_at, updated_at
		FROM llm_training_jobs
	`
	var args []interface{}
	if ownerID != "" {
		query += ` WHERE owner_id = $1 OR owner_id = ''`
		args = append(args, ownerID)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query llm_training_jobs: %w", err)
	}
	defer rows.Close()

	var jobs []model.LLMTrainingJob
	for rows.Next() {
		var j model.LLMTrainingJob
		var pipelineJSON, tagsJSON []byte
		if err := rows.Scan(
			&j.ID, &j.Name, &j.Description, &j.Type, &j.BaseModel,
			&j.TrainingDataSource, &j.TrainingDataURL, &j.TrainingDataBucket,
			&j.UseValidationData, &j.TrainingMode, &j.EntryPoint, &j.Compute,
			&j.Schedule, &pipelineJSON, &tagsJSON, &j.Status, &j.OwnerID,
			&j.Progress, &j.SessionID, &j.BucketName, &j.BucketID, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan llm_training_jobs: %w", err)
		}
		json.Unmarshal(pipelineJSON, &j.Pipeline)
		json.Unmarshal(tagsJSON, &j.Tags)
		if j.Pipeline == nil {
			j.Pipeline = []string{}
		}
		if j.Tags == nil {
			j.Tags = []string{}
		}
		j.Runs = []interface{}{}
		jobs = append(jobs, j)
	}

	if jobs == nil {
		jobs = []model.LLMTrainingJob{}
	}
	return jobs, nil
}

// select distinct vm_id  from pipeline_jobs where status='failed';
func (r *PostgresRepository) FailedJobs() ([]string, error) {
	query := `select distinct vm_id  from pipeline_jobs where status='failed' or status='running--'`
		var vmIds []string

	rows, err := r.db.Query(query)

	if err == nil {
		defer rows.Close()
		// var files []model.LLMTrainingJobFile

		for rows.Next() {
			var vmId string
			if err := rows.Scan(&vmId); err == nil {
				vmIds = append(vmIds, vmId)
			}
		}

	} else {
		vmIds = []string{}
	}

	return vmIds, nil
}

func (r *PostgresRepository) GetLLMJobByID(id string) (model.LLMTrainingJob, error) {
	query := `
		SELECT id, name, description, type, base_model,
		       training_data_source, training_data_url, training_data_bucket,
		       use_validation_data, training_mode, entry_point, compute,
		       schedule, pipeline, tags, status, owner_id,
		       progress, session_id, bucket_name, bucket_id, created_at, updated_at,
		       schedule_interval_seconds, next_run_at, last_run_at, schedule_enabled,api_key, callback_url
		FROM llm_training_jobs
		WHERE id = $1
	`
	var j model.LLMTrainingJob
	var pipelineJSON, tagsJSON []byte
	var intervalSec sql.NullInt64
	var nextRunAt, lastRunAt sql.NullTime
	var scheduleEnabled sql.NullBool

	err := r.db.QueryRow(query, id).Scan(
		&j.ID, &j.Name, &j.Description, &j.Type, &j.BaseModel,
		&j.TrainingDataSource, &j.TrainingDataURL, &j.TrainingDataBucket,
		&j.UseValidationData, &j.TrainingMode, &j.EntryPoint, &j.Compute,
		&j.Schedule, &pipelineJSON, &tagsJSON, &j.Status, &j.OwnerID,
		&j.Progress, &j.SessionID, &j.BucketName, &j.BucketID, &j.CreatedAt, &j.UpdatedAt,
		&intervalSec, &nextRunAt, &lastRunAt, &scheduleEnabled, &j.ApiKey, &j.CallBack,
	)
	if err != nil {
		return j, err
	}
	if intervalSec.Valid {
		j.ScheduleIntervalSeconds = &intervalSec.Int64
	}
	if nextRunAt.Valid {
		j.NextRunAt = &nextRunAt.Time
	}
	if lastRunAt.Valid {
		j.LastRunAt = &lastRunAt.Time
	}
	if scheduleEnabled.Valid {
		j.ScheduleEnabled = scheduleEnabled.Bool
	} else {
		j.ScheduleEnabled = true
	}

	json.Unmarshal(pipelineJSON, &j.Pipeline)
	json.Unmarshal(tagsJSON, &j.Tags)
	if j.Pipeline == nil {
		j.Pipeline = []string{}
	}
	if j.Tags == nil {
		j.Tags = []string{}
	}
	j.Runs = []interface{}{}

	filesQuery := `
		SELECT id, project_id, name, role, size, type, path, created_at
		FROM llm_training_job_files
		WHERE project_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(filesQuery, id)
	if err == nil {
		defer rows.Close()
		var files []model.LLMTrainingJobFile
		for rows.Next() {
			var f model.LLMTrainingJobFile
			if err := rows.Scan(&f.ID, &f.ProjectID, &f.Name, &f.Role, &f.Size, &f.Type, &f.Path, &f.CreatedAt); err == nil {
				files = append(files, f)
			}
		}
		if files == nil {
			files = []model.LLMTrainingJobFile{}
		}
		j.Files = files
	} else {
		j.Files = []model.LLMTrainingJobFile{}
	}

	return j, nil
}

func (r *PostgresRepository) AddTrainingJob(id string, req model.AddTrainingJobRequest,
) (model.TrainingQueue, error) {

	query := `
		INSERT INTO pipeline_runs (
			id,
			project_id,
			status
		)
		VALUES ($1, $2, $3)
		RETURNING id, status
	`

	var queue model.TrainingQueue

	err := r.db.QueryRow(
		query,
		id,
		req.ProjectID,
		req.Status,
	).Scan(
		&queue.RunID,
		&queue.Status,
	)

	if err != nil {
		return model.TrainingQueue{}, err
	}

	return queue, nil
}

func (r *PostgresRepository) UpdateLLMJob(id string, req model.LLMTrainingJob) (model.LLMTrainingJob, error) {
	var pipelineJSON, tagsJSON []byte
	var err error

	if req.Pipeline != nil {
		pipelineJSON, err = json.Marshal(req.Pipeline)
		if err != nil {
			return model.LLMTrainingJob{}, fmt.Errorf("marshal pipeline: %w", err)
		}
	}
	if req.Tags != nil {
		tagsJSON, err = json.Marshal(req.Tags)
		if err != nil {
			return model.LLMTrainingJob{}, fmt.Errorf("marshal tags: %w", err)
		}
	}

	query := `
		UPDATE llm_training_jobs SET
			name        = COALESCE(NULLIF($1, ''), name),
			description = COALESCE(NULLIF($2, ''), description),
			schedule    = COALESCE(NULLIF($3, ''), schedule),
			status      = COALESCE(NULLIF($4, ''), status),
			base_model  = COALESCE(NULLIF($5, ''), base_model),
			compute     = COALESCE(NULLIF($6, ''), compute),
			pipeline    = CASE WHEN $7::jsonb IS NOT NULL THEN $7::jsonb ELSE pipeline END,
			tags        = CASE WHEN $8::jsonb IS NOT NULL THEN $8::jsonb ELSE tags END,
			api_key     = COALESCE(NULLIF($9, ''), api_key),
			entry_point     = COALESCE(NULLIF($10, ''), entry_point),

			updated_at  = NOW()
		WHERE id = $11
	`
	_, err = r.db.Exec(
		query,
		req.Name,
		req.Description,
		req.Schedule,
		req.Status,
		req.BaseModel,
		req.Compute,
		pipelineJSON,
		tagsJSON,
		req.ApiKey,
		req.EntryPoint,
		id,
	)
	if err != nil {
		return model.LLMTrainingJob{}, fmt.Errorf("update llm_training_jobs: %w", err)
	}

	return r.GetLLMJobByID(id)
}

func (r *PostgresRepository) DeleteLLMJob(id string) error {
	_, err := r.db.Exec(`DELETE FROM llm_training_jobs WHERE id = $1`, id)
	return err
}

// Orchestratormethods

func (r *PostgresRepository) CountRunningPipelines() (int, error) {

	query := `
		SELECT COUNT(*)
		FROM pipeline_runs
		WHERE status = 'running'
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return 0, err
		}
	}

	return count, rows.Err()
}
func (r *PostgresRepository) GetQueuedRun() (*model.QueuedRun, error) {

	query := `
		SELECT
			pr.id,
			pr.project_id,
			pr.status,
			ltj.pipeline
		FROM pipeline_runs pr
		INNER JOIN llm_training_jobs ltj
			ON ltj.id = pr.project_id
		WHERE pr.status = 'queued'
		ORDER BY pr.created_at ASC
		LIMIT 1
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, rows.Err()
	}

	var run model.QueuedRun
	var pipelineRaw []byte

	if err := rows.Scan(
		&run.ID,
		&run.ProjectID,
		&run.Status,
		&pipelineRaw,
	); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(pipelineRaw, &run.Pipeline); err != nil {
		return nil, err
	}

	return &run, nil
}

func (r *PostgresRepository) AddPipelineJob(
	req model.AddPipelineJobRequest,
) (model.PipelineJob, error) {

	jobID := fmt.Sprintf("job-%d", time.Now().UnixNano())

	query := `
		INSERT INTO pipeline_jobs (
			id,
			run_id,
			step,
			status,
			project_id
		)
		VALUES ($1, $2, $3, $4,$5)
		RETURNING id, run_id, step, status,project_id
	`

	rows, err := r.db.Query(
		query,
		jobID,
		req.RunID,
		req.Step,
		req.Status,
		req.ProjectID,
	)
	if err != nil {
		return model.PipelineJob{}, err
	}
	defer rows.Close()

	var job model.PipelineJob
	if rows.Next() {
		if err := rows.Scan(
			&job.ID,
			&job.RunID,
			&job.Step,
			&job.Status,
			&job.ProjectID,
		); err != nil {
			return model.PipelineJob{}, err
		}
	}

	return job, rows.Err()
}

func (r *PostgresRepository) AddPipelineJobComplete(
	req model.AddPipelineJobRequest,
) (model.PipelineJob, error) {

	jobID := fmt.Sprintf("job-%d", time.Now().UnixNano())

	query := `
		INSERT INTO pipeline_jobs (
			id,
			run_id,
			step,
			status,
			vm_id,
			vm_ip,
			project_id
		)
		VALUES ($1, $2, $3, $4, $5,$6,$7)
		RETURNING id, run_id, step, status, vm_id, vm_ip, project_id
	`

	rows, err := r.db.Query(
		query,
		jobID,
		req.RunID,
		req.Step,
		req.Status,
		req.VMID,
		req.VMIP,
		req.ProjectID,
	)
	if err != nil {
		return model.PipelineJob{}, err
	}
	defer rows.Close()

	var job model.PipelineJob
	if rows.Next() {
		if err := rows.Scan(
			&job.ID,
			&job.RunID,
			&job.Step,
			&job.Status,
			&job.VMID,
			&job.VMIP,
			&job.ProjectID,
		); err != nil {
			return model.PipelineJob{}, err
		}
	}

	return job, rows.Err()
}

func (r *PostgresRepository) GetProjectFiles(projectID string) ([]model.TrainingJobFile, error) {

	query := `
		SELECT
			id,
			project_id,
			name,
			role,
			size,
			type,
			path,
			sha256
		FROM llm_training_job_files
		WHERE project_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []model.TrainingJobFile
	for rows.Next() {
		var f model.TrainingJobFile
		if err := rows.Scan(
			&f.ID,
			&f.ProjectID,
			&f.Name,
			&f.Role,
			&f.Size,
			&f.Type,
			&f.Path,
			&f.Sha256,
		); err != nil {
			return nil, err
		}
		files = append(files, f)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return files, nil
}

func (r *PostgresRepository) MarkPipelineRunning(runID string) error {

	query := `
		UPDATE pipeline_runs
		SET
			status = 'running',
			started_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
		  AND status = 'queued'
	`

	result, err := r.db.Exec(query, runID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *PostgresRepository) UpdatePipelineJobVM(
	jobID string,
	vmID string,
	vmIP string,
) error {

	query := `
		UPDATE pipeline_jobs
		SET
			vm_id = $1,
			vm_ip = $2,
			status = 'running',
			started_at = NOW(),
			updated_at = NOW()
		WHERE id = $3
	`

	_, err := r.db.Exec(
		query,
		vmID,
		vmIP,
		jobID,
	)

	return err
}

// VM status methods

func (r *PostgresRepository) GetPipelineJobByRunAndStep(
	runID string,
	step string,
) (*model.PipelineJob, error) {

	query := `
		SELECT
			id,
			run_id,
			step,
			status,
			vm_id,
			vm_ip,
			project_id

		FROM pipeline_jobs
		WHERE run_id = $1
		  AND step = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	var job model.PipelineJob

	err := r.db.QueryRow(
		query,
		runID,
		step,
	).Scan(
		&job.ID,
		&job.RunID,
		&job.Step,
		&job.Status,
		&job.VMID,
		&job.VMIP,
		&job.ProjectID,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &job, nil
}

func (r *PostgresRepository) MarkPipelineJobRunning(
	jobID string,
) error {

	query := `
		UPDATE pipeline_jobs
		SET
			status = 'running',
			started_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(
		query,
		jobID,
	)

	return err
}

func (r *PostgresRepository) MarkPipelineFailed(
	runID string,
) error {

	query := `
		UPDATE pipeline_runs
		SET
			status = 'failed',
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(
		query,
		runID,
	)

	return err
}

func (r *PostgresRepository) MarkPipelineCompleted(
	runID string,
) error {

	query := `
		UPDATE pipeline_runs
		SET
			status = 'completed',
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(
		query,
		runID,
	)

	return err
}

func (r *PostgresRepository) MarkPipelineJobFailed(
	jobID string,
	duration float64,
) error {

	query := `
		UPDATE pipeline_jobs
		SET
			status = 'failed',
			completed_at = NOW(),
			duration_seconds = $1,
			updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(
		query,
		duration,
		jobID,
	)

	return err
}

func (r *PostgresRepository) MarkPipelineJobCompleted(
	jobID string,
	duration float64,
) error {
	log.Printf("Marking job as completed: %s", jobID)

	query := `
		UPDATE pipeline_jobs
		SET
			status = 'completed',
			completed_at = NOW(),
			duration_seconds = $1,
			updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(
		query,
		duration,
		jobID,
	)

	return err
}

func (r *PostgresRepository) GetLLMProjectByRunID(
	runID string,
) (*model.LLMTrainingJob, error) {
	fmt.Printf("the run id# %s\n", runID)
	query := `
		SELECT
			
			p.entry_point,
			p.api_key,
			p.callback_url

		FROM llm_training_jobs p
		INNER JOIN pipeline_runs r
			ON r.project_id = p.id
		WHERE r.id = $1
		LIMIT 1
	`

	var project model.LLMTrainingJob

	err := r.db.QueryRow(
		query,
		runID,
	).Scan(

		&project.EntryPoint,
		&project.ApiKey,
		&project.CallBack,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (r *PostgresRepository) UpdateJobScheduleInfo(jobID string, intervalSeconds int64, nextRunAt time.Time, lastRunAt time.Time) error {
	query := `
		UPDATE llm_training_jobs
		SET schedule_interval_seconds = $1,
		    next_run_at = $2,
		    last_run_at = $3,
		    updated_at = NOW()
		WHERE id = $4
	`
	_, err := r.db.Exec(query, intervalSeconds, nextRunAt, lastRunAt, jobID)
	return err
}

func (r *PostgresRepository) EnqueuePipelineRun(projectID string) (string, error) {
	runID := uuid.New().String()
	query := `
		INSERT INTO pipeline_runs (
			id,
			project_id,
			status,
			created_at,
			updated_at
		)
		VALUES ($1, $2, 'queued', NOW(), NOW())
		RETURNING id
	`
	var returnedID string
	err := r.db.QueryRow(query, runID, projectID).Scan(&returnedID)
	if err != nil {
		return "", err
	}
	return returnedID, nil
}

func (r *PostgresRepository) GetDueScheduledJobs() ([]model.LLMTrainingJob, error) {
	query := `
		SELECT id, name, schedule, schedule_interval_seconds, next_run_at, last_run_at
		FROM llm_training_jobs
		WHERE schedule_enabled = true
		  AND schedule != 'manual'
		  AND schedule != '0'
		  AND schedule != ''
		  AND next_run_at IS NOT NULL
		  AND next_run_at <= NOW()
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.LLMTrainingJob
	for rows.Next() {
		var j model.LLMTrainingJob
		var intervalSec sql.NullInt64
		var nextRunAt, lastRunAt sql.NullTime
		if err := rows.Scan(&j.ID, &j.Name, &j.Schedule, &intervalSec, &nextRunAt, &lastRunAt); err != nil {
			return nil, err
		}
		if intervalSec.Valid {
			j.ScheduleIntervalSeconds = &intervalSec.Int64
		}
		if nextRunAt.Valid {
			j.NextRunAt = &nextRunAt.Time
		}
		if lastRunAt.Valid {
			j.LastRunAt = &lastRunAt.Time
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (r *PostgresRepository) HasActiveOrQueuedRun(projectID string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM pipeline_runs
		WHERE project_id = $1
		  AND status IN ('queued', 'running')
	`
	var count int
	err := r.db.QueryRow(query, projectID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
