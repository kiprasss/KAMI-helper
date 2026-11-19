export interface KnowledgeStatus {
  ready: boolean;
  refreshing: boolean;
  blocked: boolean;
  last_refresh: number;
  documents: number;
  rules: number;
  entries: number;
  progress: string;
  last_error: string;
}

export interface StatusResponse {
  data: KnowledgeStatus;
}
