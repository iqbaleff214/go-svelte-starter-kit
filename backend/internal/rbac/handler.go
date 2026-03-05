package rbac

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/404nfid/go-svelte-starter-kit/internal/email"
	"github.com/404nfid/go-svelte-starter-kit/internal/middleware"
	"github.com/404nfid/go-svelte-starter-kit/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// systemRoles are protected roles that cannot be deleted.
var systemRoles = []string{"superadmin", "admin", "user"}

type Handler struct {
	svc       *Service
	emailRepo *email.Repository
	validator *validator.Validator
}

func NewHandler(svc *Service, emailRepo *email.Repository, v *validator.Validator) *Handler {
	return &Handler{svc: svc, emailRepo: emailRepo, validator: v}
}

// isSuperAdmin returns true if the request's claims include the superadmin role.
func isSuperAdmin(r *http.Request) bool {
	claims := middleware.GetClaims(r)
	return claims != nil && slices.Contains(claims.Roles, "superadmin")
}

// ---- User management ----

// GET /api/admin/users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", 20)
	roleFilter := r.URL.Query().Get("role")

	resp, err := h.svc.ListUsers(r.Context(), page, limit, roleFilter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not fetch users")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": resp})
}

// PATCH /api/admin/users/{userId}/roles/{roleId}
func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid user ID")
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "roleId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid role ID")
		return
	}

	assignedBy := mustUserID(r)
	if err := h.svc.AssignRole(r.Context(), userID, roleID, assignedBy); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not assign role")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Role assigned"}})
}

// DELETE /api/admin/users/{userId}/roles/{roleId}
func (h *Handler) RevokeRole(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid user ID")
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "roleId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid role ID")
		return
	}

	if err := h.svc.RevokeRole(r.Context(), userID, roleID); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not revoke role")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Role revoked"}})
}

// DELETE /api/admin/users/{userId}
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid user ID")
		return
	}
	// Prevent self-deletion
	if userID == mustUserID(r) {
		respondError(w, http.StatusBadRequest, "self_delete", "Cannot delete your own account via admin panel")
		return
	}

	if err := h.svc.DeleteUser(r.Context(), userID); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not delete user")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "User deleted"}})
}

// ---- Role management ----

// GET /api/admin/roles
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.ListRoles(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not fetch roles")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": roles})
}

// POST /api/admin/roles  (superadmin only)
func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	if !isSuperAdmin(r) {
		respondForbidden(w)
		return
	}

	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}
	if errs := h.validator.Validate(req); errs != nil {
		respondValidation(w, errs)
		return
	}

	role, err := h.svc.CreateRole(r.Context(), req.Name, req.Description)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not create role")
		return
	}
	respondJSON(w, http.StatusCreated, envelope{"data": role})
}

// GET /api/admin/roles/{id}
func (h *Handler) GetRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid role ID")
		return
	}

	role, err := h.svc.GetRole(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "Role not found")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": role})
}

// PUT /api/admin/roles/{id}  (superadmin only)
func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	if !isSuperAdmin(r) {
		respondForbidden(w)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid role ID")
		return
	}

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}
	if errs := h.validator.Validate(req); errs != nil {
		respondValidation(w, errs)
		return
	}

	role, err := h.svc.UpdateRole(r.Context(), id, req.Name, req.Description)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not update role")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": role})
}

// DELETE /api/admin/roles/{id}  (superadmin only)
func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	if !isSuperAdmin(r) {
		respondForbidden(w)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid role ID")
		return
	}

	// Prevent deletion of system roles
	role, err := h.svc.GetRole(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "Role not found")
		return
	}
	if slices.Contains(systemRoles, role.Name) {
		respondError(w, http.StatusBadRequest, "system_role", "Cannot delete a system role")
		return
	}

	if err := h.svc.DeleteRole(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not delete role")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{"message": "Role deleted"}})
}

// PUT /api/admin/roles/{id}/permissions  (superadmin only)
func (h *Handler) SetPermissions(w http.ResponseWriter, r *http.Request) {
	if !isSuperAdmin(r) {
		respondForbidden(w)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "Invalid role ID")
		return
	}

	var req SetPermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}

	permIDs := make([]uuid.UUID, 0, len(req.PermissionIDs))
	for _, pidStr := range req.PermissionIDs {
		pid, err := uuid.Parse(pidStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_permission_id", "Invalid permission ID: "+pidStr)
			return
		}
		permIDs = append(permIDs, pid)
	}

	if err := h.svc.SetRolePermissions(r.Context(), id, permIDs); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not update permissions")
		return
	}

	role, err := h.svc.GetRole(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not fetch updated role")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": role})
}

// ---- Search ----

// GET /api/admin/search?q=...
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		respondJSON(w, http.StatusOK, envelope{"data": SearchResponse{
			Users: []*SearchResult{},
			Roles: []*SearchResult{},
		}})
		return
	}
	result, err := h.svc.Search(r.Context(), q)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Search failed")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": result})
}

// ---- Permissions ----

// GET /api/admin/permissions
func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.svc.ListPermissions(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not fetch permissions")
		return
	}
	respondJSON(w, http.StatusOK, envelope{"data": perms})
}

// ---- Email logs ----

// GET /api/admin/emails
func (h *Handler) ListEmailLogs(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	logs, total, err := h.emailRepo.ListLogs(r.Context(), limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "Could not fetch email logs")
		return
	}
	if logs == nil {
		logs = []*email.EmailLog{}
	}
	respondJSON(w, http.StatusOK, envelope{"data": envelope{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	}})
}

// ---- helpers ----

type envelope map[string]any

func respondJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, envelope{"error": envelope{"code": code, "message": message}})
}

func respondForbidden(w http.ResponseWriter) {
	respondError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
}

func respondValidation(w http.ResponseWriter, errs any) {
	respondJSON(w, http.StatusUnprocessableEntity, envelope{
		"error": envelope{"code": "validation_failed", "message": "Validation failed", "details": errs},
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
