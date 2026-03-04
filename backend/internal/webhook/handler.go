package webhook

import (
	"encoding/json"
	"net/http"

	"github.com/404nfid/go-svelte-starter-kit/internal/middleware"
	"github.com/404nfid/go-svelte-starter-kit/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	repo *Repository
	v    *validator.Validator
}

func NewHandler(repo *Repository, v *validator.Validator) *Handler {
	return &Handler{repo: repo, v: v}
}

// POST /api/me/webhooks or POST /api/v1/webhooks
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)

	var req CreateWebhookRequest
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

	wh, err := h.repo.Create(r.Context(), userID, req.URL, req.Events)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not create webhook")
		return
	}
	respondJSON(w, http.StatusCreated, envelope{"data": wh})
}

// GET /api/me/webhooks or GET /api/v1/webhooks
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	whs, err := h.repo.ListByUser(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not list webhooks")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": whs})
}

// DELETE /api/me/webhooks/{id} or DELETE /api/v1/webhooks/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid webhook ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id, userID); err != nil {
		respondError(w, http.StatusNotFound, "not_found", "Webhook not found")
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
