package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"llm-inference-service/internal/middleware"
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

	// --- 1. Extract OwnerID from JWT ---
	ownerID := middleware.GetOwnerID(r)
	if ownerID == "" {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// --- 2. Parse request body ---
	var req models.InferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// --- 3. Validate input ---
	if req.Prompt == "" {
		writeError(w, "prompt is required", http.StatusBadRequest)
		return
	}

	if req.ModelID == "" {
		writeError(w, "model_id is required", http.StatusBadRequest)
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
		writeError(w, err.Error(), http.StatusGatewayTimeout)
		return
	}

	// --- 7. Response ---
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
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

func writeError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	json.NewEncoder(w).Encode(errorResponse{
		Error: msg,
	})
}