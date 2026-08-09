package whatsapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/internal/middleware"
	"github.com/404nfid/go-svelte-starter-kit/pkg/token"
	"github.com/404nfid/go-svelte-starter-kit/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc          *Service
	validator    *validator.Validator
	tokenManager *token.Manager
}

func NewHandler(svc *Service, v *validator.Validator, tm *token.Manager) *Handler {
	return &Handler{svc: svc, validator: v, tokenManager: tm}
}

// POST /api/admin/whatsapp/sessions
func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if errs := h.validator.Validate(req); len(errs) > 0 {
		respondValidation(w, errs)
		return
	}

	userID := h.callerID(r)
	sess, err := h.svc.CreateSession(r.Context(), req, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "create_failed", "Failed to create session")
		return
	}
	respondJSON(w, http.StatusCreated, envelope{"data": sess})
}

// GET /api/admin/whatsapp/sessions
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.svc.ListSessions(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "list_failed", "Failed to list sessions")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": sessions})
}

// DELETE /api/admin/whatsapp/sessions/:id
func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteSession(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "delete_failed", "Failed to delete session")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Session deleted"}})
}

// PATCH /api/admin/whatsapp/sessions/:id/pause
func (h *Handler) PauseSession(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.PauseSession(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "pause_failed", "Failed to pause session")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Session paused"}})
}

// PATCH /api/admin/whatsapp/sessions/:id/resume
func (h *Handler) ResumeSession(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.ResumeSession(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "resume_failed", "Failed to resume session")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Session resumed"}})
}

// GET /api/admin/whatsapp/sessions/:id/qr  (SSE)
// Auth via ?token= query param because EventSource cannot set Authorization headers.
func (h *Handler) StreamQR(w http.ResponseWriter, r *http.Request) {
	// Manual auth — EventSource can't send headers
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	claims, err := h.tokenManager.VerifyAccess(tokenStr)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	isSuperadmin := false
	for _, role := range claims.Roles {
		if role == "superadmin" {
			isSuperadmin = true
			break
		}
	}
	if !isSuperadmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}

	// Write SSE headers and flush immediately — before Connect() which can take
	// several seconds reaching WhatsApp servers. Without this the proxy sees no
	// response and closes the socket ("socket hang up").
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, hasFlusher := w.(http.Flusher)
	if hasFlusher {
		flusher.Flush()
	}

	// Clear the server's write deadline so the 30 s WriteTimeout doesn't cut
	// the stream while we're waiting for QR codes.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(zeroTime)

	sendEvt := func(evt QREvent) {
		data, _ := json.Marshal(evt)
		fmt.Fprintf(w, "data: %s\n\n", data)
		if hasFlusher {
			flusher.Flush()
		}
	}

	qrCh, err := h.svc.StartQR(r.Context(), id)
	if err != nil {
		sendEvt(QREvent{Type: "error", Message: err.Error()})
		return
	}

	for {
		select {
		case evt, open := <-qrCh:
			if !open {
				return
			}
			sendEvt(evt)
			if evt.Type == "connected" || evt.Type == "timeout" || evt.Type == "error" {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}

var zeroTime = time.Time{}

// POST /api/admin/whatsapp/sessions/:id/pair
func (h *Handler) GetPairingCode(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}

	var req PairRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if errs := h.validator.Validate(req); len(errs) > 0 {
		respondValidation(w, errs)
		return
	}

	code, err := h.svc.GetPairingCode(r.Context(), id, req.Phone)
	if err != nil {
		respondError(w, http.StatusBadRequest, "pair_failed", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"code": code}})
}

// POST /api/admin/whatsapp/messages
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req SendRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if errs := h.validator.Validate(req); len(errs) > 0 {
		respondValidation(w, errs)
		return
	}

	userID := h.callerID(r)
	msg, err := h.svc.Enqueue(r.Context(), req, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "enqueue_failed", "Failed to queue message")
		return
	}
	respondJSON(w, http.StatusAccepted, envelope{"data": msg})
}

// POST /api/admin/whatsapp/messages/batch
func (h *Handler) SendBatch(w http.ResponseWriter, r *http.Request) {
	var req BatchSendRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if errs := h.validator.Validate(req); len(errs) > 0 {
		respondValidation(w, errs)
		return
	}

	userID := h.callerID(r)
	msgs, err := h.svc.EnqueueBatch(r.Context(), req, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "batch_failed", err.Error())
		return
	}
	respondJSON(w, http.StatusAccepted, envelope{
		"data": msgs,
		"meta": envelope{"queued": len(msgs)},
	})
}

// GET /api/admin/whatsapp/messages
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	status := q.Get("status")

	msgs, total, err := h.svc.ListMessages(r.Context(), limit, offset, status)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "list_failed", "Failed to list messages")
		return
	}
	respondJSON(w, http.StatusOK, envelope{
		"data": msgs,
		"meta": envelope{"total": total, "limit": limit, "offset": offset},
	})
}

// ---- helpers ----

func (h *Handler) callerID(r *http.Request) *uuid.UUID {
	claims := middleware.GetClaims(r)
	if claims == nil {
		return nil
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil
	}
	return &id
}

func parseID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	raw := chi.URLParam(r, param)
	id, err := uuid.Parse(raw)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid UUID")
		return uuid.Nil, false
	}
	return id, true
}

// ---- response helpers (package-local copies, same pattern as other domains) ----

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

func respondValidation(w http.ResponseWriter, errs []validator.FieldError) {
	respondJSON(w, http.StatusUnprocessableEntity, envelope{
		"error": envelope{
			"code":    "validation_failed",
			"message": "Validation failed",
			"details": errs,
		},
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return false
	}
	return true
}
