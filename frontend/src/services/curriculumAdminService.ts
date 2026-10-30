import { apiClient } from '@/lib/axios';
import type { CommonResponse } from '@/types/common';
import type {
  CurriculumStats,
  MajorItem,
  CurriculumCourse,
  CreateCoursePayload,
  UpdateCoursePayload,
  CurriculumVisualizationData,
  CourseSubstitution,
  CreateSubstitutionPayload,
  UpdateSubstitutionPayload,
  PaginatedResult,
} from '@/types/curriculum';

export type RequirementRuleType = 'TOTAL_MIN' | 'CHOOSE_ONE' | 'ELECTIVE_POOL';

export interface RequirementRule {
  rule_type: RequirementRuleType;
  group_key: string;
  min_credits: number;
  course_codes: string;
  note: string;
  source: string;
}

export const curriculumAdminService = {
  async getRequirementRules(cohort: string, major: string): Promise<RequirementRule[]> {
    const res = await apiClient.get<CommonResponse<RequirementRule[]>>('/admin/curriculums/requirements', {
      params: { cohort, major },
    });
    return res.data.data ?? [];
  },

  async replaceRequirementRules(cohort: string, majorName: string, rules: RequirementRule[]): Promise<RequirementRule[]> {
    const res = await apiClient.put<CommonResponse<RequirementRule[]>>('/admin/curriculums/requirements', {
      cohort,
      major_name: majorName,
      rules,
    });
    return res.data.data ?? [];
  },

  async getStats(): Promise<CurriculumStats> {
    const res = await apiClient.get<CommonResponse<CurriculumStats>>('/admin/curriculums/stats');
    return res.data.data!;
  },

  async getCohorts(): Promise<string[]> {
    const res = await apiClient.get<CommonResponse<string[]>>('/admin/curriculums/cohorts');
    return res.data.data!;
  },

  async getMajors(cohort?: string): Promise<MajorItem[]> {
    const res = await apiClient.get<CommonResponse<MajorItem[]>>('/admin/curriculums/majors', {
      params: cohort ? { cohort } : undefined,
    });
    return res.data.data!;
  },

  async getVisualization(cohort: string, major: string): Promise<CurriculumVisualizationData> {
    const res = await apiClient.get<CommonResponse<CurriculumVisualizationData>>(
      '/admin/curriculums/visualization',
      {
        params: { cohort, major },
      }
    );
    return res.data.data!;
  },

  async getCourses(params: {
    cohort?: string;
    major?: string;
    knowledge_block?: string;
    keyword?: string;
    semester?: number;
    is_elearning?: number;
    page?: number;
    limit?: number;
  }): Promise<PaginatedResult<CurriculumCourse>> {
    const res = await apiClient.get<CommonResponse<PaginatedResult<CurriculumCourse>>>(
      '/admin/curriculums/courses',
      { params }
    );
    return res.data.data!;
  },

  async createCourse(data: CreateCoursePayload): Promise<CurriculumCourse> {
    const res = await apiClient.post<CommonResponse<CurriculumCourse>>(
      '/admin/curriculums/courses',
      data
    );
    return res.data.data!;
  },

  async updateCourse(id: string, data: UpdateCoursePayload): Promise<CurriculumCourse> {
    const res = await apiClient.put<CommonResponse<CurriculumCourse>>(
      `/admin/curriculums/courses/${id}`,
      data
    );
    return res.data.data!;
  },

  async deleteCourse(id: string): Promise<void> {
    await apiClient.delete(`/admin/curriculums/courses/${id}`);
  },

  async getSubstitutions(params: {
    cohort?: string;
    major?: string;
    keyword?: string;
    page?: number;
    limit?: number;
  }): Promise<PaginatedResult<CourseSubstitution>> {
    const res = await apiClient.get<CommonResponse<PaginatedResult<CourseSubstitution>>>(
      '/admin/curriculums/substitutions',
      { params }
    );
    return res.data.data!;
  },

  async createSubstitution(data: CreateSubstitutionPayload): Promise<CourseSubstitution> {
    const res = await apiClient.post<CommonResponse<CourseSubstitution>>(
      '/admin/curriculums/substitutions',
      data
    );
    return res.data.data!;
  },

  async updateSubstitution(
    id: string,
    data: UpdateSubstitutionPayload
  ): Promise<CourseSubstitution> {
    const res = await apiClient.put<CommonResponse<CourseSubstitution>>(
      `/admin/curriculums/substitutions/${id}`,
      data
    );
    return res.data.data!;
  },

  async deleteSubstitution(id: string): Promise<void> {
    await apiClient.delete(`/admin/curriculums/substitutions/${id}`);
  },

};
