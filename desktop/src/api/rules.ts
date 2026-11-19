import { apiGet } from "./client";
import type { RuleDetails, RuleResponse } from "../types/rules";

export async function getRule(id: string): Promise<RuleDetails> {
  const response = await apiGet<RuleResponse>(`/rules/${encodeURIComponent(id)}`);
  return response.data;
}
