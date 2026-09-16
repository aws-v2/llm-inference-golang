package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	model "llm-inference-service/internal/domain/models"
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

func (h *TrainingHandler) RemoveFile(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r)
	jobID := chi.URLParam(r, "jobID")
	fileID := chi.URLParam(r, "fileID")

	if fileID == "" || jobID == "" {
		log.Printf("[Handler:RemoveFile] Failed to remove file record to job %s: %v", requestID, "Invalid file or job id")

		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Failed to add file record to job"))
		return
	}

	err := h.service.RemoveFile(fileID, jobID)
	if err != nil {
		log.Printf("[Handler:RemoveFile] Failed to remove file %s record to job %s error: %v", fileID, jobID, err)

		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Failed to remove file %s from record for jobId %s", fileID, jobID))
		return

	}

	utils.WriteJSONSucces(w, http.StatusOK, "Files removed successfully", fileID)


}
func (h *TrainingHandler) AddFileRecord(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r)
	jobID := chi.URLParam(r, "jobID")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Handler:CreateJob] Service call, for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("Invalid body"))
		return
	}

	var addFileReqs []model.AddFileRequest

	var req model.AddFileRequest
	err = json.Unmarshal(bodyBytes, &req)

	if err != nil {
		log.Printf("[Handler:CreateJob] CreateLLMProject error for requestID %s: %v", requestID, err)
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Failed to create job"))
		return
	}
	addFileReqs = append(addFileReqs, req)
	err = h.service.AddFileRecord(jobID, addFileReqs)

	if err != nil {
		log.Printf("[Handler:AddFileRecord] Failed to add file record to job %s: %v", requestID, err)

		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Failed to add file record to job"))
		return

	}

	type AddFileRecordResponse struct {
		StatusCode int    `json:"status_code"`
		FileName   string `json:"file_name"`
	}

	utils.WriteJSONSucces(w, http.StatusOK, "Files added successfully", AddFileRecordResponse{FileName: req.FileName, StatusCode: 200})

}
func (h *TrainingHandler) AddFile(w http.ResponseWriter, r *http.Request) {
	log.Printf("len(fileHeaders): ****")

	requestID := middleware.GetRequestID(r)
	ownerID := middleware.GetOwnerID(r)
	jobID := chi.URLParam(r, "jobID")

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		log.Printf("[Handler:AddFile] ParseMultipartForm error for requestID %v: %v", requestID, err)
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("invalid multipart form data"))
		return
	}

	file_role := strings.TrimSpace(r.FormValue("role"))

	fileHeaders := r.MultipartForm.File["file"]
	if len(fileHeaders) == 0 {
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("at least one file is required"))
		return
	}

	var addFileReqs []model.AddFileRequest

	for _, fh := range fileHeaders {
		file, err := fh.Open()
		if err != nil {
			log.Printf("[Handler:AddFile] failed to open file %s for requestID %v: %v", fh.Filename, requestID, err)
			utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("failed to read file %s", fh.Filename))
			return
		}

		fileData, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			log.Printf("[Handler:AddFile] failed to read file %s for requestID %v: %v", fh.Filename, requestID, err)
			utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("failed to read file %s", fh.Filename))
			return
		}

		sha256Hex := CalculateSHA256Bytes(fileData)

		addFileReqs = append(addFileReqs, model.AddFileRequest{
			FileName:    fh.Filename,
			Role:        file_role,
			OwnerID:     ownerID,
			ProjectId:   jobID,
			Sha256:      sha256Hex,
			Size:        fh.Size,
			ContentType: fh.Header.Get("Content-Type"),
		})
	}

	results, err := h.service.AddFile(jobID, addFileReqs)
	if err != nil {
		log.Printf("[Handler:AddFile] service call failed for requestID %v: %v", requestID, err)
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to add file(s)"))
		return
	}

	log.Println("[Handler:AddFile] files added, ownerID:", ownerID, "jobID:", results[0])
	// type FileUploadResult struct {
	//     FileName  string `json:"file_name"`
	//     UploadURL string `json:"upload_url"`
	// }
	type AddFileResponse struct {
		Files []model.FileUploadResult `json:"files"`
	}

	utils.WriteJSONSucces(w, http.StatusOK, "Files added successfully", AddFileResponse{
		Files: results,
	})
}

func (h *TrainingHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r)
	ownerID := middleware.GetOwnerID(r)

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Handler:CreateJob] Service call, for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("Invalid body"))
		return
	}

	var req model.CreateLLMJobRequest
	if err := json.Unmarshal(bodyBytes, &req); err == nil && (req.Name != "" || len(req.Files) > 0 || req.BaseModel != "") {
		resp, err := h.service.CreateLLMProject(ownerID, req, r.Host)
		if err != nil {
			log.Printf("[Handler:CreateJob] CreateLLMProject error for requestID %s: %v", requestID, err)
			utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Failed to create job"))
			return
		}
		utils.WriteJSONSucces(w, http.StatusOK, "LLM project created successfully", resp)
		return
	}

	var legacyReq model.TrainingJobDto
	if err := json.Unmarshal(bodyBytes, &legacyReq); err != nil {
		log.Printf("[Handler:CreateJob] Invalid body for requestID %s", requestID)
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("Invalid body"))
		return
	}

	sessionID := uuid.New().String()
	legacyReq.SessionID = sessionID
	job, err := h.service.CreateJob(ownerID, legacyReq)
	if err != nil {
		log.Printf("[Handler:CreateJob] Service call, for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Failed to create job"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "LLM project created successfully", job)
}

func (h *TrainingHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r)
	jobID := chi.URLParam(r, "id")
	if jobID == "" {
		jobID = chi.URLParam(r, "jobID")
	}
	if jobID == "" {
		log.Printf("[Handler:UpdateJob] Service call, for requestID %s, missing job ID", requestID)
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("Missing job ID"))
		return
	}

	var req model.LLMTrainingJob
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Handler:UpdateJob] Service call, for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("Invalid body"))
		return
	}

	updated, err := h.service.UpdateLLMProject(jobID, req)
	if err != nil {
		log.Printf("[Handler:UpdateJob] Service call, for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Failed to update project"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "Project updated successfully", updated)
}

func (h *TrainingHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r)
	jobID := chi.URLParam(r, "id")
	if jobID == "" {
		jobID = chi.URLParam(r, "jobID")
	}
	if jobID == "" {
		log.Printf("[Handler:DeleteJob] Service call, for requestID %s, missing job ID", requestID)
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("Missing job ID"))
		return
	}

	if err := h.service.DeleteLLMProject(jobID); err != nil {
		log.Printf("[Handler:DeleteJob] Service call, for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Failed to delete project"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "Project deleted successfully", nil)
}

func (h *TrainingHandler) GetAllJobs(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r)
	ownerID := middleware.GetOwnerID(r)

	llmJobs, err := h.service.GetAllLLMProjects(ownerID)
	if err == nil && len(llmJobs) > 0 {
		utils.WriteJSONSucces(w, http.StatusOK, "Projects retrieved successfully", llmJobs)
		return
	}

	legacyJobs, err := h.service.GetAll(ownerID)
	if err != nil {
		log.Printf("[Handler:GetAllJobs] Service call, for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONSucces(w, http.StatusOK, "Projects retrieved successfully", []model.LLMTrainingJob{})
		return
	}
	utils.WriteJSONSucces(w, http.StatusOK, "Projects retrieved successfully", legacyJobs)
}

func (h *TrainingHandler) HandleVMReportStatus(w http.ResponseWriter, r *http.Request) {
	step := chi.URLParam(r, "step")

	var req model.VMReportStatus
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error decoding vm status response %v", err)
		return
	}

	log.Printf("this step is %v", step)
	log.Printf("The returned data: run_id %v, step: %v, status: %v, duration:%v", req.RunID, req.Step, req.Status, req.Duration)

	h.service.HandleVMReportStatus(req)

	utils.WriteJSONSucces(w, http.StatusOK, "Projects retrieved successfully", "")

}
func (h *TrainingHandler) RunJobByID(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r)
	// ownerID := middleware.GetOwnerID(r)

	projectId := chi.URLParam(r, "id")
	if projectId == "" {
		projectId = chi.URLParam(r, "jobID")
	}

	project, err := h.service.GetLLMProjectByID(projectId)
	if err != nil {
		log.Printf("[Handler:GetJobByID] Service call**, for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusOK, fmt.Errorf("%s", err.Error()))
		return
	}

	run, err := h.service.RunPipeline(project)

	utils.WriteJSONSucces(w, http.StatusOK, "Project retrieved successfully", run)
	return

}

func (h *TrainingHandler) GetJobByID(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r)
	ownerID := middleware.GetOwnerID(r)

	jobID := chi.URLParam(r, "id")
	if jobID == "" {
		jobID = chi.URLParam(r, "jobID")
	}

	job, err := h.service.GetLLMProjectByID(jobID)
	if err == nil {
		utils.WriteJSONSucces(w, http.StatusOK, "Project retrieved successfully", job)
		return
	}

	legacyJob, err := h.service.GetByID(jobID, ownerID)
	if err != nil {
		log.Printf("[Handler:GetJobByID] Service call, for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusNotFound, fmt.Errorf("Job not found"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "Project retrieved successfully", legacyJob)
}

// CalculateSHA256Bytes calculates the SHA256 hash of the byte slice.
func CalculateSHA256Bytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
