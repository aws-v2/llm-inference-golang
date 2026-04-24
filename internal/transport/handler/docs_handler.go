package handler

import (
	"encoding/json"
	service "llm-inference-service/internal/services"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type DocsHandler struct {
	service *service.DocsService
}

func NewDocsHandler(service *service.DocsService) *DocsHandler {
	return &DocsHandler{service: service}
}

/**
 * GET /docs/manifest
 */
func (h *DocsHandler) GetPublicManifest(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.GetManifest(false)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": data,
	})
}

/**
 * GET /docs/internal/manifest
 */
func (h *DocsHandler) GetInternalManifest(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.GetManifest(true)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": data,
	})
}

/**
 * GET /docs/{slug}
 */
func (h *DocsHandler) GetPublicDoc(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	doc, err := h.service.GetDoc(slug, false)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": doc,
	})
}

/**
 * GET /docs/internal/{slug}
 */
func (h *DocsHandler) GetInternalDoc(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	doc, err := h.service.GetDoc(slug, true)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": doc,
	})
}

/**
 * Helper: JSON writer
 */
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "service/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}