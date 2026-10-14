import { apiClient } from '@/lib/axios';
import type { CommonResponse } from '@/types/common';
import type {
  TranscriptSummaryResponse,
  ExtractTranscriptImagesResponse,
  BatchSaveTranscriptResponse,
  StudentCourseGradeItem,
} from '@/types/transcript';

export const transcriptService = {
  async getMyTranscript(): Promise<TranscriptSummaryResponse> {
    const res = await apiClient.get<CommonResponse<TranscriptSummaryResponse>>('/transcript/my');
    return res.data.data!;
  },

  async extractTranscriptFromImages(images: string[]): Promise<ExtractTranscriptImagesResponse> {
    const res = await apiClient.post<CommonResponse<ExtractTranscriptImagesResponse>>('/transcript/extract-images', {
      images,
    });
    return res.data.data!;
  },

  async batchSaveTranscript(grades: StudentCourseGradeItem[]): Promise<BatchSaveTranscriptResponse> {
    const res = await apiClient.post<CommonResponse<BatchSaveTranscriptResponse>>('/transcript/batch-save', {
      grades,
    });
    return res.data.data!;
  },
};
