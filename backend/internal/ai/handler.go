package ai

import (
	"encoding/json"
	"net/http"

	"github.com/404nfid/go-svelte-starter-kit/internal/middleware"
	"github.com/404nfid/go-svelte-starter-kit/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
	v   *validator.Validator
}

func NewHandler(svc *Service, v *validator.Validator) *Handler {
	return &Handler{svc: svc, v: v}
}

// POST /api/ai/chat — SSE stream
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	userID, _ := uuid.Parse(claims.UserID)

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}
	if errs := h.v.Validate(req); errs != nil {
		respondJSON(w, http.StatusUnprocessableEntity, envelope{
			"error": envelope{"code": "validation_failed", "message": "Validation failed", "details": errs},
		})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if err := h.svc.Chat(r.Context(), userID, claims, req, w); err != nil {
		writeSSE(w, map[string]any{"type": "error", "message": err.Error()})
		flush(w)
	}
}

// GET /api/ai/conversations
func (h *Handler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	convs, err := h.svc.ListConversations(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not list conversations")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": convs})
}

// GET /api/ai/conversations/{id}
func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid conversation ID")
		return
	}
	conv, err := h.svc.GetConversation(r.Context(), id, userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "Conversation not found")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": conv})
}

// DELETE /api/ai/conversations/{id}
func (h *Handler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid conversation ID")
		return
	}
	if err := h.svc.DeleteConversation(r.Context(), id, userID); err != nil {
		respondError(w, http.StatusNotFound, "not_found", "Conversation not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- helpers ----

type envelope map[string]any

func respondJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, envelope{
		"error": envelope{"code": code, "message": message},
	})
}

func mustUserID(r *http.Request) uuid.UUID {
	claims := middleware.GetClaims(r)
	id, _ := uuid.Parse(claims.UserID)
	return id
}
