package db

import (
	"database/sql"
	"fmt"
	"log"

	"llm-inference-service/internal/config"
	"llm-inference-service/pkg/logger"

	_ "github.com/lib/pq"
)

func NewPostgres(cfg config.DBConfig) *sql.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("DB connection error:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("DB ping failed:", err)
	}

	logger.Info("Connected to PostgreSQL")
	return db
}