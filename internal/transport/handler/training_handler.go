package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	model "llm-inference-service/internal/models"
	service "llm-inference-service/internal/services"
	"llm-inference-service/internal/transport/middleware"
	"llm-inference-service/internal/utils"

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
	requestID := middleware.GetRequestID(r)
	ownerID := middleware.GetOwnerID(r)

	sessionID := uuid.New().String()
	log.Printf("this is the ownerid: %s", ownerID)
	if ownerID == "" {
		log.Printf("[Handler:CreateJob] Service call,  for requestID %s, with error %s", requestID, "unauthorized")
		utils.WriteJSONError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	var req model.TrainingJobDto
	req.SessionID = sessionID
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Handler:CreateJob] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("Invalid body"))
		return
	}
	// req.OwnerID =ownerID

	job, err := h.service.CreateJob(ownerID, req)
	if err != nil {
		log.Printf("[Handler:CreateJob] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Failed to create job"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "job created successfully", job)
}

func (h *TrainingHandler) ExecuteNode(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r)
	ownerID := middleware.GetOwnerID(r)
	if ownerID == "" {
		log.Printf("[Handler:ExecuteNode] Service call,  for requestID %s, with error %s", requestID, "unauthorized")
		utils.WriteJSONError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	jobID := chi.URLParam(r, "jobID")
	nodeID := chi.URLParam(r, "nodeID")
	scriptID := r.URL.Query().Get("script_id")

	if jobID == "" || nodeID == "" {
		log.Printf("[Handler:ExecuteNode] Service call,  for requestID %s, with error %s", requestID, "Missing jobID or nodeID")
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("Missing jobID or nodeID"))
		return
	}

	// Generate a unique session ID for accurate SSE tracking.
	sessionID := uuid.New().String()

	if scriptID != "" {
		log.Printf("[EXECUTION] Owner: %s, Job: %s, Node: %s -> Executing SINGLE SCRIPT: %s (Session: %s)", ownerID, jobID, nodeID, scriptID, sessionID)
	} else {
		log.Printf("[EXECUTION] Owner: %s, Job: %s, Node: %s -> Executing FULL NODE (Session: %s)", ownerID, jobID, nodeID, sessionID)
	}

	// Launch execution in the background so we can return the sessionID immediately.
	// This allows the frontend to establish its SSE connection before provisioning starts.
	go func() {
		_, err := h.service.ExecuteNode(ownerID, jobID, nodeID, scriptID, sessionID)
		if err != nil {
			log.Printf("[ASYNC EXECUTION ERROR] Job: %s, Node: %s, Session: %s: %v", jobID, nodeID, sessionID, err)
		}
	}()

	// Return initial status and sessionID immediately.
	utils.WriteJSONSucces(w, http.StatusOK, "execution started", map[string]interface{}{
		"session_id": sessionID,
		"job_id":     jobID,
		"node_id":    nodeID,
		"status":     "running",
	})
}

// CalculateSHA256Bytes calculates the SHA256 hash of the byte slice.
func CalculateSHA256Bytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (h *TrainingHandler) UploadScript(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r)
	ownerID := middleware.GetOwnerID(r)
	if ownerID == "" {
		log.Printf("[Handler:UploadScript] Service call,  for requestID %s, with error %s", requestID, "unauthorized")
		utils.WriteJSONError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	jobID := chi.URLParam(r, "jobID")
	nodeID := r.FormValue("node_id")
	nodeType := r.FormValue("node_type")
	routeTo := r.FormValue("route_to")

	file, _, err := r.FormFile("file")
	if err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, err)
	return
	}
	defer file.Close()

	data, err:= io.ReadAll(file)
	checksum :=CalculateSHA256Bytes(data)
	if err !=nil{
		utils.WriteJSONError(w,http.StatusBadRequest, err)
		return 
	}
 
	log.Printf("this seems to be a file upload moving on to the next thing ")

	// ── Flow 2: file upload ───────────────────────────────────────────────
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Printf("[Handler:UploadScript] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("Failed to parse form"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("[Handler:UploadScript] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("Provide either (name + path) or a file"))
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	resp, err := h.service.GetScriptUploadURL(ownerID, jobID, nodeID, nodeType, header.Filename, contentType, routeTo, file, checksum)
	if err != nil {
		log.Printf("[Handler:UploadScript] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get upload URL"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "script upload URL generated", resp)
}
func (h *TrainingHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r)
	ownerID := middleware.GetOwnerID(r)
	if ownerID == "" {
		log.Printf("[Handler:UpdateJob] Service call,  for requestID %s, with error %s", requestID, "unauthorized")
		utils.WriteJSONError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	jobID := chi.URLParam(r, "jobID")
	if jobID == "" {
		log.Printf("[Handler:UpdateJob] Service call,  for requestID %s, with error %s", requestID, "Missing job ID")
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("Missing job ID"))
		return
	}

	var req model.TrainingJobDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Handler:UpdateJob] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("Invalid body"))
		return
	}

	// These override anything the client sends
	req.ID = jobID
	req.OwnerID = ownerID

	log.Printf("updating job id: %s owner: %s", req.ID, req.OwnerID)

	response, err := h.service.UpdateJob(req)
	if err != nil {
		log.Printf("[Handler:UpdateJob] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Failed to update job"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "job updated successfully", response)
}

/**
 * GET ALL TRAINING JOBS
 */
func (h *TrainingHandler) GetAllJobs(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r)
	ownerID := middleware.GetOwnerID(r)
	if ownerID == "" {
		log.Printf("[Handler:GetAllJobs] Service call,  for requestID %s, with error %s", requestID, "unauthorized")
		utils.WriteJSONError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	jobs, err := h.service.GetAll(ownerID)
	if err != nil {
		log.Printf("[Handler:GetAllJobs] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Failed to fetch jobs"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "jobs fetched successfully", jobs)
}

func (h *TrainingHandler) GetJobByID(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r)
	ownerID := middleware.GetOwnerID(r)
	if ownerID == "" {
		log.Printf("[Handler:GetJobByID] Service call,  for requestID %s, with error %s", requestID, "unauthorized")
		utils.WriteJSONError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	jobID := chi.URLParam(r, "jobID")

	job, err := h.service.GetByID(jobID, ownerID)
	if err != nil {
		log.Printf("[Handler:GetJobByID] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusNotFound, fmt.Errorf("Job not found"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "job fetched successfully", job)
}
