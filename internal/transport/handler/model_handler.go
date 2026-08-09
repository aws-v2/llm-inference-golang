package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"llm-inference-service/internal/transport/middleware"
	model "llm-inference-service/internal/models"
	"llm-inference-service/internal/nats"
	"llm-inference-service/internal/services"
	"llm-inference-service/internal/utils"

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
	requestID := r.Context().Value("requestId").(string)
	log.Println("[GetMyModels] extracted userID:", userID)

	if userID == "" {
		log.Printf("[Handler:GetMyModels] Service call,  for requestID %s, with error %s", requestID, "unauthorized")
		utils.WriteJSONError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	models, err := h.service.GetByOwner(userID) // handle the error now returned
	if err != nil {
		log.Printf("[Handler:GetMyModels] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Failed to fetch models"))
		return
	}
	log.Printf("[GetMyModels] models fetched: count=%d\n", len(models))

	utils.WriteJSONSucces(w, http.StatusOK, "models fetched successfully", models)
	log.Println("[GetMyModels] response sent successfully")
}

func (h *ModelHandler) GetModelDetails(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetOwnerID(r)
	requestID := r.Context().Value("requestId").(string)
	modelID := chi.URLParam(r, "modelID")

	m, err := h.service.GetByID(modelID, userID) // go through the service, not store directly
	if err != nil {
		log.Printf("[Handler:GetModelDetails] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusNotFound, fmt.Errorf("model not found"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "model fetched successfully", m)
}

type updateConfigRequest struct {
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

func (h *ModelHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetOwnerID(r)
	requestID := r.Context().Value("requestId").(string)
	modelID := chi.URLParam(r, "modelID")

	var req updateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Handler:UpdateConfig] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("invalid body"))
		return
	}

	if err := h.service.UpdateConfig(modelID, userID, req.Temperature, req.MaxTokens); err != nil {
		log.Printf("[Handler:UpdateConfig] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("failed to update config"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "config updated successfully", map[string]string{"id": modelID, "status": "updated"})
}

func (h *ModelHandler) Health(w http.ResponseWriter, r *http.Request) {
	mapResponse :=make(map[string]string)
	mapResponse["ping"] ="pong"
	utils.WriteJSONSucces(w, http.StatusOK, "model registered successfully", mapResponse)


}
func (h *ModelHandler) RegisterModel(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("requestId").(string)
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Handler:RegisterModel] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("invalid body"))
		return
	}

	ownerID := middleware.GetOwnerID(r)

	m, err := h.service.Register(req.Name, ownerID) // handle error now returned
	if err != nil {
		log.Printf("[Handler:RegisterModel] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to register model"))
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
		log.Printf("[Handler:RegisterModel] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to create upload url"))
		return
	}

	var presignResp struct {
		URL string `json:"upload_url"`
	}

	log.Println("Presign response bytes:", string(respBytes))

	if err := json.Unmarshal(respBytes, &presignResp); err != nil {
		log.Printf("[Handler:RegisterModel] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to parse presign response"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "model registered successfully", registerResponse{
		ModelID:   m.ID,
		UploadURL: "http://localhost:8080" + presignResp.URL,
		Status:    string(model.StatusPendingUpload),
	})
}



func (h *ModelHandler) DeployModel(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("requestId").(string)

	ownerID := middleware.GetOwnerID(r)
	if ownerID == "" {
		log.Printf("[Handler:DeployModel] Service call,  for requestID %s, with error %s", requestID, "unauthorized")
		utils.WriteJSONError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	modelID := chi.URLParam(r, "modelID")

	model, err := h.service.Deploy(modelID, ownerID)
	if err != nil {
		log.Printf("[Handler:DeployModel] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("failed to deploy model"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "model deployed successfully", model)
}