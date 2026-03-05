import { api } from "./client";
import type {
  AdminUsersResponse,
  Role,
  Permission,
  EmailLogsResponse,
} from "$types";

export const adminApi = {
  // ---- Users ----
  listUsers(page = 1, limit = 20, role = ""): Promise<AdminUsersResponse> {
    const params = new URLSearchParams({
      page: String(page),
      limit: String(limit),
    });
    if (role) params.set("role", role);
    return api.get(`/admin/users?${params}`);
  },

  assignRole(userId: string, roleId: string): Promise<void> {
    return api.patch(`/admin/users/${userId}/roles/${roleId}`);
  },

  revokeRole(userId: string, roleId: string): Promise<void> {
    return api.delete(`/admin/users/${userId}/roles/${roleId}`);
  },

  deleteUser(userId: string): Promise<void> {
    return api.delete(`/admin/users/${userId}`);
  },

  // ---- Roles ----
  listRoles(): Promise<Role[]> {
    return api.get("/admin/roles");
  },

  getRole(id: string): Promise<Role> {
    return api.get(`/admin/roles/${id}`);
  },

  createRole(data: { name: string; description?: string }): Promise<Role> {
    return api.post("/admin/roles", data);
  },

  updateRole(
    id: string,
    data: { name?: string; description?: string },
  ): Promise<Role> {
    return api.put(`/admin/roles/${id}`, data);
  },

  deleteRole(id: string): Promise<void> {
    return api.delete(`/admin/roles/${id}`);
  },

  setRolePermissions(roleId: string, permissionIds: string[]): Promise<Role> {
    return api.put(`/admin/roles/${roleId}/permissions`, {
      permission_ids: permissionIds,
    });
  },

  // ---- Permissions ----
  listPermissions(): Promise<Permission[]> {
    return api.get("/admin/permissions");
  },

  // ---- Email logs ----
  listEmailLogs(page = 1, limit = 20): Promise<EmailLogsResponse> {
    return api.get(`/admin/emails?page=${page}&limit=${limit}`);
  },
};
