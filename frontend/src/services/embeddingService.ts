import { apiClient } from '@/lib/axios';
import type {
  EmbeddingSettingsDTO,
  OnnxCatalogDTO,
  OnnxDownloadStatusDTO,
  OnnxModelDTO,
  OnnxValidateResponseDTO,
  StartOnnxDownloadResponseDTO,
  UpdateEmbeddingSettingsDTO,
} from '@/types/embedding';

// embeddingService exposes admin endpoints for embedding provider settings and ONNX model management.
export const embeddingService = {
  async getSettings(): Promise<EmbeddingSettingsDTO> {
    const res = await apiClient.get<{ status: boolean; data: EmbeddingSettingsDTO }>('/admin/embedding/settings');
    return res.data.data;
  },

  async updateSettings(data: UpdateEmbeddingSettingsDTO): Promise<EmbeddingSettingsDTO> {
    const res = await apiClient.put<{ status: boolean; data: EmbeddingSettingsDTO }>('/admin/embedding/settings', data);
    return res.data.data;
  },

  async listOnnxModels(): Promise<OnnxModelDTO[]> {
    const res = await apiClient.get<{ status: boolean; data: OnnxModelDTO[] }>('/admin/embedding/onnx/models');
    return res.data.data;
  },

  async getOnnxCatalog(): Promise<OnnxCatalogDTO> {
    const res = await apiClient.get<{ status: boolean; data: OnnxCatalogDTO }>('/admin/embedding/onnx/catalog');
    return res.data.data;
  },

  async startOnnxDownload(presetId: string): Promise<StartOnnxDownloadResponseDTO> {
    const res = await apiClient.post<{ status: boolean; data: StartOnnxDownloadResponseDTO }>(
      '/admin/embedding/onnx/download',
      { preset_id: presetId }
    );
    return res.data.data;
  },

  async getOnnxDownloadStatus(): Promise<OnnxDownloadStatusDTO> {
    const res = await apiClient.get<{ status: boolean; data: OnnxDownloadStatusDTO }>(
      '/admin/embedding/onnx/download/status'
    );
    return res.data.data;
  },

  async validateOnnxModel(modelId: string): Promise<OnnxValidateResponseDTO> {
    const res = await apiClient.post<{ status: boolean; data: OnnxValidateResponseDTO }>(
      `/admin/embedding/onnx/models/${modelId}/validate`
    );
    return res.data.data;
  },

  async activateOnnxModel(modelId: string): Promise<void> {
    await apiClient.post(`/admin/embedding/onnx/models/${modelId}/activate`);
  },

  async deleteOnnxModel(modelId: string): Promise<void> {
    await apiClient.delete(`/admin/embedding/onnx/models/${modelId}`);
  },
};
