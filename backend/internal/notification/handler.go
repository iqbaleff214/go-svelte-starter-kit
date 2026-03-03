package notification

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/404nfid/go-svelte-starter-kit/internal/middleware"
	"github.com/404nfid/go-svelte-starter-kit/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc       *Service
	validator *validator.Validator
}

func NewHandler(svc *Service, v *validator.Validator) *Handler {
	return &Handler{svc: svc, validator: v}
}

// GET /api/notifications?page=1&limit=20
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", 20)

	resp, err := h.svc.List(r.Context(), userID, page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not fetch notifications")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": resp})
}

// GET /api/notifications/unread-count
func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	count, err := h.svc.UnreadCount(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not fetch count")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": UnreadCountResponse{Count: count}})
}

// PATCH /api/notifications/{id}/read
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid notification ID")
		return
	}

	if err := h.svc.MarkRead(r.Context(), id, userID); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not mark as read")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Marked as read"}})
}

// PATCH /api/notifications/read-all
func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	if err := h.svc.MarkAllRead(r.Context(), userID); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not mark all as read")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "All notifications marked as read"}})
}

// POST /api/notifications/test  — push a test notification to yourself
func (h *Handler) Test(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)

	var body struct {
		Type  string `json:"type"`
		Title string `json:"title"`
		Body  string `json:"body"`
		Link  string `json:"link"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Title == "" {
		body.Type = "info"
		body.Title = "Test notification"
		body.Body = "This is a test notification sent from the API."
	}
	if body.Type == "" {
		body.Type = "info"
	}

	if err := h.svc.Push(r.Context(), userID, body.Type, body.Title, body.Body, body.Link); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not push notification")
		return
	}
	respondJSON(w, http.StatusCreated, envelope{"data": envelope{"message": "Notification pushed"}})
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

func queryInt(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}
