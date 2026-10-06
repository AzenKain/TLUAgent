import { apiClient } from '@/lib/axios';

export interface DocumentDTO {
  id: string;
  doc_code: string;
  type: string;
  title: string;
  domain: string;
  validity_tier: string;
  status: string;
  superseded_by_id?: string;
  department_code: string;
  publish_date: string;
  effective_from?: string;
  effective_to?: string;
  academic_year: string;
  semester: string;
  applicable_cohort: string;
  priority_level: number;
  file_path: string;
  char_count: number;
  created_at: string;
  updated_at: string;
}

export interface DocumentChunkDTO {
  id: string;
  document_id: string;
  rule_id?: string;
  chunk_index: number;
  chunk_type: string;
  content: string;
  status: string;
  domain: string;
  applicable_cohort: string;
  academic_year: string;
  priority_level: number;
  created_at: string;
}

export interface SupersessionAuditLogDTO {
  id: number;
  active_doc_id: string;
  superseded_doc_id: string;
  rule_type: string;
  reason: string;
  created_at: string;
}

export interface DocumentDetailDTO {
  document: DocumentDTO;
  superseded_by?: DocumentDTO;
  supersedes?: DocumentDTO[];
  chunks: DocumentChunkDTO[];
  audit_logs: SupersessionAuditLogDTO[];
}

export interface KGNodeDTO {
  id: string;
  node_type: string;
  name: string;
  code?: string;
  status: string;
  created_at: string;
}

export interface KGEdgeDTO {
  id: string;
  source_node_id: string;
  target_node_id: string;
  relation_type: string;
  weight: number;
  status: string;
  created_at: string;
}

export interface KnowledgeGraphDataDTO {
  nodes: KGNodeDTO[];
  edges: KGEdgeDTO[];
}

export interface KnowledgeStatsDTO {
  total_documents: number;
  active_documents: number;
  superseded_documents: number;
  total_chunks: number;
  total_nodes: number;
  total_edges: number;
}

export interface DocumentListParams {
  domain?: string;
  status?: string;
  cohort?: string;
  search?: string;
  page?: number;
  limit?: number;
}

export const ragAdminService = {
  listDocuments: async (params: DocumentListParams = {}) => {
    const res = await apiClient.get<{
      status: boolean;
      data: {
        documents: DocumentDTO[];
        total: number;
        page: number;
        limit: number;
      };
    }>('/admin/documents', { params });
    return res.data.data;
  },

  getDocumentDetail: async (id: string) => {
    const res = await apiClient.get<{
      status: boolean;
      data: DocumentDetailDTO;
    }>(`/admin/documents/${encodeURIComponent(id)}`);
    return res.data.data;
  },

  updateDocumentStatus: async (
    id: string,
    payload: { status: string; superseded_by_id?: string; reason?: string }
  ) => {
    const res = await apiClient.patch<{
      status: boolean;
      message: string;
    }>(`/admin/documents/${encodeURIComponent(id)}/status`, payload);
    return res.data;
  },

  getKnowledgeGraph: async () => {
    const res = await apiClient.get<{
      status: boolean;
      data: KnowledgeGraphDataDTO;
    }>('/admin/knowledge-graph');
    return res.data.data;
  },

  upsertKGEdge: async (payload: {
    id?: string;
    source_node_id: string;
    target_node_id: string;
    relation_type: string;
    weight?: number;
    status?: string;
  }) => {
    const res = await apiClient.post<{
      status: boolean;
      data: KGEdgeDTO;
      message: string;
    }>('/admin/knowledge-graph/edges', payload);
    return res.data.data;
  },

  deleteKGEdge: async (id: string) => {
    const res = await apiClient.delete<{
      status: boolean;
      message: string;
    }>(`/admin/knowledge-graph/edges/${encodeURIComponent(id)}`);
    return res.data;
  },

  getStats: async () => {
    const res = await apiClient.get<{
      status: boolean;
      data: KnowledgeStatsDTO;
    }>('/admin/rag/stats');
    return res.data.data;
  },

  getAuditLogs: async (params: { page?: number; limit?: number } = {}) => {
    const res = await apiClient.get<{
      status: boolean;
      data: {
        logs: SupersessionAuditLogDTO[];
        total: number;
        page: number;
        limit: number;
      };
    }>('/admin/rag/audit-logs', { params });
    return res.data.data;
  },

  syncIndex: async () => {
    const res = await apiClient.post<{
      status: boolean;
      message: string;
      data: { ingested_documents: number };
    }>('/admin/rag/sync');
    return res.data;
  },
};
