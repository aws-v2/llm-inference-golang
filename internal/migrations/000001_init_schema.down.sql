DROP TABLE training_jobs;
DROP TABLE models;


--  migrate -path ./internal/migrations   -database "postgres://root:root@localhost:5432/sagemaker_inference_db?sslmode=disable" up