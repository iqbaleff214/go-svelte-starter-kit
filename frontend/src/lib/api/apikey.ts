import { api } from './client';
import type { APIKey, APIKeyCreateResponse, APIKeyLog } from '$types';

export const apikeyApi = {
	list(): Promise<APIKey[]> {
		return api.get<APIKey[]>('/me/api-keys');
	},

	create(req: { name: string; scopes: string[]; expires_at?: string }): Promise<APIKeyCreateResponse> {
		return api.post<APIKeyCreateResponse>('/me/api-keys', req);
	},

	revoke(id: string): Promise<void> {
		return api.delete<void>(`/me/api-keys/${id}`);
	},

	listLogs(
		id: string,
		limit = 20,
		offset = 0
	): Promise<{ logs: APIKeyLog[]; total: number; limit: number; offset: number }> {
		return api.get(`/me/api-keys/${id}/logs?limit=${limit}&offset=${offset}`);
	}
};
