import { api } from './client';
import type { LoginResponse, TwoFASetupResponse, TwoFAConfirmResponse } from '$types';

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
	},

	googleExchange(code: string) {
		return api.post<LoginResponse>('/auth/google/exchange', { code });
	},

	twoFaSetup() {
		return api.post<TwoFASetupResponse>('/auth/2fa/setup');
	},

	twoFaConfirm(code: string) {
		return api.post<TwoFAConfirmResponse>('/auth/2fa/confirm', { code });
	},

	twoFaVerify(pre_auth_token: string, code?: string, backup_code?: string) {
		return api.post<LoginResponse>('/auth/2fa/verify', { pre_auth_token, code, backup_code });
	},

	twoFaDisable(code?: string, backup_code?: string) {
		return api.delete<void>('/auth/2fa', { code, backup_code });
	},

	forgotPassword(email: string) {
		return api.post<{ message: string }>('/auth/forgot-password', { email });
	},

	resetPassword(token: string, password: string) {
		return api.post<{ message: string }>('/auth/reset-password', { token, password });
	},

	verifyEmail(token: string) {
		return api.post<{ message: string }>('/auth/verify-email', { token });
	},

	resendVerification() {
		return api.post<{ message: string }>('/auth/resend-verification');
	}
};
