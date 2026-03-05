import { writable } from 'svelte/store';
import { PUBLIC_API_URL } from '$env/static/public';
import { notificationStore } from './notification';
import { toast } from './toast';

const notifToast: Record<string, (title: string, msg?: string) => void> = {
	info: (t, m) => toast.info(t, m),
	success: (t, m) => toast.success(t, m),
	warning: (t, m) => toast.warning(t, m),
	alert: (t, m) => toast.error(t, m)
};

export const wsConnected = writable<boolean>(false);

let socket: WebSocket | null = null;
let shouldReconnect = false;
let reconnectDelay = 1000; // ms; doubles each attempt, capped at 30s
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let currentToken = '';

/**
 * Build the WebSocket URL. When PUBLIC_API_URL is set (e.g. http://localhost:8080
 * in dev, https://api.example.com in production), we connect directly to the
 * backend — bypassing the Vite dev-server proxy, which shares its port with HMR
 * WebSockets and can silently drop upgrade requests. The backend CORS already
 * allows the frontend origin, so cross-origin WS works fine.
 */
function buildURL(token: string): string {
	const base = PUBLIC_API_URL
		? PUBLIC_API_URL.replace(/^http/, 'ws')
		: `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}`;
	return `${base}/api/ws?token=${encodeURIComponent(token)}`;
}

function clearTimer() {
	if (reconnectTimer !== null) {
		clearTimeout(reconnectTimer);
		reconnectTimer = null;
	}
}

function open(token: string) {
	if (socket) {
		socket.onclose = null; // suppress reconnect from the old socket
		socket.close();
	}

	const url = buildURL(token);
	console.log('[WS] connecting to:', url.replace(/token=[^&]+/, 'token=<redacted>'));
	socket = new WebSocket(url);

	socket.onopen = () => {
		wsConnected.set(true);
		reconnectDelay = 1000; // reset backoff on successful connect
		console.log('[WS] connected:', socket?.url);
	};

	socket.onmessage = (event: MessageEvent) => {
		console.log('[WS] message received:', event.data);
		try {
			const msg = JSON.parse(event.data as string);
			if (msg?.type === 'notification' && msg.data) {
				const n = msg.data;
				notificationStore.add(n);
				const showToast = notifToast[n.type] ?? toast.info;
				showToast(n.title, n.body ?? undefined);
			}
		} catch {
			// ignore malformed frames
			console.warn('[WS] Received malformed WS message:', event.data);
		}
	};

	socket.onerror = (e) => {
		console.error('[WS] error:', e);
		// onclose will fire right after; handle reconnect there
	};

	socket.onclose = (e) => {
		console.log('[WS] closed, code:', e.code, 'reason:', e.reason, 'wasClean:', e.wasClean);
		wsConnected.set(false);
		socket = null;
		if (!shouldReconnect) return;
		clearTimer();
		reconnectTimer = setTimeout(() => {
			reconnectDelay = Math.min(reconnectDelay * 2, 30_000);
			open(currentToken);
		}, reconnectDelay);
	};
}

export const wsStore = {
	connect(token: string) {
		currentToken = token;
		shouldReconnect = true;
		reconnectDelay = 1000;
		clearTimer();
		open(token);
	},

	disconnect() {
		shouldReconnect = false;
		clearTimer();
		if (socket) {
			socket.onclose = null;
			socket.close();
			socket = null;
		}
		wsConnected.set(false);
	}
};
