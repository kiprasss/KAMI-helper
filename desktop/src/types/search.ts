export type DocumentType = "project_rules" | "legislation";

export interface SearchResult {
  id: string;
  kind: "rule" | "text";
  point: string;
  document_title: string;
  breadcrumbs: string;
  document_type: DocumentType;
  snippet: string;
  punishment: string;
  url: string;
  score: number;
}

export interface SearchResponse {
  data: SearchResult[];
  meta: {
    total: number;
    limit: number;
  };
}
