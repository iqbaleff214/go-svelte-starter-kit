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
