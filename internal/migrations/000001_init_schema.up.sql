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
    
 
 