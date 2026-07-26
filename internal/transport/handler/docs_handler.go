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
	userRole := r.Context().Value("userRole").(string)
	requestID := r.Context().Value("requestId").(string)

	data, err := h.service.GetManifest(userRole)
	if err != nil {
		log.Printf("[Handler:GetPublicManifest] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Error manifest not found"))
		return
	}
	utils.WriteJSONSucces(w, http.StatusOK, "public manifest successfully", data)
}

/**
 * GET /docs/{slug}
 */
func (h *DocsHandler) GetPublicDoc(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("requestId").(string)

	slug := chi.URLParam(r, "slug")
	log.Printf("[docs] GET /docs/%s", slug)

	doc, err := h.service.GetDoc(slug, false)
	if err != nil {
		log.Printf("[Handler:GetPublicDoc] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Error doc not found"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "public doc fetched successfully", doc)

}

/**
 * GET /docs/internal/{slug}
 */
func (h *DocsHandler) GetInternalDoc(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("requestId").(string)

	slug := chi.URLParam(r, "slug")
	log.Printf("[docs] GET /docs/internal/%s", slug)

	doc, err := h.service.GetDoc(slug, true)
	if err != nil {
		log.Printf("[Handler:GetInternalDoc] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.WriteJSONError(w, http.StatusInternalServerError, fmt.Errorf("Error doc not found"))
		return
	}

	utils.WriteJSONSucces(w, http.StatusOK, "internal doc fetched successfully", doc)

}
