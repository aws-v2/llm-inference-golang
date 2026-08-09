package handler

import (
	"fmt"
	service "llm-inference-service/internal/services"
	"llm-inference-service/internal/utils"
	"log"
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
	userRole := r.Context().Value("role")
	requestID := r.Context().Value("requestId")
	if userRole == "USER" {
		data, err := h.service.GetManifest(false)
		if err != nil {
			log.Printf("[Handler:GetManifest] Service call, requestID %s  error %s", requestID, err.Error())
			utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to get public docs manifest"))
			return
		}

		utils.WriteJSONSucces(w, http.StatusOK, "Fetched documents successfully", map[string]interface{}{
			"service":    data.Service,
			"apiVersion": data.APIVersion,
			"scope":      "public",
			"internal":   []service.DocCategory{},
			"public":     data.Public,
		})
		return
	}

	// For administrative/internal roles, return both public and internal manifests
	publicData, err := h.service.GetManifest(false)
	if err != nil {
		log.Printf("[Handler:GetManifest] Service call, requestID %s  error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to get public docs manifest"))
		return
	}

	internalData, err := h.service.GetManifest(true)
	if err != nil {
		log.Printf("[Handler:GetManifest] Service call, requestID %s  error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("failed to get private docs manifest"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "Fetched documents successfully", map[string]interface{}{
		"service":    chooseString(publicData, internalData),
		"apiVersion": chooseVersion(publicData, internalData),
		"scope":      "internal",
		"internal":   safeCategories(internalData),
		"public":     safeCategories(publicData),
	})
}

/**
 * GET /docs/{slug}
 */
func (h *DocsHandler) GetPublicDoc(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("requestId")
	role := r.Context().Value("role")

	slug := chi.URLParam(r, "slug")
	log.Printf("[docs] GET /docs/%s", slug)

	if role == "USER" {
		doc, err := h.service.GetDoc(slug, false)
		if err != nil {
			log.Printf("[Handler:GetDoc] Service call, requestID %s  error %s", requestID, err.Error())
			utils.WriteJSONError(w, http.StatusNotFound, fmt.Errorf("document not found"))
			return
		}

		utils.WriteJSONSucces(w, http.StatusOK, "Fetched document successfully", doc)
		return
	}

	// First try internal doc, fallback to public doc if not found
	doc, err := h.service.GetDoc(slug, true)
	if err != nil {
		doc, err = h.service.GetDoc(slug, false)
		if err != nil {
			log.Printf("[Handler:GetDoc] Service call, requestID %s  error %s", requestID, err.Error())
			utils.WriteJSONError(w, http.StatusNotFound, fmt.Errorf("document not found"))
			return
		}
	}

	utils.WriteJSONSucces(w, http.StatusOK, "Fetched document successfully", doc)
}

// Helpers to safely extract fields when one of the manifests might be nil
func safeCategories(m *service.DocManifest) []service.DocCategory {
	if m == nil {
		return []service.DocCategory{}
	}
	// prefer Public slice if present, otherwise Internal
	if len(m.Public) > 0 {
		return m.Public
	}
	if len(m.Internal) > 0 {
		return m.Internal
	}
	return []service.DocCategory{}
}

func chooseString(a, b *service.DocManifest) string {
	if a != nil && a.Service != "" {
		return a.Service
	}
	if b != nil {
		return b.Service
	}
	return ""
}

func chooseVersion(a, b *service.DocManifest) string {
	if a != nil && a.APIVersion != "" {
		return a.APIVersion
	}
	if b != nil {
		return b.APIVersion
	}
	return ""
}
