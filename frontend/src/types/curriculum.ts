export interface CurriculumStats {
  curriculums: number;
  courses: number;
  substitutions: number;
  prerequisites: number;
}

export interface MajorItem {
  major_name: string;
  major_code: string;
  academic_year: string;
  total_courses: number;
  total_credits: number;
}

export interface CurriculumCourse {
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
  created_at: string;
  updated_at: string;
}

export interface CreateCoursePayload {
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
  is_elearning?: boolean;
}

export interface UpdateCoursePayload {
  course_name: string;
  credits: number;
  weight: number;
  hours: string;
  knowledge_block: string;
  prerequisites_text: string;
  suggested_semester: number;
  is_elearning?: boolean;
}

export interface SemesterGroup {
  semester: number;
  courses: CurriculumCourse[];
  total_credits: number;
}

export interface KnowledgeBlockDistribution {
  knowledge_block: string;
  total_credits: number;
  course_count: number;
  percentage: number;
}

export interface PrerequisiteEdge {
  from_course_code?: string;
  to_course_code?: string;
  from?: string;
  to?: string;
  from_course_name?: string;
  to_course_name?: string;
  from_credits?: number;
  to_credits?: number;
  from_knowledge_block?: string;
  to_knowledge_block?: string;
  min_grade: number;
  condition_type: string;
}

export interface CurriculumVisualizationData {
  cohort?: string;
  major_name?: string;
  curriculum?: {
    id?: string;
    cohort?: string;
    major_name?: string;
    major_code?: string;
    faculty_code?: string;
    academic_year?: string;
  };
  total_courses: number;
  total_credits: number;
  courses?: CurriculumCourse[];
  semesters: SemesterGroup[];
  knowledge_blocks: KnowledgeBlockDistribution[];
  prerequisite_edges: PrerequisiteEdge[];
}

export interface CourseSubstitution {
  id: string;
  original_course_code: string;
  original_course_name: string;
  replacement_course_code: string;
  replacement_course_name: string;
  scope_cohort: string;
  scope_major: string;
  min_cohort?: number;
  max_cohort?: number;
  is_bidirectional: boolean;
  credits_original: number;
  credits_replacement: number;
  rule_type: string;
  rule_note: string;
  source_reference: string;
  created_at: string;
  updated_at: string;
}

export interface CreateSubstitutionPayload {
  original_course_code: string;
  original_course_name: string;
  replacement_course_code: string;
  replacement_course_name: string;
  scope_cohort: string;
  scope_major: string;
  min_cohort?: number;
  max_cohort?: number;
  is_bidirectional: boolean;
  credits_original: number;
  credits_replacement: number;
  rule_type?: string;
  rule_note?: string;
  source_reference?: string;
}

export interface UpdateSubstitutionPayload {
  original_course_name: string;
  replacement_course_name: string;
  scope_cohort: string;
  scope_major: string;
  min_cohort?: number;
  max_cohort?: number;
  is_bidirectional: boolean;
  credits_original: number;
  credits_replacement: number;
  rule_type?: string;
  rule_note?: string;
  source_reference?: string;
}

export interface PaginatedResult<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
}

export interface StudentCourseGrade {
  course_code: string;
  course_name?: string;
  credits?: number;
  grade: string;
  score: number;
  semester_taken?: number;
  is_retake?: boolean;
}

export interface PersonalityQuizAnswers {
  pace_preference: string;
  primary_strength: string;
  delivery_preference: string;
  summer_willingness: boolean;
}

export interface PersonalityQuizResult {
  target_credits_per_semester: number;
  recommended_graduation_term: number;
  competency_radar: Record<string, number>;
  delivery_bias: string;
  summary_notes: string;
}

export interface RoadmapOptimizationRequest {
  cohort: string;
  major_name: string;
  current_semester: number;
  target_graduation_term?: number;
  transcript: StudentCourseGrade[];
  quiz_answers?: PersonalityQuizAnswers;
}

export interface ScheduledCourse {
  course_code: string;
  course_name: string;
  credits: number;
  knowledge_block: string;
  is_elearning: boolean;
  is_retake: boolean;
  priority_reason: string;
  prerequisites_met: string[];
  substituted_from?: string;
  available_substitutes?: string[];
}

export interface ScheduledSemester {
  semester_number: number;
  is_summer: boolean;
  total_credits: number;
  courses: ScheduledCourse[];
  pace_feedback: string;
}

export interface FreeElectiveWaterfall {
  target_credits: number;
  earned_credits: number;
  cross_major_courses: string[];
  extra_languages: string[];
  excess_electives: string[];
  is_fulfilled: boolean;
}

export interface DegreeDowngradeRiskWarning {
  total_retake_credits: number;
  max_allowed_credits: number;
  is_at_risk: boolean;
  warning_message: string;
}

export interface RoadmapOptimizationResult {
  cohort: string;
  major_name: string;
  accumulated_credits: number;
  remaining_credits: number;
  total_curriculum_credits: number;
  current_semester: number;
  target_graduation_term: number;
  semesters: ScheduledSemester[];
  critical_path_courses: string[];
  free_electives: FreeElectiveWaterfall;
  downgrade_risk: DegreeDowngradeRiskWarning;
  recommendations: string[];
}
