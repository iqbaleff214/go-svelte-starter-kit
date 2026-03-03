import { api } from './client';
import type { LoginResponse } from '$types';

export const authApi = {
	register(body: { display_name: string; email: string; password: string }) {
		return api.post<LoginResponse>('/auth/register', body);
	},

	login(body: { email: string; password: string }) {
		return api.post<LoginResponse>('/auth/login', body);
	},

	refresh() {
		return api.post<LoginResponse>('/auth/refresh');
	},

	logout() {
		return api.post<void>('/auth/logout');
	}
};
