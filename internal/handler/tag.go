package handler

import (
	"encoding/json"
	"music-bot-api/internal/service"
	"net/http"
)

type TagHandler struct {
	service *service.TagService
}

func NewTagHandler(s *service.TagService) *TagHandler {
	return &TagHandler{service: s}
}

func (h *TagHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getAllTags(w, r)
	case http.MethodPost:
		h.createTag(w, r)
	case http.MethodPut:
		h.updateTag(w, r)
	case http.MethodDelete:
		h.deleteTag(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *TagHandler) getAllTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.service.GetAllTags()
	if err != nil {
		http.Error(w, "Failed to get tags", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tags)
}

func (h *TagHandler) createTag(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Name == "" {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	tag, err := h.service.CreateTag(input.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tag)
}

func (h *TagHandler) updateTag(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ID == "" || input.Name == "" {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	err := h.service.UpdateTag(&Tag{ID: input.ID, Name: input.Name})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Tag updated"))
}

func (h *TagHandler) deleteTag(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing tag ID", http.StatusBadRequest)
		return
	}
	err := h.service.DeleteTag(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Tag deleted"))
}
