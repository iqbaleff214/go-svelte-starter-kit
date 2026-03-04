export interface User {
	id: string;
	email: string;
	display_name: string;
	avatar_url: string | null;
	email_verified_at: string | null;
	two_fa_enabled: boolean;
	roles: string[];
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

export interface NotificationListResponse {
	notifications: Notification[];
	total: number;
	page: number;
	limit: number;
}

export interface UnreadCountResponse {
	count: number;
}

export interface Permission {
	id: string;
	name: string;
	description: string | null;
	created_at: string;
}

export interface Role {
	id: string;
	name: string;
	description: string | null;
	permissions: Permission[];
	created_at: string;
	updated_at: string;
}

export interface AdminUser {
	id: string;
	email: string;
	display_name: string;
	avatar_url: string | null;
	email_verified_at: string | null;
	two_fa_enabled: boolean;
	roles: string[];
	created_at: string;
}

export interface AdminUsersResponse {
	users: AdminUser[];
	total: number;
	page: number;
	limit: number;
}

export interface EmailLog {
	id: string;
	user_id: string | null;
	template: string;
	recipient: string;
	status: string;
	error: string;
	attempts: number;
	sent_at: string | null;
	created_at: string;
}

export interface EmailLogsResponse {
	logs: EmailLog[];
	total: number;
	page: number;
	limit: number;
}

export interface ChatMessage {
	role: 'user' | 'assistant';
	content: string;
}

export interface Conversation {
	id: string;
	title: string;
	model: string;
	messages: ChatMessage[];
	token_usage: number;
	created_at: string;
	updated_at: string;
}

export interface ConversationSummary {
	id: string;
	title: string;
	model: string;
	token_usage: number;
	updated_at: string;
	created_at: string;
}

export interface APIKey {
	id: string;
	name: string;
	key_prefix: string;
	scopes: string[];
	last_used_at: string | null;
	expires_at: string | null;
	revoked_at: string | null;
	created_at: string;
}

export interface APIKeyCreateResponse extends APIKey {
	key: string; // plaintext — shown once only
}

export interface APIKeyLog {
	id: string;
	api_key_id: string;
	method: string;
	path: string;
	status_code: number;
	ip: string;
	created_at: string;
}

export interface Webhook {
	id: string;
	url: string;
	events: string[];
	active: boolean;
	created_at: string;
	updated_at: string;
}

export type ToastType = 'success' | 'error' | 'info' | 'warning';

export interface Toast {
	id: string;
	type: ToastType;
	title: string;
	message?: string;
	duration?: number;
}
