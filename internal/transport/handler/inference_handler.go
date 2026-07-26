package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"llm-inference-service/internal/utils"
	"llm-inference-service/internal/nats"
	"llm-inference-service/pkg/models"
)

type InferenceHandler struct {
	natsClient *nats.Client
	timeout    time.Duration
}

func NewInferenceHandler(nc *nats.Client) *InferenceHandler {
	return &InferenceHandler{
		natsClient: nc,
		timeout:    30 * time.Second,
	}
}

type errorResponse struct {
	Error string `json:"error"`
}
func (h *InferenceHandler) Infer(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

		ownerID := r.Context().Value("userId").(string)
	requestID := r.Context().Value("requestId").(string)


	if ownerID == "" {
		log.Printf("[Handler:Infer] Service call,  for requestID %s, with error %s", requestID, "unauthorized")
		utils.WriteJSONError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		return
	}

	// --- 2. Parse request body ---
	var req models.InferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Handler:Infer] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON body"))
		return
	}

	// --- 3. Validate input ---
	if req.Prompt == "" {
		log.Printf("[Handler:Infer] Service call,  for requestID %s, with error %s", requestID, "prompt is required")
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("prompt is required"))
		return
	}

	if req.ModelID == "" {
		log.Printf("[Handler:Infer] Service call,  for requestID %s, with error %s", requestID, "model_id is required")
		utils.WriteJSONError(w, http.StatusBadRequest, fmt.Errorf("model_id is required"))
		return
	}

	// --- 4. Attach owner to request (IMPORTANT) ---

	// --- 5. Static NATS subject ---
	subject := "llmgateway.task.infer"

	log.Println("Inference request",
		"owner:", ownerID,
		"model:", req.ModelID,
		"subject:", subject,
	)

	// --- 6. NATS request ---
	respBytes, err := h.requestWithContext(ctx, subject, req)
	if err != nil {
		log.Printf("[Handler:Infer] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusGatewayTimeout, fmt.Errorf("Gateway timeout"))
		return
	}

	// --- 7. Response ---
	var rawData interface{}
	// attempt to parse the JSON bytes returned to map/struct so we can put it in Data
	if err := json.Unmarshal(respBytes, &rawData); err != nil {
		rawData = string(respBytes) // fallback
	}
	utils.WriteJSONSucces(w, http.StatusOK, "inference successful", rawData)
}




func (h *InferenceHandler) requestWithContext(ctx context.Context, subject string, payload interface{}) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}



	ch := make(chan result, 1)

	go func() {
		resp, err := h.natsClient.Publisher.Request(subject, payload)
		ch <- result{data: resp, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		return res.data, res.err
	}
}
