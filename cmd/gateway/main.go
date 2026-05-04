package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"

	"llm-inference-service/internal/config"
	"llm-inference-service/internal/db"
	discovery "llm-inference-service/internal/eureka"
	"llm-inference-service/internal/nats"
	"llm-inference-service/internal/repository"
	service "llm-inference-service/internal/services"
	handler "llm-inference-service/internal/transport/handler"
	"llm-inference-service/pkg/logger"
)

// runMigrations applies all pending up migrations from ./internal/migrations.
func runMigrations(cfg config.DBConfig) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
	)
	m, err := migrate.New("file://./internal/migrations", dsn)
	if err != nil {
		log.Fatalf("migrate: failed to initialise: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migrate: up failed: %v", err)
	}
	log.Println("Database migrations applied")
}

func main() {
	cfg := config.Load()
	// Initialize logger FIRST
	logger.Init(
		cfg.ServiceName, // or hardcode if you don’t have it yet
		cfg.Profile,
		cfg.Region,
	)

	defer logger.Log.Sync()

	// Run DB migrations before opening connections
	runMigrations(cfg.DB)

	// DB
	database := db.NewPostgres(cfg.DB)

	// NATS
	nc, err := nats.NewClient(
		cfg.NatsURL,
		cfg.NatsUser,
		cfg.NatsPassword,
		cfg.NatsPrefix,
	)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Conn.Close()
	log.Println("Connected to NATS")

	// Model store (still in-memory for now)
	modelStore := repository.NewPostgresModelRepository(database)
	modelService := service.NewModelService(modelStore)

	trainingRepo := repository.NewPostgresTrainingRepo(database)
	trainingService := service.NewTrainingService(trainingRepo)

	docsService := service.NewDocsService("./docs")

	// Workers

	// Handlers
	inferenceHandler := handler.NewInferenceHandler(nc)
	modelHandler := handler.NewModelHandler(modelService, nc)
	trainingHandler := handler.NewTrainingHandler(trainingService)
	docsHandlers := handler.NewDocsHandler(docsService)

	// Router
	r := chi.NewRouter()
	apiVersion := "/api/v1/llm"

	r.Route(apiVersion+"/models", func(r chi.Router) {
		// r.Use(middleware.Auth) // JWT middleware
		r.Post("/register", modelHandler.RegisterModel)

		r.Post("/infer", inferenceHandler.Infer)

		r.Get("/", modelHandler.GetMyModels)              // ✅ all models for user
		r.Get("/{modelID}", modelHandler.GetModelDetails) // ✅ single model

		r.Put("/{modelID}/config", modelHandler.UpdateConfig) // ✅ update config
	})

	r.Route(apiVersion+"/training", func(r chi.Router) {
		r.Post("/jobs", trainingHandler.CreateJob)
		r.Get("/jobs", trainingHandler.GetAllJobs)

		r.Get("/jobs/{jobID}", trainingHandler.GetJobByID)
		r.Post("/jobs/{jobID}/deploy", modelHandler.DeployModel)

	})
	apiVersionDocs := "/api/v1/llm/docs"

	r.Route(apiVersionDocs, func(r chi.Router) {

		r.Get("/", docsHandlers.GetPublicManifest)
		r.Get("/internal/", docsHandlers.GetInternalManifest)

		r.Get("/{slug}", docsHandlers.GetPublicDoc)
		r.Get("/internal/{slug}", docsHandlers.GetInternalDoc)
	})

	// 1. Register with Eureka (with retries)
	logger.Log.Info("Attempting Eureka registration",
		zap.String("app", cfg.Eureka.AppName),
	)

	for i := 0; i < 3; i++ {
		err := discovery.RegisterWithEureka(cfg.Eureka, logger.Log)
		if err != nil {
			logger.Log.Warn("Eureka registration attempt failed",
				zap.Int("attempt", i+1),
				zap.Error(err),
			)

			if i < 2 {
				time.Sleep(5 * time.Second)
			}
		} else {
			logger.Log.Info("Eureka registration successful")
			break
		}
	}

	// Start heartbeat
	go discovery.SendHeartbeat(cfg.Eureka, logger.Log)

	log.Println("Server running on", cfg.ServerPort)
	http.ListenAndServe(cfg.ServerPort, r)
}
