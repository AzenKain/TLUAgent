import { apiClient } from '@/lib/axios';
import type { CommonResponse } from '@/types/common';
import type {
  BatchSaveTranscriptResponse,
  StudentCourseGradeItem,
} from '@/types/transcript';

export const transcriptService = {

  async batchSaveTranscript(grades: StudentCourseGradeItem[]): Promise<BatchSaveTranscriptResponse> {
    const res = await apiClient.post<CommonResponse<BatchSaveTranscriptResponse>>('/transcript/batch-save', {
      grades,
    });
    return res.data.data!;
  },
};
