import { RequirementProgressList } from '@/components/common/RequirementProgressList';
import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import type { TFunction } from 'i18next';
import {
  Compass,
  CheckCircle2,
  AlertTriangle,
  Unlock,
  Lock,
  Flame,
  Search,
  BookOpen,
  TrendingUp,
  Award,
  X,
  Sparkles,
} from 'lucide-react';
import { studentRoadmapService } from '@/services/studentRoadmapService';
import { curriculumAdminService } from '@/services/curriculumAdminService';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { PageHeader } from '@/components/common/PageHeader';
import { RefreshButton } from '@/components/common/RefreshButton';
import {
  formatDowngradeWarning,
  formatRoadmapRecommendation,
} from '@/utils/roadmapFormatter';
import type {
  StudentPersonalRoadmapData,
  StudentRoadmapCourse,
  CourseStatus,
} from '@/types/studentRoadmap';

const extractCreditRequirementDetail = (text: string): string | null => {
  if (!text) return null;
  const m = text.match(/(?:[≥>=]\s*)?(\d{2,3})\s*(?:,\s*)?(?:t[\u00EDi]n\s*ch[\u1EC9i]|t[\u00EDi]n|tc)\b/i);
  if (m) {
    return m[1];
  }
  const m2 = text.match(/([≥>=]\s*\d{2,3})/);
  if (m2) {
    return m2[1].replace('>=', '').replace('≥', '').trim();
  }
  return null;
};

const extractCreditBadge = (text: string, t: TFunction): string | null => {
  const credits = extractCreditRequirementDetail(text);
  if (!credits) return null;
  return t('common.credit_badge', { credits });
};

export const StudentRoadmapPage: React.FC = () => {
  const { t } = useTranslation();
  const [data, setData] = useState<StudentPersonalRoadmapData | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const [cohorts, setCohorts] = useState<string[]>([]);
  const [majors, setMajors] = useState<string[]>([]);
  const [selectedCohort, setSelectedCohort] = useState<string>('');
  const [selectedMajor, setSelectedMajor] = useState<string>('');

  const [statusFilter, setStatusFilter] = useState<'ALL' | CourseStatus>('ALL');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [inspectCourse, setInspectCourse] = useState<StudentRoadmapCourse | null>(null);

  const loadData = useCallback(async (c?: string, m?: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const res = await studentRoadmapService.getMyRoadmap(c, m);
      setData(res);
      if (res?.summary?.cohort && !selectedCohort) {
        setSelectedCohort(res.summary.cohort);
      }
      if (res?.summary?.major_name && !selectedMajor) {
        setSelectedMajor(res.summary.major_name);
      }
    } catch {
      setError(t('student_roadmap.error_load'));
    } finally {
      setIsLoading(false);
    }
  }, [t, selectedCohort, selectedMajor]);

  useEffect(() => {
    let isMounted = true;
    const initialize = async () => {
      try {
        const [cList, mList, roadmapRes] = await Promise.all([
          curriculumAdminService.getCohorts(),
          curriculumAdminService.getMajors(),
          studentRoadmapService.getMyRoadmap(),
        ]);
        if (!isMounted) return;
        setCohorts(cList);
        const uniqueMajors: string[] = Array.from(new Set(mList.map((item) => item.major_name)));
        setMajors(uniqueMajors);
        setData(roadmapRes);
        if (roadmapRes?.summary?.cohort) {
          setSelectedCohort(roadmapRes.summary.cohort);
        }
        if (roadmapRes?.summary?.major_name) {
          setSelectedMajor(roadmapRes.summary.major_name);
        }
      } catch {
        if (isMounted) {
          setError(t('student_roadmap.error_load'));
        }
      } finally {
        if (isMounted) {
          setIsLoading(false);
        }
      }
    };
    initialize();
    return () => {
      isMounted = false;
    };
  }, [t]);

  const handleApplyFilters = () => {
    loadData(selectedCohort, selectedMajor);
  };

  const filteredSemesters = useMemo(() => {
    if (!data?.semesters) return [];
    const query = searchQuery.trim().toUpperCase();

    return data.semesters.map((sem) => {
      const matchingCourses = sem.courses.filter((course) => {
        if (statusFilter !== 'ALL' && course.status !== statusFilter) {
          return false;
        }
        if (query) {
          const matchCode = course.course_code.toUpperCase().includes(query);
          const matchName = course.course_name.toUpperCase().includes(query);
          return matchCode || matchName;
        }
        return true;
      });

      return {
        ...sem,
        courses: matchingCourses,
      };
    });
  }, [data, statusFilter, searchQuery]);

  const summary = data?.summary;
  const progressPct = summary?.progress_percentage ?? 0;

  return (
    <div className="space-y-6 max-w-7xl mx-auto pb-10">
      {/* Page Header */}
      <PageHeader
        icon={<Compass className="h-6 w-6" />}
        title={t('student_roadmap.title')}
        subtitle={t('student_roadmap.subtitle')}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            {cohorts.length > 0 && (
              <select
                aria-label={t('student_roadmap.summary_cohort')}
                value={selectedCohort}
                onChange={(e) => setSelectedCohort(e.target.value)}
                className="rounded-xl border border-slate-300 bg-white px-3 py-2 text-xs font-medium text-slate-700 shadow-xs focus:border-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 min-h-10"
              >
                <option value="">{t('student_roadmap.select_cohort')}</option>
                {cohorts.map((c) => (
                  <option key={c} value={c}>
                    {c}
                  </option>
                ))}
              </select>
            )}

            {majors.length > 0 && (
              <select
                aria-label={t('student_roadmap.summary_major')}
                value={selectedMajor}
                onChange={(e) => setSelectedMajor(e.target.value)}
                className="max-w-45 truncate rounded-xl border border-slate-300 bg-white px-3 py-2 text-xs font-medium text-slate-700 shadow-xs focus:border-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 min-h-10"
              >
                <option value="">{t('student_roadmap.select_major')}</option>
                {majors.map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </select>
            )}

            <button
              onClick={handleApplyFilters}
              className="rounded-xl bg-tlu-navy hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-[#1541B8] px-3.5 py-2 text-xs font-semibold text-white shadow-xs focus:outline-none transition cursor-pointer min-h-10"
            >
              {t('common.apply')}
            </button>

            <RefreshButton
              onClick={() => loadData(selectedCohort, selectedMajor)}
              isLoading={isLoading}
            />
          </div>
        }
      />

      {error && (
        <div className="flex items-center gap-2 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-400">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <LoadingSpinner size="lg" />
        </div>
      ) : (
        <>
          {/* Top KPI Metrics Cards */}
          {summary && (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              {/* Card 1: Completed Credits & Progress */}
              <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-2xs dark:border-blue-900/30 dark:bg-[#111A35]">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                    {t('student_roadmap.credits_completed')}
                  </span>
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-50 text-tlu-blue dark:bg-blue-950/50 dark:text-tlu-cyan">
                    <TrendingUp className="h-4 w-4" />
                  </div>
                </div>
                <div className="mt-2 flex items-baseline gap-2">
                  <span className="text-2xl font-extrabold text-slate-900 dark:text-slate-100">
                    {summary.total_credits_completed}
                  </span>
                  <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
                    / {summary.total_curriculum_credits ?? 0} {t('student_roadmap.credits_unit')}
                  </span>
                </div>
                {/* Progress bar */}
                <div className="mt-3">
                  <div className="flex justify-between text-xs font-medium text-slate-500 dark:text-slate-400">
                    <span>{t('student_roadmap.progress')}</span>
                    <span className="font-semibold text-tlu-blue dark:text-tlu-cyan">{progressPct.toFixed(1)}%</span>
                  </div>
                  <div className="mt-1.5 h-2 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800">
                    <div
                      className="h-full rounded-full bg-tlu-blue transition-all duration-500"
                      style={{ width: `${Math.min(100, Math.max(0, progressPct))}%` }}
                    />
                  </div>
                </div>
              </div>

              {/* Card 2: Cumulative GPA */}
              <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-2xs dark:border-blue-900/30 dark:bg-[#111A35]">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                    {t('student_roadmap.cumulative_gpa')}
                  </span>
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-50 text-emerald-600 dark:bg-emerald-950/50 dark:text-emerald-400">
                    <Award className="h-4 w-4" />
                  </div>
                </div>
                <div className="mt-2 flex items-baseline gap-3">
                  <div>
                    <div className="text-2xl font-extrabold text-tlu-blue dark:text-tlu-cyan">
                      {summary.cumulative_gpa_10 > 0 ? summary.cumulative_gpa_10.toFixed(2) : '--'}
                    </div>
                    <div className="text-[10px] font-medium text-slate-400">{t('student_roadmap.gpa_scale_10')}</div>
                  </div>
                  <div className="border-l border-slate-200 pl-3 dark:border-slate-800">
                    <div className="text-xl font-bold text-slate-700 dark:text-slate-300">
                      {summary.cumulative_gpa_4 > 0 ? summary.cumulative_gpa_4.toFixed(2) : '--'}
                    </div>
                    <div className="text-[10px] font-medium text-slate-400">{t('student_roadmap.gpa_scale_4')}</div>
                  </div>
                </div>
              </div>

              {/* Card 3: Passed Courses */}
              <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-2xs dark:border-blue-900/30 dark:bg-[#111A35]">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                    {t('student_roadmap.courses_passed')}
                  </span>
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-50 text-emerald-600 dark:bg-emerald-950/50 dark:text-emerald-400">
                    <CheckCircle2 className="h-4 w-4" />
                  </div>
                </div>
                <div className="mt-2 text-2xl font-extrabold text-emerald-600 dark:text-emerald-400">
                  {summary.passed_courses_count}
                </div>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  {t('student_roadmap.total_courses_count', { count: summary.total_courses })}
                </p>
              </div>

              {/* Card 4: Retakes & Remaining Credits */}
              <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-2xs dark:border-blue-900/30 dark:bg-[#111A35]">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                    {t('student_roadmap.retakes_and_remaining')}
                  </span>
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-rose-50 text-tlu-red dark:bg-rose-950/50 dark:text-rose-400">
                    <AlertTriangle className="h-4 w-4" />
                  </div>
                </div>
                <div className="mt-2 flex items-baseline gap-2">
                  <span className={`text-2xl font-extrabold ${summary.failed_courses_count > 0 ? 'text-tlu-red dark:text-rose-400' : 'text-slate-900 dark:text-slate-100'}`}>
                    {summary.failed_courses_count}
                  </span>
                  <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
                    {t('student_roadmap.retake_required_unit')}
                  </span>
                </div>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  {t('student_roadmap.credits_remaining')}: <span className="font-semibold text-slate-700 dark:text-slate-300">{summary.total_credits_remaining}</span>
                </p>
              </div>
            </div>
          )}

          {/* DAG Optimization Plan Recommendation Banner */}
          {data?.optimization_plan && (
            <div className="rounded-2xl border border-blue-200/80 bg-linear-to-r from-blue-50/70 via-white to-slate-50/70 p-5 shadow-2xs dark:border-blue-900/40 dark:from-[#111A35] dark:via-[#111A35]/90 dark:to-[#0B1021]">
              <div className="flex items-start gap-3">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-tlu-navy text-white shadow-xs">
                  <Sparkles className="h-5 w-5 text-tlu-cyan" />
                </div>
                <div className="flex-1 space-y-2">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <h3 className="text-sm font-bold text-slate-900 dark:text-slate-100">
                      {t('student_roadmap.opt_plan_title')}
                    </h3>
                    <span className="inline-flex items-center rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-medium text-blue-800 dark:bg-blue-900/60 dark:text-blue-300">
                      {t('student_roadmap.target_grad_sem', { sem: data.optimization_plan.target_graduation_term })}
                    </span>
                  </div>

                  {data.optimization_plan.downgrade_risk?.is_at_risk && (
                    <div className="flex items-center gap-2 rounded-lg bg-rose-100/80 p-2 text-xs font-medium text-rose-800 dark:bg-rose-950/60 dark:text-rose-300">
                      <AlertTriangle className="h-4 w-4 shrink-0 text-rose-600" />
                      <span>{formatDowngradeWarning(data.optimization_plan.downgrade_risk, t)}</span>
                    </div>
                  )}

                  <RequirementProgressList items={data.optimization_plan.requirement_progress} />

                  {data.optimization_plan.recommendations && data.optimization_plan.recommendations.length > 0 && (
                    <ul className="space-y-1 text-xs text-slate-600 dark:text-slate-300">
                      {data.optimization_plan.recommendations.map((rec, idx) => (
                        <li key={idx} className="flex items-start gap-1.5">
                          <span className="font-bold text-tlu-blue">•</span>
                          <span>{formatRoadmapRecommendation(rec, t)}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* Filter Bar & Search */}
          <div className="flex flex-col gap-3 rounded-2xl border border-slate-200 bg-white p-4 shadow-2xs dark:border-blue-900/30 dark:bg-[#111A35] sm:flex-row sm:items-center sm:justify-between">
            {/* Status Pills */}
            <div className="flex flex-wrap items-center gap-1.5">
              {(['ALL', 'PASSED', 'FAILED_RETAKE', 'UNLOCKED', 'LOCKED'] as const).map((st) => {
                const isActive = statusFilter === st;
                return (
                  <button
                    key={st}
                    onClick={() => setStatusFilter(st)}
                    className={`rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors focus:outline-none cursor-pointer ${
                      isActive
                        ? 'bg-tlu-navy text-white dark:bg-tlu-blue'
                        : 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700'
                    }`}
                  >
                    {t(`student_roadmap.filter_${st.toLowerCase()}`)}
                  </button>
                );
              })}
            </div>

            {/* Keyword Search */}
            <div className="relative w-full sm:w-64">
              <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
              <input
                type="text"
                placeholder={t('student_roadmap.search_placeholder')}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full rounded-lg border border-slate-300 bg-slate-50 py-1.5 pl-9 pr-3 text-xs text-slate-900 placeholder-slate-400 focus:border-blue-500 focus:bg-white focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100 dark:placeholder-slate-500"
              />
            </div>
          </div>

          {/* Semester Kanban Columns */}
          {filteredSemesters.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-slate-300 p-12 text-center dark:border-slate-800">
              <BookOpen className="h-10 w-10 text-slate-400 dark:text-slate-600" />
              <p className="mt-3 text-sm font-medium text-slate-600 dark:text-slate-400">
                {t('student_roadmap.empty_courses')}
              </p>
            </div>
          ) : (
            <div className="space-y-6">
              {filteredSemesters.map((sem) => (
                <div
                  key={sem.semester}
                  className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900"
                >
                  {/* Semester Header */}
                  <div className="flex flex-wrap items-center justify-between border-b border-slate-100 pb-3 dark:border-slate-800">
                    <div className="flex items-center gap-2">
                      <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-tlu-navy text-xs font-bold text-white shadow-2xs dark:bg-tlu-blue">
                        {sem.semester}
                      </div>
                      <h2 className="text-base font-bold text-slate-900 dark:text-slate-100">
                        {t('student_roadmap.semester_title', { sem: sem.semester })}
                      </h2>
                    </div>
                    <div className="flex items-center gap-3 text-xs font-medium text-slate-500 dark:text-slate-400">
                      <span>{t('student_roadmap.course_count', { count: sem.courses.length })}</span>
                      <span>•</span>
                      <span>{t('student_roadmap.total_credits_count', { credits: sem.total_credits })}</span>
                    </div>
                  </div>

                  {/* Course Cards Grid */}
                  <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
                    {sem.courses.map((c) => {
                      let borderClass = 'border-slate-200 dark:border-slate-800';
                      let bgClass = 'bg-slate-50/50 dark:bg-slate-800/40';
                      let statusBadge = null;

                      if (c.status === 'PASSED') {
                        borderClass = 'border-emerald-300 dark:border-emerald-800/60';
                        bgClass = 'bg-emerald-50/40 dark:bg-emerald-950/20';
                        statusBadge = (
                          <span className="inline-flex items-center gap-1 rounded-md bg-emerald-100 px-2 py-0.5 text-[11px] font-bold text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-300">
                            <CheckCircle2 className="h-3 w-3 text-emerald-600" />
                            {c.grade_letter ? `${c.grade_letter} (${c.score_10?.toFixed(1)})` : t('student_roadmap.status_passed')}
                          </span>
                        );
                      } else if (c.status === 'FAILED_RETAKE') {
                        borderClass = 'border-rose-300 dark:border-rose-800/60';
                        bgClass = 'bg-rose-50/40 dark:bg-rose-950/20';
                        statusBadge = (
                          <span className="inline-flex items-center gap-1 rounded-md bg-rose-100 px-2 py-0.5 text-[11px] font-bold text-rose-800 dark:bg-rose-900/60 dark:text-rose-300">
                            <AlertTriangle className="h-3 w-3 text-rose-600" />
                            {t('student_roadmap.status_retake')}
                          </span>
                        );
                      } else if (c.status === 'UNLOCKED') {
                        borderClass = 'border-amber-300 dark:border-amber-800/60';
                        bgClass = 'bg-amber-50/30 dark:bg-amber-950/20';
                        statusBadge = (
                          <span className="inline-flex items-center gap-1 rounded-md bg-amber-100 px-2 py-0.5 text-[11px] font-bold text-amber-800 dark:bg-amber-900/60 dark:text-amber-300">
                            <Unlock className="h-3 w-3 text-amber-600" />
                            {t('student_roadmap.status_unlocked')}
                          </span>
                        );
                      } else {
                        statusBadge = (
                          <span className="inline-flex items-center gap-1 rounded-md bg-slate-100 px-2 py-0.5 text-[11px] font-semibold text-slate-600 dark:bg-slate-800 dark:text-slate-400">
                            <Lock className="h-3 w-3 text-slate-400" />
                            {t('student_roadmap.status_locked')}
                          </span>
                        );
                      }

                      const creditBadge = extractCreditBadge(c.prerequisites_text, t);

                      return (
                        <div
                          key={c.id || c.course_code}
                          onClick={() => setInspectCourse(c)}
                          className={`cursor-pointer rounded-xl border p-3.5 transition-all hover:shadow-md ${borderClass} ${bgClass}`}
                        >
                          <div className="flex items-start justify-between gap-2">
                            <div className="flex items-center gap-1.5">
                              <span className="font-mono text-xs font-bold text-slate-900 dark:text-slate-100">
                                {c.course_code}
                              </span>
                              {c.is_critical_path && (
                                <span
                                  title={t('student_roadmap.critical_path_tooltip')}
                                  className="inline-flex items-center text-amber-500"
                                >
                                  <Flame className="h-3.5 w-3.5 fill-amber-500" />
                                </span>
                              )}
                            </div>
                            {statusBadge}
                          </div>

                          <div className="mt-1.5 line-clamp-2 text-xs font-semibold text-slate-800 dark:text-slate-200" title={c.course_name}>
                            {c.course_name}
                          </div>

                          <div className="mt-2.5 flex items-center justify-between text-[11px] text-slate-500 dark:text-slate-400">
                            <span>{c.credits} {t('student_roadmap.credits_unit')}</span>
                            {c.knowledge_block && (
                              <span className="max-w-30 truncate rounded bg-slate-200/60 px-1.5 py-0.5 text-[10px] dark:bg-slate-700/60">
                                {c.knowledge_block}
                              </span>
                            )}
                          </div>

                          {creditBadge && (
                            <div className="mt-2 flex items-center">
                              <span
                                title={c.prerequisites_text}
                                className="inline-flex items-center rounded-md bg-amber-500/10 border border-amber-500/25 px-1.5 py-0.5 text-[10px] font-bold text-amber-800 dark:bg-amber-950/50 dark:border-amber-800/60 dark:text-amber-300"
                              >
                                <Award className="mr-1 h-3 w-3 text-amber-600 dark:text-amber-400" />
                                {creditBadge}
                              </span>
                            </div>
                          )}

                          {c.missing_prereqs && c.missing_prereqs.length > 0 && (
                            <div className="mt-2 text-[10px] text-rose-600 dark:text-rose-400">
                              <span className="font-bold">{t('student_roadmap.missing_prereqs_label')}: </span>
                              <span>{c.missing_prereqs.join(', ')}</span>
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}

      {/* Course Detail Modal */}
      {inspectCourse && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4">
          <div className="w-full max-w-lg rounded-2xl border border-slate-200 bg-white p-6 shadow-2xl dark:border-slate-800 dark:bg-slate-900">
            <div className="flex items-start justify-between border-b border-slate-100 pb-3 dark:border-slate-800">
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-mono text-base font-bold text-blue-600 dark:text-blue-400">
                    {inspectCourse.course_code}
                  </span>
                  {inspectCourse.is_critical_path && (
                    <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-bold text-amber-800 dark:bg-amber-900/60 dark:text-amber-300">
                      <Flame className="h-3 w-3 fill-amber-500" />
                      {t('student_roadmap.critical_path')}
                    </span>
                  )}
                </div>
                <h3 className="mt-1 text-base font-bold text-slate-900 dark:text-slate-100">
                  {inspectCourse.course_name}
                </h3>
              </div>
              <button
                onClick={() => setInspectCourse(null)}
                className="rounded-lg p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-slate-800 dark:hover:text-slate-200"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="mt-4 space-y-3 text-xs">
              <div className="grid grid-cols-2 gap-3 rounded-xl bg-slate-50 p-3 dark:bg-slate-800/50">
                <div>
                  <span className="text-slate-400">{t('student_roadmap.credits_label')}:</span>
                  <p className="font-bold text-slate-800 dark:text-slate-200">{inspectCourse.credits} {t('student_roadmap.credits_unit')}</p>
                </div>
                <div>
                  <span className="text-slate-400">{t('student_roadmap.suggested_sem')}:</span>
                  <p className="font-bold text-slate-800 dark:text-slate-200">{t('student_roadmap.semester_title', { sem: inspectCourse.suggested_semester })}</p>
                </div>
                <div>
                  <span className="text-slate-400">{t('student_roadmap.knowledge_block')}:</span>
                  <p className="font-bold text-slate-800 dark:text-slate-200">{inspectCourse.knowledge_block || '--'}</p>
                </div>
                <div>
                  <span className="text-slate-400">{t('student_roadmap.status_label')}:</span>
                  <p className="font-bold text-slate-800 dark:text-slate-200">{t(`student_roadmap.status_${inspectCourse.status.toLowerCase()}`)}</p>
                </div>
              </div>

              {(() => {
                const creditReq = inspectCourse.prerequisites_text
                  ? extractCreditRequirementDetail(inspectCourse.prerequisites_text)
                  : null;
                return (
                  <>
                    {creditReq && (
                      <div className="flex items-center gap-2 rounded-xl border border-amber-300/80 bg-amber-100/70 p-3 text-xs font-semibold text-amber-900 shadow-2xs dark:border-amber-700/60 dark:bg-amber-950/40 dark:text-amber-200">
                        <Award className="h-4 w-4 shrink-0 text-amber-600 dark:text-amber-400" />
                        <div>
                          <span className="text-[10px] font-bold uppercase tracking-wider text-amber-700/90 dark:text-amber-400 block">
                            {t('curriculum.viz.credit_prereq_heading')}
                          </span>
                          <span>
                            {t('curriculum.viz.credit_prereq_requirement', { count: creditReq })}
                          </span>
                        </div>
                      </div>
                    )}
                    {inspectCourse.prerequisites_text && (
                      <div className="rounded-xl border border-slate-200 p-3 dark:border-slate-800">
                        <span className="font-bold text-slate-700 dark:text-slate-300">
                          {t('student_roadmap.prereqs_condition')}:
                        </span>
                        <p className="mt-1 font-mono text-slate-600 dark:text-slate-400">
                          {inspectCourse.prerequisites_text}
                        </p>
                      </div>
                    )}
                  </>
                );
              })()}

              {inspectCourse.missing_prereqs && inspectCourse.missing_prereqs.length > 0 && (
                <div className="rounded-xl border border-rose-200 bg-rose-50/60 p-3 dark:border-rose-900/50 dark:bg-rose-950/30">
                  <span className="font-bold text-rose-800 dark:text-rose-300">
                    {t('student_roadmap.missing_prereqs_warning')}:
                  </span>
                  <p className="mt-1 font-semibold text-rose-700 dark:text-rose-400">
                    {inspectCourse.missing_prereqs.join(', ')}
                  </p>
                </div>
              )}

              {inspectCourse.score_10 !== undefined && (
                <div className="rounded-xl border border-emerald-200 bg-emerald-50/60 p-3 dark:border-emerald-900/50 dark:bg-emerald-950/30">
                  <span className="font-bold text-emerald-800 dark:text-emerald-300">
                    {t('student_roadmap.grade_record_history')}:
                  </span>
                  <div className="mt-1 flex items-center gap-4 font-semibold text-emerald-900 dark:text-emerald-200">
                    <span>{t('student_roadmap.grade_letter')}: {inspectCourse.grade_letter}</span>
                    <span>{t('student_roadmap.score_10')}: {inspectCourse.score_10}</span>
                    {inspectCourse.term_taken && <span>{t('student_roadmap.term_taken')}: {inspectCourse.term_taken}</span>}
                  </div>
                </div>
              )}
            </div>

            <div className="mt-5 flex justify-end">
              <button
                onClick={() => setInspectCourse(null)}
                className="rounded-lg bg-tlu-navy hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-[#1541B8] px-4 py-2 text-xs font-semibold text-white transition cursor-pointer"
              >
                {t('common.close')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
