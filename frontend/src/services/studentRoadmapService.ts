import { apiClient } from '@/lib/axios';
import type { CommonResponse } from '@/types/common';
import type { StudentPersonalRoadmapData } from '@/types/studentRoadmap';

export const studentRoadmapService = {
  async getMyRoadmap(cohort?: string, major?: string): Promise<StudentPersonalRoadmapData> {
    const params: Record<string, string> = {};
    if (cohort) params.cohort = cohort;
    if (major) params.major = major;

    const res = await apiClient.get<CommonResponse<StudentPersonalRoadmapData>>('/curriculum/my-roadmap', {
      params,
    });
    return res.data.data!;
  },
};
