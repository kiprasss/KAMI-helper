import { apiGet } from "./client";
import type { KnowledgeStatus, StatusResponse } from "../types/status";

export async function getStatus(): Promise<KnowledgeStatus> {
  const response = await apiGet<StatusResponse>("/status");
  return response.data;
}
