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

	// "go.uber.org/zap"

	"llm-inference-service/internal/config"
	"llm-inference-service/internal/db"
	discovery "llm-inference-service/internal/eureka"
	"llm-inference-service/internal/nats"
	"llm-inference-service/internal/repository"
	service "llm-inference-service/internal/services"
	"llm-inference-service/internal/sse"
	handler "llm-inference-service/internal/transport/handler"
	"llm-inference-service/internal/transport/middleware"
	"llm-inference-service/pkg/logger"
	// "llm-inference-service/pkg/logger"
)

// runMigrations applies all pending up migrations from ./internal/migrations.
func runMigrations(cfg config.DBConfig) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode,
	)
	m, err := migrate.New("file://./internal/migrations", dsn)
	if err != nil {
		log.Fatalf("migrate: failed to initialise: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migrate: up failed: %v", err)
	}
	logger.Info("Database migrations applied")

}

func main() {
	cfg := config.Load()

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
		logger.Warn("NATS not connected, pressing on without it", zap.Error(err))
		nc = nil

	}
	sseHub := sse.NewHub()

	if nc != nil {
		// SSE hub — routes EC2 lifecycle events to browser clients.
		if err := nc.Subscriber.SubscribeInstanceEvents(sseHub); err != nil {
			log.Fatalf("Failed to subscribe to instance events: %v", err)
		}
		defer nc.Conn.Close()
	}
	logger.Info("Connected to NATS")

	// Model store (still in-memory for now)
	modelStore := repository.NewPostgresModelRepository(database)
	modelService := service.NewModelService(modelStore)

	trainingRepo := repository.NewPostgresTrainingRepo(database)
	trainingService := service.NewTrainingService(trainingRepo, nc, cfg.NatsPrefix)

	docsService := service.NewDocsService(cfg.DocsPath)

	// Workers

	// Handlers
	inferenceHandler := handler.NewInferenceHandler(nc)
	modelHandler := handler.NewModelHandler(modelService, nc)
	trainingHandler := handler.NewTrainingHandler(trainingService)
	docsHandlers := handler.NewDocsHandler(docsService)
	vmHandler := handler.NewVMHandler(sseHub)

	// Router
	r := chi.NewRouter()
	apiVersion := "/api/v1/llm"
	r.Use(middleware.AuthMiddleware)

	r.Route(apiVersion, func(r chi.Router) {
		r.Get("/health", modelHandler.Health)

	})

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
		r.Post("/jobs/{jobID}/nodes/{nodeID}/execute", trainingHandler.ExecuteNode)
		r.Put("/jobs/{jobID}", trainingHandler.UpdateJob)
		r.Post("/jobs/{jobID}/scripts/upload", trainingHandler.UploadScript)
		r.Get("/jobs/{jobID}", trainingHandler.GetJobByID)
		r.Post("/jobs/{jobID}/deploy", modelHandler.DeployModel)

	})

	r.Route("/api/v1/llm/vm", func(r chi.Router) {
		r.Get("/events/{sessionID}", vmHandler.StreamEvents)
		r.Options("/events/{sessionID}", vmHandler.StreamEvents)
	})

	apiVersionDocs := "/api/v1/llm/docs"

	r.Route(apiVersionDocs, func(r chi.Router) {

		r.Get("/", docsHandlers.GetPublicManifest)
		r.Get("/{slug}", docsHandlers.GetPublicDoc)
	})

	// 1. Register with Eureka (with retries)
	logger.Info("Attempting Eureka registration: " + cfg.Eureka.AppName)

	for i := 0; i < 3; i++ {
		err := discovery.RegisterWithEureka(cfg.Eureka)
		if err != nil {
			logger.Warn("Eureka registration attempt failed attempt %d with error %v", i+1, err.Error())

			if i < 2 {
				time.Sleep(5 * time.Second)
			}
		} else {
			logger.Info("Eureka registration successful")
			break
		}
	}

	// Start heartbeat
	go discovery.SendHeartbeat(cfg.Eureka)

	logger.Info("Server running on " + cfg.ServerPort)
	if err := http.ListenAndServe(cfg.ServerPort, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
