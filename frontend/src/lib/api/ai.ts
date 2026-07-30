import { api } from "./client";
import type { Conversation, ConversationSummary } from "$types";

export const aiApi = {
  // Returns raw Response — caller reads body.getReader() for SSE stream
  streamChat(
    message: string,
    conversationId?: string | null,
    signal?: AbortSignal,
  ): Promise<Response> {
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
