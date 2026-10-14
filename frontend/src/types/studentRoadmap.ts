import type {
  KnowledgeBlockDistribution,
  PrerequisiteEdge,
  RoadmapOptimizationResult,
} from './curriculum';

export type CourseStatus = 'PASSED' | 'FAILED_RETAKE' | 'UNLOCKED' | 'LOCKED';

export interface StudentRoadmapCourse {
  id: string;
  curriculum_id: string;
  cohort: string;
  major_name: string;
  course_code: string;
  course_name: string;
  credits: number;
  weight: number;
  hours: string;
  knowledge_block: string;
  prerequisites_text: string;
  suggested_semester: number;
  is_elearning: boolean;
  status: CourseStatus;
  score_10?: number;
  score_4?: number;
  grade_letter?: string;
  term_taken?: string;
  missing_prereqs?: string[];
  is_critical_path: boolean;
}

export interface StudentSemesterGroup {
  semester: number;
  total_credits: number;
  course_count: number;
  courses: StudentRoadmapCourse[];
}

export interface StudentRoadmapSummary {
  cohort: string;
  major_name: string;
  total_curriculum_credits: number;
  total_credits_completed: number;
  total_credits_remaining: number;
  progress_percentage: number;
  cumulative_gpa_10: number;
  cumulative_gpa_4: number;
  total_courses: number;
  passed_courses_count: number;
  failed_courses_count: number;
}

export interface StudentPersonalRoadmapData {
  summary: StudentRoadmapSummary;
  semesters: StudentSemesterGroup[];
  knowledge_blocks?: KnowledgeBlockDistribution[];
  prerequisite_edges?: PrerequisiteEdge[];
  optimization_plan?: RoadmapOptimizationResult;
}
