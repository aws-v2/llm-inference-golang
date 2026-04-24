package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"llm-inference-service/internal/middleware"
	"llm-inference-service/internal/models"
	service "llm-inference-service/internal/services"

	"github.com/go-chi/chi/v5"
)

type TrainingHandler struct {
	service *service.TrainingService
}

func NewTrainingHandler(s *service.TrainingService) *TrainingHandler {
	return &TrainingHandler{service: s}
}

/**
 * CREATE TRAINING JOB
 */
func (h *TrainingHandler) CreateJob(w http.ResponseWriter, r *http.Request) {

	ownerID := middleware.GetOwnerID(r)
	if ownerID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req model.TrainingJob
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	job, err := h.service.CreateJob(ownerID, req)
	if err != nil {
		log.Println("Failed to create job", err)
		http.Error(w, "Failed to create job", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(job)
}

/**
 * GET ALL TRAINING JOBS
 */
func (h *TrainingHandler) GetAllJobs(w http.ResponseWriter, r *http.Request) {

	ownerID := middleware.GetOwnerID(r)
	if ownerID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	jobs, err := h.service.GetAll(ownerID)
	if err != nil {
		http.Error(w, "Failed to fetch jobs", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(jobs)
}

func (h *TrainingHandler) GetJobByID(w http.ResponseWriter, r *http.Request) {

	ownerID := middleware.GetOwnerID(r)
	if ownerID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	jobID := chi.URLParam(r, "jobID")

	job, err := h.service.GetByID(jobID, ownerID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(job)
}