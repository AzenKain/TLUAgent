import { apiClient } from '@/lib/axios';
import type { DocumentDetailDTO } from './ragAdminService';

export const documentService = {
  getDocumentDetail: async (identifier: string): Promise<DocumentDetailDTO> => {
    const res = await apiClient.get<{
      status: boolean;
      data: DocumentDetailDTO;
    }>(`/rag/documents/${encodeURIComponent(identifier)}`);
    return res.data.data;
  },
};
