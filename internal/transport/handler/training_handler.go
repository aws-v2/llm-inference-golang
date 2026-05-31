package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"llm-inference-service/internal/middleware"
	model "llm-inference-service/internal/models"
	service "llm-inference-service/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

	sessionID := uuid.New().String()
	log.Printf("this is the ownerid: %s", ownerID)
	if ownerID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req model.TrainingJobDto
	req.SessionID = sessionID
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	// req.OwnerID =ownerID

	job, err := h.service.CreateJob(ownerID, req)
	if err != nil {
		log.Println("Failed to create job", err)
		http.Error(w, "Failed to create job", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(job)
}

func (h *TrainingHandler) ExecuteNode(w http.ResponseWriter, r *http.Request) {
	ownerID := middleware.GetOwnerID(r)
	if ownerID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	jobID  := chi.URLParam(r, "jobID")
	nodeID := chi.URLParam(r, "nodeID")
	
	// Get the optional script_id from query parameters
	scriptID := r.URL.Query().Get("script_id")

	if jobID == "" || nodeID == "" {
		http.Error(w, "Missing jobID or nodeID", http.StatusBadRequest)
		return
	}

	// Log the mode of execution
	if scriptID != "" {
		log.Printf("[EXECUTION] Owner: %s, Job: %s, Node: %s -> Executing SINGLE SCRIPT: %s", ownerID, jobID, nodeID, scriptID)
	} else {
		log.Printf("[EXECUTION] Owner: %s, Job: %s, Node: %s -> Executing FULL NODE (all scripts)", ownerID, jobID, nodeID)
	}

	// Pass the scriptID to the service. (Ensure your service method signature is updated to accept it)
	result, err := h.service.ExecuteNode(ownerID, jobID, nodeID, scriptID)
	if err != nil {
		log.Printf("execute node error: %s", err)
		http.Error(w, "Failed to execute node", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}












func (h *TrainingHandler) UploadScript(w http.ResponseWriter, r *http.Request) {
	
	ownerID := middleware.GetOwnerID(r)
	if ownerID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	jobID    := chi.URLParam(r, "jobID")
	nodeID   := r.FormValue("node_id")
	nodeType := r.FormValue("node_type")
	routeTo  := r.FormValue("route_to")

	// ── Flow 1: name + s3 path typed manually ────────────────────────────
	scriptName := r.FormValue("name")
	scriptPath := r.FormValue("path")

	if scriptName != "" && scriptPath != "" {
		script := model.PipelineScript{
			ID:      fmt.Sprintf("script-%s", uuid.New().String()[:8]),
			Name:    scriptName,
			Path:    scriptPath,
			RouteTo: routeTo,
		}
		result, err := h.service.AddScriptByPath(ownerID, jobID, nodeID, script)
		if err != nil {
			log.Printf("add script by path error: %s", err)
			http.Error(w, "Failed to add script", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(result)
		return
	}


		log.Printf("this seems to be a file upload moving on to the next thing ")
	

	// ── Flow 2: file upload ───────────────────────────────────────────────
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Provide either (name + path) or a file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	resp, err := h.service.GetScriptUploadURL(ownerID, jobID, nodeID, nodeType, header.Filename, contentType, routeTo, file)
	if err != nil {
		log.Printf("upload script error: %s", err)
		http.Error(w, "Failed to get upload URL", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *TrainingHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {
    ownerID := middleware.GetOwnerID(r)
    if ownerID == "" {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    jobID := chi.URLParam(r, "jobID")
    if jobID == "" {
        http.Error(w, "Missing job ID", http.StatusBadRequest)
        return
    }

    var req model.TrainingJobDto
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid body", http.StatusBadRequest)
        return
    }

    // These override anything the client sends
    req.ID      = jobID
    req.OwnerID = ownerID

    log.Printf("updating job id: %s owner: %s", req.ID, req.OwnerID)

    response, err := h.service.UpdateJob(req)
    if err != nil {
        log.Printf("update error: %s", err)
        http.Error(w, "Failed to update job", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(response)
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

		log.Printf("thiss the erro: %s", err)
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
		log.Printf("the error is: %s", err)
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(job)
}
