import type { ApiError, ApiResponse, LoginResponse } from "$types";

const BASE_URL = "/api";

function getCsrfToken(): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : null;
}

class ApiClient {
  private accessToken: string | null = null;
  private onUnauthorized: (() => void) | null = null;
  private onRefresh: ((user: unknown, token: string) => void) | null = null;
  private isRefreshing = false;
  private refreshQueue: Array<(success: boolean) => void> = [];

  setAccessToken(token: string | null) {
    this.accessToken = token;
  }

  getAccessToken(): string | null {
    return this.accessToken;
  }

  setOnUnauthorized(cb: () => void) {
    this.onUnauthorized = cb;
  }

  setOnRefresh(cb: (user: unknown, token: string) => void) {
    this.onRefresh = cb;
  }

  refresh(): Promise<boolean> {
    return this.tryRefresh();
  }

  handleUnauthorized(): void {
    this.onUnauthorized?.();
  }

  private async tryRefresh(): Promise<boolean> {
    // Deduplicate: queue callers while a refresh is already in flight
    if (this.isRefreshing) {
      return new Promise((resolve) => this.refreshQueue.push(resolve));
    }

    this.isRefreshing = true;
    try {
      const res = await fetch(`${BASE_URL}/auth/refresh`, {
        method: "POST",
        credentials: "include",
      });
      if (!res.ok) throw new Error("refresh failed");

      const body = await res.json();
      const { user, token } = body.data as LoginResponse;
      this.accessToken = token.access_token;
      this.onRefresh?.(user, token.access_token);
      this.refreshQueue.forEach((cb) => cb(true));
      return true;
    } catch {
      this.refreshQueue.forEach((cb) => cb(false));
      return false;
    } finally {
      this.refreshQueue = [];
      this.isRefreshing = false;
    }
  }

  private headers(): HeadersInit {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };
    if (this.accessToken) {
      headers["Authorization"] = `Bearer ${this.accessToken}`;
    }
    const csrf = getCsrfToken();
    if (csrf) {
      headers["X-CSRF-Token"] = csrf;
    }
    return headers;
  }

  private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const res = await fetch(`${BASE_URL}${path}`, {
      ...init,
      credentials: "include",
      headers: {
        ...this.headers(),
        ...(init.headers ?? {}),
      },
    });

    if (!res.ok) {
      if (res.status === 401 && path !== "/auth/refresh") {
        const refreshed = await this.tryRefresh();
        if (refreshed) return this.request<T>(path, init);
        this.onUnauthorized?.();
      }
      let error: ApiError = {
        code: "unknown_error",
        message: "An unexpected error occurred",
      };
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
    return this.request<T>(path, { method: "GET" });
  }

  post<T>(path: string, body?: unknown) {
    return this.request<T>(path, {
      method: "POST",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  }

  patch<T>(path: string, body?: unknown) {
    return this.request<T>(path, {
      method: "PATCH",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  }

  put<T>(path: string, body?: unknown) {
    return this.request<T>(path, {
      method: "PUT",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  }

  delete<T>(path: string, body?: unknown) {
    return this.request<T>(path, {
      method: "DELETE",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  }

  async postForm<T>(path: string, form: FormData): Promise<T> {
    // Must NOT go through request() which always adds Content-Type: application/json.
    // Let the browser set Content-Type automatically so it includes the multipart boundary.
    const makeHeaders = () => {
      const headers: Record<string, string> = {};
      if (this.accessToken) headers["Authorization"] = `Bearer ${this.accessToken}`;
      const csrf = getCsrfToken();
      if (csrf) headers["X-CSRF-Token"] = csrf;
      return headers;
    };

    const doFetch = () =>
      fetch(`${BASE_URL}${path}`, {
        method: "POST",
        body: form,
        credentials: "include",
        headers: makeHeaders(),
      });

    let res = await doFetch();

    if (res.status === 401) {
      const refreshed = await this.tryRefresh();
      if (refreshed) {
        res = await doFetch(); // retry with new token in headers
      } else {
        this.onUnauthorized?.();
      }
    }

    if (!res.ok) {
      let error: ApiError = {
        code: "unknown_error",
        message: "An unexpected error occurred",
      };
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
