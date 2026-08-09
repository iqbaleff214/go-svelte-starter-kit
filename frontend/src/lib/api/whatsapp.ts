import { api } from "./client";

export interface WASession {
  id: string;
  name: string;
  phone: string;
  jid: string;
  status: "pending" | "connected" | "disconnected" | "banned";
  paused: boolean;
  last_used_at: string | null;
  sent_today: number;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface WAMessage {
  id: string;
  session_id: string | null;
  recipient: string;
  body: string;
  status: "queued" | "sent" | "failed";
  error?: string;
  queued_at: string;
  sent_at: string | null;
  created_by: string | null;
}

export interface QREvent {
  Type: "qr" | "connected" | "timeout" | "error";
  Code: string;
  Message: string;
}

export interface WAListMessagesParams {
  limit?: number;
  offset?: number;
  status?: string;
}

export const whatsappApi = {
  // Sessions
  createSession(name: string) {
    return api.post<WASession>("/admin/whatsapp/sessions", { name });
  },

  listSessions() {
    return api.get<WASession[]>("/admin/whatsapp/sessions");
  },

  deleteSession(id: string) {
    return api.delete<{ message: string }>(`/admin/whatsapp/sessions/${id}`);
  },

  pauseSession(id: string) {
    return api.patch<{ message: string }>(`/admin/whatsapp/sessions/${id}/pause`);
  },

  resumeSession(id: string) {
    return api.patch<{ message: string }>(`/admin/whatsapp/sessions/${id}/resume`);
  },

  getPairingCode(id: string, phone: string) {
    return api.post<{ code: string }>(`/admin/whatsapp/sessions/${id}/pair`, { phone });
  },

  // QR streaming — returns an EventSource; caller must close it.
  // Token passed as query param because EventSource cannot set Authorization headers.
  streamQR(id: string, onEvent: (evt: QREvent) => void, onError?: () => void): EventSource {
    const token = api.getAccessToken() ?? "";
    const es = new EventSource(`/api/admin/whatsapp/sessions/${id}/qr?token=${encodeURIComponent(token)}`);
    es.onmessage = (e) => {
      try {
        onEvent(JSON.parse(e.data) as QREvent);
      } catch {
        // ignore malformed frames
        console.warn("Malformed QR event:", e.data)
      }
    };
    if (onError) es.onerror = onError;
    return es;
  },

  // Messages
  sendMessage(recipient: string, body: string) {
    return api.post<WAMessage>("/admin/whatsapp/messages", { recipient, body });
  },

  sendBatch(messages: { recipient: string; body: string }[]) {
    return api.post<{ data: WAMessage[]; meta: { queued: number } }>(
      "/admin/whatsapp/messages/batch",
      { messages }
    );
  },

  listMessages(params: WAListMessagesParams = {}) {
    const q = new URLSearchParams();
    if (params.limit) q.set("limit", String(params.limit));
    if (params.offset) q.set("offset", String(params.offset));
    if (params.status) q.set("status", params.status);
    const qs = q.toString();
    return api.get<WAMessage[]>(`/admin/whatsapp/messages${qs ? "?" + qs : ""}`);
  },
};
