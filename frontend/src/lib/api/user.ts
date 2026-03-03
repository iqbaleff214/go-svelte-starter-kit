import { api } from './client';
import type { ProfileResponse, Session } from '$types';

export const userApi = {
	getProfile() {
		return api.get<ProfileResponse>('/me');
	},

	updateProfile(data: { display_name?: string; bio?: string }) {
		return api.patch<ProfileResponse>('/me', data);
	},

	uploadAvatar(file: File) {
		const form = new FormData();
		form.append('avatar', file);
		// Use fetch directly so the browser sets the correct multipart boundary
		return api.postForm<{ avatar_url: string }>('/me/avatar', form);
	},

	changePassword(current_password: string, new_password: string) {
		return api.post<{ message: string }>('/me/change-password', { current_password, new_password });
	},

	deleteAccount() {
		return api.delete<{ message: string }>('/me');
	},

	listSessions() {
		return api.get<Session[]>('/me/sessions');
	},

	revokeSession(id: string) {
		return api.delete<{ message: string }>(`/me/sessions/${id}`);
	},

	revokeAllOtherSessions() {
		return api.delete<{ message: string }>('/me/sessions');
	}
};
