package handler

import (
	"encoding/json"
	"log"
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
	log.Println("[docs] GET /docs/manifest")

	data, err := h.service.GetManifest(false)
	if err != nil {
		log.Printf("[docs] GetPublicManifest error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	log.Println("[docs] GetPublicManifest OK")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": data,
	})
}

/**
 * GET /docs/internal/manifest
 */
func (h *DocsHandler) GetInternalManifest(w http.ResponseWriter, r *http.Request) {
	log.Println("[docs] GET /docs/internal/manifest")

	data, err := h.service.GetManifest(true)
	if err != nil {
		log.Printf("[docs] GetInternalManifest error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	log.Println("[docs] GetInternalManifest OK")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": data,
	})
}

/**
 * GET /docs/{slug}
 */
func (h *DocsHandler) GetPublicDoc(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	log.Printf("[docs] GET /docs/%s", slug)

	doc, err := h.service.GetDoc(slug, false)
	if err != nil {
		log.Printf("[docs] GetPublicDoc slug=%s not found: %v", slug, err)
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not found",
		})
		return
	}

	log.Printf("[docs] GetPublicDoc slug=%s OK", slug)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": doc,
	})
}

/**
 * GET /docs/internal/{slug}
 */
func (h *DocsHandler) GetInternalDoc(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	log.Printf("[docs] GET /docs/internal/%s", slug)

	doc, err := h.service.GetDoc(slug, true)
	if err != nil {
		log.Printf("[docs] GetInternalDoc slug=%s not found: %v", slug, err)
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not found",
		})
		return
	}

	log.Printf("[docs] GetInternalDoc slug=%s OK", slug)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": doc,
	})
}

/**
 * Helper: JSON writer
 */
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}