CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


CREATE TABLE training_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id UUID NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    instance TEXT NOT NULL,
    input_path TEXT NOT NULL,
    output_path TEXT NOT NULL,
    progress INT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);


CREATE TABLE models (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id UUID NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    file_path TEXT NOT NULL,
    temperature FLOAT NOT NULL,
    max_tokens INT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
    

    ALTER TABLE training_jobs DROP COLUMN IF EXISTS instance; 
    ALTER TABLE training_jobs DROP COLUMN IF EXISTS input_path; 
    ALTER TABLE training_jobs DROP COLUMN IF EXISTS output_path;



    
     
 
 

    ALTER TABLE training_jobs ADD COLUMN IF NOT EXISTS description TEXT; 
    ALTER TABLE training_jobs ADD COLUMN IF  NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT NOW(); 
    ALTER TABLE training_jobs ADD COLUMN IF  NOT EXISTS bucket_name TEXT DEFAULT ""; 
    ALTER TABLE training_jobs ADD COLUMN IF  NOT EXISTS session_id text TEXT NOT NULL; 



ALTER TABLE training_jobs 
    ADD COLUMN IF NOT EXISTS nodes    jsonb DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS edges    jsonb DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS pipeline jsonb DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS tags     jsonb DEFAULT '[]';