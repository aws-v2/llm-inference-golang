package handler

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"llm-inference-service/internal/nats"
	service "llm-inference-service/internal/services"
	"llm-inference-service/internal/transport/middleware"
	"llm-inference-service/internal/utils"
	model "llm-inference-service/internal/domain/models"


	"github.com/go-chi/chi/v5"
)

type ModelHandler struct {
	service         *service.ModelService
	nc              *nats.Client
	controlPlaneURL string
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
	requestID := middleware.GetRequestID(r)
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

// func (h *ModelHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
// 	userID := middleware.GetOwnerID(r)
// 	requestID := r.Context().Value("requestId").(string)
// 	modelID := chi.URLParam(r, "modelID")

// 	var req updateConfigRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		log.Printf("[Handler:UpdateConfig] Service call,  for requestID %s, with error %s", requestID, err.Error())
// 		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("invalid body"))
// 		return
// 	}

// 	if err := h.service.UpdateConfig(modelID, userID, req.Temperature, req.MaxTokens); err != nil {
// 		log.Printf("[Handler:UpdateConfig] Service call,  for requestID %s, with error %s", requestID, err.Error())
// 		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("failed to update config"))
// 		return
// 	}

// 	utils.WriteJSONSucces(w, http.StatusOK, "config updated successfully", map[string]string{"id": modelID, "status": "updated"})
// }

func (h *ModelHandler) Health(w http.ResponseWriter, r *http.Request) {
	mapResponse := make(map[string]string)
	mapResponse["ping"] = "pong"
	utils.WriteJSONSucces(w, http.StatusOK, "model registered successfully", mapResponse)

}
func (h *ModelHandler) RegisterModel(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("requestId")

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		log.Printf("[Handler:RegisterModel] ParseMultipartForm error for requestID %v: %v", requestID, err)
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("invalid multipart form data"))
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("name is required"))
		return
	}

	isPublicStr := strings.TrimSpace(r.FormValue("is_public"))
	if isPublicStr == "" {
		isPublicStr = strings.TrimSpace(r.FormValue("isPublic"))
	}
	isPublic := isPublicStr == "true" || isPublicStr == "1" || isPublicStr == "TRUE"

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("[Handler:RegisterModel] FormFile error for requestID %v: %v", requestID, err)
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("file is required"))
		return
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		log.Printf("[Handler:RegisterModel] Failed to read file for sha256 calculation: %v", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to process file"))
		return
	}

	fileData, err := io.ReadAll(file)
	if err != nil {
		log.Printf("[Handler:RegisterModel] FormFile error for requestID %v: %v", requestID, err)
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("file is required"))
		return
	}
	// sha256Hex := hex.EncodeToString(hasher.Sum(nil))
	sha256Hex :=CalculateSHA256Bytes(fileData)

	ownerID := middleware.GetOwnerID(r)
	req := model.RegisterRequest{
		Name:     name,
		OwnerID:  ownerID,
		IsPublic: isPublic,
		Sha256:   sha256Hex,
	}

	m, err := h.service.Register(req)
	if err != nil {
		log.Printf("[Handler:RegisterModel] Service call, for requestID %v, with error %v", requestID, err)
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to register model"))
		return
	}
	log.Println("[RegisterModel] Model registered, ownerID:", ownerID)

	filename := header.Filename
	if filename == "" {
		filename = req.Name
	}

	s3Key := fmt.Sprintf("models/%s/%s", m.ID, filename)

	respBytes, err := h.nc.Publisher.Request("s3.task.create_presigned_url", model.CreatePresignedUploadURLRequest{
		UserID:     ownerID,
		GameID:     m.ID,
		AssetID:    m.ID,
		AssetType:  "models",
		AssetName:  req.Name,
		Key:        s3Key,
		Sha256:     req.Sha256,
		BucketName: "sagemaker-base-models",
	})
	if err != nil {
		log.Printf("[Handler:RegisterModel] Service call, for requestID %v, with error %v", requestID, err)
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to create upload url"))
		return
	}

	var presignResp struct {
		URL string `json:"upload_url"`
	}

	log.Println("Presign response bytes:", string(respBytes))

	if err := json.Unmarshal(respBytes, &presignResp); err != nil {
		log.Printf("[Handler:RegisterModel] Service call, for requestID %v, with error %v", requestID, err)
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to parse presign response"))
		return
	}

	uploadURL := presignResp.URL
	if uploadURL != "" && !strings.HasPrefix(uploadURL, "http") && !strings.HasPrefix(uploadURL, "https://") {
		uploadURL = h.controlPlaneURL + uploadURL
	}

// http://localhost:8080/api/v1/api/v1/s3/files/upload


	utils.WriteJSONSucces(w, http.StatusOK, "model registered successfully", registerResponse{
		ModelID:   m.ID,
		UploadURL: strings.TrimPrefix(uploadURL,"/api/v1"),
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
