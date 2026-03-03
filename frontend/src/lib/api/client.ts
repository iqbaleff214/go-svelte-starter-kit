import type { ApiError, ApiResponse } from '$types';

const BASE_URL = '/api';

class ApiClient {
	private accessToken: string | null = null;

	setAccessToken(token: string | null) {
		this.accessToken = token;
	}

	private headers(): HeadersInit {
		const headers: Record<string, string> = {
			'Content-Type': 'application/json'
		};
		if (this.accessToken) {
			headers['Authorization'] = `Bearer ${this.accessToken}`;
		}
		return headers;
	}

	private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
		const res = await fetch(`${BASE_URL}${path}`, {
			...init,
			credentials: 'include', // include cookies for refresh token
			headers: {
				...this.headers(),
				...(init.headers ?? {})
			}
		});

		if (!res.ok) {
			let error: ApiError = { code: 'unknown_error', message: 'An unexpected error occurred' };
			try {
				const body = await res.json();
				if (body.error) error = body.error;
			} catch {
				// ignore parse errors
			}
			throw error;
		}

		if (res.status === 204) return undefined as T;
		return res.json().then((body: ApiResponse<T>) => body.data);
	}

	get<T>(path: string) {
		return this.request<T>(path, { method: 'GET' });
	}

	post<T>(path: string, body?: unknown) {
		return this.request<T>(path, {
			method: 'POST',
			body: body !== undefined ? JSON.stringify(body) : undefined
		});
	}

	patch<T>(path: string, body?: unknown) {
		return this.request<T>(path, {
			method: 'PATCH',
			body: body !== undefined ? JSON.stringify(body) : undefined
		});
	}

	put<T>(path: string, body?: unknown) {
		return this.request<T>(path, {
			method: 'PUT',
			body: body !== undefined ? JSON.stringify(body) : undefined
		});
	}

	delete<T>(path: string, body?: unknown) {
		return this.request<T>(path, {
			method: 'DELETE',
			body: body !== undefined ? JSON.stringify(body) : undefined
		});
	}

	async postForm<T>(path: string, form: FormData): Promise<T> {
		// Must NOT go through request() which always adds Content-Type: application/json.
		// Let the browser set Content-Type automatically so it includes the multipart boundary.
		const headers: Record<string, string> = {};
		if (this.accessToken) {
			headers['Authorization'] = `Bearer ${this.accessToken}`;
		}
		const res = await fetch(`${BASE_URL}${path}`, {
			method: 'POST',
			body: form,
			credentials: 'include',
			headers
		});

		if (!res.ok) {
			let error: ApiError = { code: 'unknown_error', message: 'An unexpected error occurred' };
			try {
				const body = await res.json();
				if (body.error) error = body.error;
			} catch {
				// ignore parse errors
			}
			throw error;
		}

		if (res.status === 204) return undefined as T;
		return res.json().then((body: ApiResponse<T>) => body.data);
	}
}

export const api = new ApiClient();
