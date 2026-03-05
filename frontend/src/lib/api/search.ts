import { api } from "./client";

export interface SearchResult {
  type: "user" | "role";
  id: string;
  title: string;
  subtitle: string;
  avatar_url?: string | null;
  href: string;
}

export interface SearchResponse {
  users: SearchResult[];
  roles: SearchResult[];
}

export const searchApi = {
  search(q: string): Promise<SearchResponse> {
    return api.get(`/admin/search?q=${encodeURIComponent(q)}`);
  },
};
