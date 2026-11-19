import { apiGet } from "./client";
import type { SearchResponse } from "../types/search";

interface SearchParams {
  query: string;
  server: string;
  limit?: number;
}

export async function searchRules({
  query,
  server,
  limit = 30,
}: SearchParams): Promise<SearchResponse> {
  const params = new URLSearchParams({
    q: query,
    server,
    limit: String(limit),
  });

  return apiGet<SearchResponse>(`/search?${params.toString()}`);
}
