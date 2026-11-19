import type { DocumentType } from "./search";

export interface RuleDetails {
  id: string;
  kind: "rule" | "text";
  url: string;
  document_title: string;
  breadcrumbs: string;
  document_type: DocumentType;
  point: string;
  text: string;
  punishment: string;
  images: string[] | null;
}

export interface RuleResponse {
  data: RuleDetails;
}
