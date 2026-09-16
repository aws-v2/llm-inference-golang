CREATE TABLE IF NOT EXISTS llm_training_jobs (
    id VARCHAR(128) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    type VARCHAR(64) DEFAULT 'llm',
    base_model VARCHAR(128) DEFAULT '',
    training_data_source VARCHAR(64) DEFAULT '',
    training_data_url TEXT DEFAULT '',
    training_data_bucket VARCHAR(255) DEFAULT '',
    use_validation_data BOOLEAN DEFAULT FALSE,
    training_mode VARCHAR(64) DEFAULT 'serwin',
    entry_point VARCHAR(255) DEFAULT '',
    compute VARCHAR(64) DEFAULT '',
    schedule VARCHAR(64) DEFAULT 'manual',
    pipeline JSONB DEFAULT '[]',
    tags JSONB DEFAULT '[]',
    status VARCHAR(64) DEFAULT 'idle',
    owner_id VARCHAR(128) DEFAULT '',
    nodes JSONB DEFAULT '[]',
    edges JSONB DEFAULT '[]',
    progress DOUBLE PRECISION DEFAULT 0,
    session_id TEXT DEFAULT '',
    bucket_name TEXT DEFAULT '',
    bucket_id TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS llm_training_job_files (
    id VARCHAR(128) PRIMARY KEY,
    project_id VARCHAR(128) NOT NULL REFERENCES llm_training_jobs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    sha256 VARCHAR(255) NOT NULL default '',
    role VARCHAR(64) NOT NULL,
    size BIGINT DEFAULT 0,
    type VARCHAR(128) DEFAULT '',
    path TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_llm_training_job_files_project_id ON llm_training_job_files(project_id);







CREATE TABLE pipeline_runs (
    id UUID PRIMARY KEY,
    project_id VARCHAR(128) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'queued',

    started_at TIMESTAMP WITH TIME ZONE,
    current_step VARCHAR(30) not null default '',
    completed_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_pipeline_runs_project
        FOREIGN KEY (project_id)
        REFERENCES llm_training_jobs(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_pipeline_runs_project_id
    ON pipeline_runs(project_id);

CREATE INDEX idx_pipeline_runs_status
    ON pipeline_runs(status);




CREATE TABLE pipeline_jobs (
    id VARCHAR(100) PRIMARY KEY,
    run_id UUID NOT NULL,

    step VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'queued',

    vm_id VARCHAR(100),
    vm_ip VARCHAR(100),

    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_pipeline_jobs_run
        FOREIGN KEY (run_id)
        REFERENCES pipeline_runs(id)
        ON DELETE CASCADE
);
CREATE INDEX idx_pipeline_jobs_run_id
    ON pipeline_jobs(run_id);

CREATE INDEX idx_pipeline_jobs_status
    ON pipeline_jobs(status);




ALTER TABLE pipeline_jobs
ADD COLUMN  if not exists duration_seconds DOUBLE PRECISION;




--tinyyllam aa54a5fb99ace5b964859cf072346631b2da6109715a805d07161d157c66ce7f
-- if userid==00000, mark the base model as public ie. for everyone

-- create table base_models(
--     id ,
--     name, 
--     sha256
--     owner_id

-- );


-- select * from base_models where uploader_id == uploader_id or uploader_id=0000000 




ALTER TABLE llm_training_jobs
ADD COLUMN if not exists schedule_interval_seconds BIGINT,
ADD COLUMN if not exists next_run_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN if not exists last_run_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN if not exists schedule_enabled BOOLEAN NOT NULL DEFAULT TRUE;




alter table pipeline_jobs add column  if not exists  project_id VARCHAR(100) ; 

alter table llm_training_jobs add column if not exists  api_key text not null default '';




alter table llm_training_jobs add column if not exists  callback_url text;