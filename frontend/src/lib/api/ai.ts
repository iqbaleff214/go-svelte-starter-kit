import { api } from "./client";
import type { Conversation, ConversationSummary } from "$types";

export const aiApi = {
  // Returns raw Response — caller reads body.getReader() for SSE stream
  async streamChat(
    message: string,
    conversationId?: string | null,
    signal?: AbortSignal,
  ): Promise<Response> {
    const doFetch = () => {
      const token = api.getAccessToken();
      const headers: HeadersInit = { "Content-Type": "application/json" };
      if (token) headers["Authorization"] = `Bearer ${token}`;
      return fetch("/api/ai/chat", {
        method: "POST",
        headers,
        credentials: "include",
        signal,
        body: JSON.stringify({
          message,
          conversation_id: conversationId ?? null,
        }),
      });
    };

    const resp = await doFetch();
    if (resp.status === 401) {
      const refreshed = await api.refresh();
      if (refreshed) return doFetch();
      api.handleUnauthorized();
    }
    return resp;
  },

  listConversations(): Promise<ConversationSummary[]> {
    return api.get<ConversationSummary[]>("/ai/conversations");
  },

  getConversation(id: string): Promise<Conversation> {
    return api.get<Conversation>(`/ai/conversations/${id}`);
  },

  deleteConversation(id: string): Promise<void> {
    return api.delete<void>(`/ai/conversations/${id}`);
  },
};
