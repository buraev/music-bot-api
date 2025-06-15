package handler

import (
	"encoding/json"
	"music-bot-api/internal/model"
	"music-bot-api/internal/service"
	"net/http"
	"strings"
)

type LinkHandler struct {
	service *service.LinkService
}

func NewLinkHandler(s *service.LinkService) *LinkHandler {
	return &LinkHandler{service: s}
}

func (h *LinkHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	case http.MethodDelete:
		h.handleDelete(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *LinkHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	tagsQuery := r.URL.Query().Get("tags")
	var links []model.Link
	var err error

	if tagsQuery == "" {
		links, err = h.service.GetAllLinks()
	} else {
		tags := strings.Split(tagsQuery, ",")
		links, err = h.service.GetLinksByTags(tags)
	}
	if err != nil {
		http.Error(w, "Failed to get links", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(links)
}

func (h *LinkHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL  string `json:"url"`
		Tags []struct {
			ID string `json:"id"`
		} `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.URL == "" {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	tagIDs := []string{}
	for _, t := range input.Tags {
		if t.ID != "" {
			tagIDs = append(tagIDs, t.ID)
		}
	}

	link, err := h.service.CreateOrUpdateLink(input.URL, tagIDs)
	if err != nil {
		http.Error(w, "Failed to create or update link", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(link)
}

func (h *LinkHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing link ID", http.StatusBadRequest)
		return
	}

	err := h.service.DeleteLink(id)
	if err != nil {
		http.Error(w, "Failed to delete link", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Link deleted"))
}
