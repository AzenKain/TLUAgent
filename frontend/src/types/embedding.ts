export type EmbeddingProvider = 'api' | 'onnx';

export type OnnxModelStatus = 'ok' | 'invalid' | 'unvalidated';

export type OnnxDownloadState = 'running' | 'done' | 'error';

export interface EmbeddingRuntimeInfo {
  os: string;
  arch: string;
  lib_present: boolean;
  lib_path: string;
}

export interface EmbeddingSettingsDTO {
  provider: EmbeddingProvider;
  active_onnx_model_id: string | null;
  runtime: EmbeddingRuntimeInfo;
}

export interface UpdateEmbeddingSettingsDTO {
  provider: EmbeddingProvider;
  active_onnx_model_id?: string | null;
}

export interface OnnxModelDTO {
  model_id: string;
  display_name: string;
  dim: number;
  size_bytes: number;
  status: OnnxModelStatus;
  error: string | null;
  validated_at: string | null;
  is_active: boolean;
  sha256: string | null;
}

export interface OnnxModelPresetDTO {
  preset_id: string;
  display_name: string;
  dim: number;
  description: string;
  file_urls: string[];
  approx_size_bytes: number;
}

export interface OnnxRuntimePresetDTO {
  preset_id: string;
  goos: string;
  goarch: string;
  display_name: string;
  url: string;
  archive: string;
  size_bytes: number;
  approx_size_bytes: number;
}

export interface OnnxCatalogDTO {
  models: OnnxModelPresetDTO[];
  runtimes: OnnxRuntimePresetDTO[];
}

export interface StartOnnxDownloadDTO {
  preset_id: string;
}

export interface StartOnnxDownloadResponseDTO {
  job_id: string;
}

export interface OnnxDownloadJobDTO {
  job_id: string;
  preset_id: string;
  state: OnnxDownloadState;
  downloaded_bytes: number;
  total_bytes: number;
  error: string | null;
}

export interface OnnxDownloadStatusDTO {
  jobs: OnnxDownloadJobDTO[];
}

export interface OnnxValidateResponseDTO {
  ok: boolean;
  dim: number;
  latency_ms: number;
  error: string | null;
}
