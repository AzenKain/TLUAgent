export interface StudentCourseGradeItem {
  id?: string;
  user_id?: string;
  course_code: string;
  course_name: string;
  credits: number;
  score_10: number;
  score_4: number;
  grade_letter: string;
  is_passed: boolean;
  is_retake: boolean;
  term_taken: string;
}

export interface TranscriptDiffItemResponse {
  course_code: string;
  course_name: string;
  credits: number;
  old_score_10: number;
  new_score_10: number;
  old_grade_letter: string;
  new_grade_letter: string;
  old_is_passed: boolean;
  new_is_passed: boolean;
  change_type: 'NEW' | 'UPGRADED' | 'UNCHANGED' | 'LOWER';
}

export interface TranscriptDiffResultResponse {
  has_changes: boolean;
  new_courses_count: number;
  upgraded_courses_count: number;
  unchanged_courses_count: number;
  items: TranscriptDiffItemResponse[];
  confirmation_prompt: string;
}

export interface TranscriptSummaryResponse {
  total_credits_completed: number;
  total_credits_attempted: number;
  cumulative_gpa_10: number;
  cumulative_gpa_4: number;
  total_courses: number;
  passed_courses_count: number;
  failed_courses_count: number;
  grades: StudentCourseGradeItem[];
}

export interface ExtractTranscriptImagesResponse {
  extracted: StudentCourseGradeItem[];
  diff: TranscriptDiffResultResponse;
}

export interface BatchSaveTranscriptResponse {
  summary: TranscriptSummaryResponse;
  message: string;
}
