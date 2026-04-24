package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"llm-inference-service/internal/middleware"
	model "llm-inference-service/internal/models"
	"llm-inference-service/internal/nats"
	"llm-inference-service/internal/services"

	"github.com/go-chi/chi/v5"
)

type ModelHandler struct {
	service *service.ModelService
	nc      *nats.Client
}

type registerRequest struct {
	Name string `json:"name"`
}

type registerResponse struct {
	ModelID   string `json:"model_id"`
	UploadURL string `json:"upload_url"`
	Status    string `json:"status"`
}

func NewModelHandler(s *service.ModelService, nc *nats.Client) *ModelHandler {
	return &ModelHandler{
		service: s,
		nc:      nc,
	}
}

func (h *ModelHandler) GetMyModels(w http.ResponseWriter, r *http.Request) {
	log.Println("[GetMyModels] request received")

	userID := middleware.GetOwnerID(r)
	log.Println("[GetMyModels] extracted userID:", userID)

	if userID == "" {
		log.Println("[GetMyModels] ERROR: missing userID (unauthorized)")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	models, err := h.service.GetByOwner(userID) // handle the error now returned
	if err != nil {
		log.Println("[GetMyModels] ERROR: failed to fetch models:", err)
		http.Error(w, "Failed to fetch models", http.StatusInternalServerError)
		return
	}
	log.Printf("[GetMyModels] models fetched: count=%d\n", len(models))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(models); err != nil {
		log.Println("[GetMyModels] ERROR: failed to encode response:", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	log.Println("[GetMyModels] response sent successfully")
}

func (h *ModelHandler) GetModelDetails(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetOwnerID(r)
	modelID := chi.URLParam(r, "modelID")

	m, err := h.service.GetByID(modelID, userID) // go through the service, not store directly
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(m)
}

type updateConfigRequest struct {
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

func (h *ModelHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetOwnerID(r)
	modelID := chi.URLParam(r, "modelID")

	var req updateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateConfig(modelID, userID, req.Temperature, req.MaxTokens); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ModelHandler) RegisterModel(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	ownerID := middleware.GetOwnerID(r)

	m, err := h.service.Register(req.Name, ownerID) // handle error now returned
	if err != nil {
		log.Println("[RegisterModel] ERROR: failed to register model:", err)
		http.Error(w, "failed to register model", http.StatusInternalServerError)
		return
	}
	log.Println("[RegisterModel] Model registered, ownerID:", ownerID)

	type presignRequest struct {
		ModelID string
		OwnerID string
	}

	respBytes, err := h.nc.Publisher.Request("s3.create_presigned_url", presignRequest{
		ModelID: m.ID,
		OwnerID: ownerID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var presignResp struct {
		URL string `json:"upload_url"`
	}

	log.Println("Presign response bytes:", string(respBytes))

	if err := json.Unmarshal(respBytes, &presignResp); err != nil {
		log.Println("[RegisterModel] ERROR: failed to unmarshal presign response:", err)
		http.Error(w, "failed to parse presign response", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(registerResponse{
		ModelID:   m.ID,
		UploadURL: "http://localhost:8080" + presignResp.URL,
		Status:    string(model.StatusPendingUpload),
	})
}



func (h *ModelHandler) DeployModel(w http.ResponseWriter, r *http.Request) {

	ownerID := middleware.GetOwnerID(r)
	if ownerID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	modelID := chi.URLParam(r, "modelID")

	model, err := h.service.Deploy(modelID, ownerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model)
}