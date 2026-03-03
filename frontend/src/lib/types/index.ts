export interface User {
	id: string;
	email: string;
	display_name: string;
	avatar_url: string | null;
	email_verified_at: string | null;
	two_fa_enabled: boolean;
	created_at: string;
}

export interface TokenResponse {
	access_token: string;
	token_type: string;
	expires_in: number;
}

export interface LoginResponse {
	user: User;
	token: TokenResponse;
	two_fa_required?: boolean;
	pre_auth_token?: string;
}

export interface Session {
	id: string;
	user_agent: string;
	ip_address: string;
	last_seen_at: string;
	created_at: string;
	is_current: boolean;
}

export interface TwoFASetupResponse {
	secret: string;
	otpauth_url: string;
	qr_code_png: string; // base64-encoded PNG
}

export interface TwoFAConfirmResponse {
	backup_codes: string[];
}

export interface ProfileResponse {
	id: string;
	email: string;
	display_name: string;
	avatar_url: string | null;
	bio: string | null;
	email_verified_at: string | null;
	two_fa_enabled: boolean;
	created_at: string;
}

export interface ApiError {
	code: string;
	message: string;
	details?: FieldError[];
}

export interface FieldError {
	field: string;
	message: string;
}

export interface ApiResponse<T> {
	data: T;
	meta?: Record<string, unknown>;
}

export interface Notification {
	id: string;
	type: 'info' | 'success' | 'warning' | 'alert';
	title: string;
	body: string | null;
	link: string | null;
	read_at: string | null;
	created_at: string;
}

export type ToastType = 'success' | 'error' | 'info' | 'warning';

export interface Toast {
	id: string;
	type: ToastType;
	title: string;
	message?: string;
	duration?: number;
}
