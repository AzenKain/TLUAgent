import { RequirementProgressList } from '@/components/common/RequirementProgressList';
import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  GraduationCap,
  Flame,
  AlertTriangle,
  BookOpen,
  Calendar,
  ExternalLink,
  Sparkles,
  Layers,
} from 'lucide-react';
import type { RoadmapOptimizationResult } from '@/types/curriculum';
import {
  formatDowngradeWarning,
  formatRoadmapRecommendation,
} from '@/utils/roadmapFormatter';

// RoadmapCardProps defines props for RoadmapCard.
export interface RoadmapCardProps {
  roadmap: RoadmapOptimizationResult;
  onViewFullRoadmap?: () => void;
  onQuickAction?: (actionPrompt: string) => void;
}

// RoadmapCard visualizes DAG optimization results directly in chat.
export const RoadmapCard: React.FC<RoadmapCardProps> = ({ roadmap, onViewFullRoadmap, onQuickAction }) => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<number>(0);

  const progressPercent =
    roadmap.total_curriculum_credits > 0
      ? Math.min(100, Math.round((roadmap.accumulated_credits / roadmap.total_curriculum_credits) * 100))
      : 0;

  const currentSemesterPlan = roadmap.semesters && roadmap.semesters.length > 0 ? roadmap.semesters[activeTab] : null;

  return (
    <div className="mt-4 overflow-hidden rounded-2xl border border-blue-200/90 dark:border-blue-900/40 bg-linear-to-br from-white via-blue-50/20 to-slate-50 dark:from-slate-900 dark:via-[#0c1322] dark:to-slate-900/90 shadow-md">
      {/* Header */}
      <div className="border-b border-blue-100 dark:border-slate-800 bg-linear-to-r from-blue-500/10 via-sky-500/5 to-transparent px-5 py-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-tlu-navy text-white shadow-sm dark:bg-tlu-cyan dark:text-slate-950">
              <GraduationCap className="h-5 w-5" />
            </div>
            <div>
              <div className="flex items-center gap-2 flex-wrap">
                <h4 className="text-sm font-bold tracking-tight text-slate-900 dark:text-white">
                  {t('chat.roadmap_card.title')}
                </h4>
                <span className="inline-flex items-center gap-1 rounded-full bg-emerald-100 dark:bg-emerald-950/60 px-2 py-0.5 text-xs font-semibold text-emerald-700 dark:text-emerald-300">
                  <Sparkles className="h-3 w-3" />
                  {t('chat.roadmap_card.preview_badge')}
                </span>
                <span className="inline-flex items-center gap-1 rounded-full bg-blue-100 dark:bg-blue-950/60 px-2 py-0.5 text-xs font-semibold text-blue-700 dark:text-blue-300">
                  DAG
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                {t('chat.roadmap_card.cohort_major', {
                  cohort: roadmap.cohort,
                  major: roadmap.major_name,
                })}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <span className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-white/80 dark:bg-slate-800/80 px-2.5 py-1 text-xs font-semibold text-slate-700 dark:text-slate-300 shadow-2xs">
              <Calendar className="h-3.5 w-3.5 text-blue-500" />
              {t('chat.roadmap_card.target_graduation', {
                term: roadmap.target_graduation_term,
              })}
            </span>
          </div>
        </div>

        {/* Credit Progress Bar */}
        <div className="mt-4">
          <div className="flex items-center justify-between text-xs font-medium text-slate-600 dark:text-slate-300 mb-1.5">
            <span className="flex items-center gap-1.5 font-semibold text-slate-800 dark:text-slate-200">
              <Layers className="h-3.5 w-3.5 text-blue-500" />
              {t('chat.roadmap_card.progress_label')}
            </span>
            <span>
              {t('chat.roadmap_card.credits_progress', {
                accumulated: roadmap.accumulated_credits.toFixed(1),
                total: roadmap.total_curriculum_credits.toFixed(1),
              })}{' '}
              <span className="font-bold text-blue-600 dark:text-blue-400">({progressPercent}%)</span>
            </span>
          </div>
          <div className="h-2 w-full overflow-hidden rounded-full bg-slate-200/80 dark:bg-slate-800">
            <div
              className="h-full rounded-full bg-linear-to-r from-blue-600 via-sky-500 to-teal-400 transition-all duration-500"
              style={{ width: `${progressPercent}%` }}
            />
          </div>
          <div className="mt-1.5 flex justify-end">
            <span className="text-xs text-slate-500 dark:text-slate-400">
              {t('chat.roadmap_card.remaining_credits', {
                credits: roadmap.remaining_credits.toFixed(1),
              })}
            </span>
          </div>
        </div>
      </div>

      <div className="p-5 space-y-4">
        {/* Degree Downgrade Risk Alert */}
        {roadmap.downgrade_risk?.is_at_risk && (
          <div className="rounded-xl border border-amber-300/80 dark:border-amber-900/60 bg-amber-50/90 dark:bg-amber-950/30 p-3.5 text-amber-900 dark:text-amber-200 shadow-2xs">
            <div className="flex items-start gap-2.5">
              <AlertTriangle className="h-4 w-4 shrink-0 text-amber-600 dark:text-amber-400 mt-0.5" />
              <div className="text-xs leading-relaxed">
                <p className="font-bold text-amber-900 dark:text-amber-100">
                  {t('chat.roadmap_card.downgrade_warning_title')}
                </p>
                <p className="mt-0.5 text-amber-800/90 dark:text-amber-300/90">
                  {formatDowngradeWarning(roadmap.downgrade_risk, t)}
                </p>
                <p className="mt-1 font-semibold text-amber-700 dark:text-amber-400">
                  {t('chat.roadmap_card.downgrade_retake_info', {
                    retake: roadmap.downgrade_risk.total_retake_credits.toFixed(1),
                    max: roadmap.downgrade_risk.max_allowed_credits.toFixed(1),
                  })}
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Critical Path Courses */}
        {roadmap.critical_path_courses && roadmap.critical_path_courses.length > 0 && (
          <div>
            <div className="flex items-center gap-2 text-xs font-bold text-slate-800 dark:text-slate-200 mb-2">
              <Flame className="h-4 w-4 text-rose-500 animate-pulse" />
              <span>{t('chat.roadmap_card.critical_path')}</span>
            </div>
            <div className="flex flex-wrap gap-1.5">
              {roadmap.critical_path_courses.map((courseCode) => (
                <span
                  key={courseCode}
                  className="inline-flex items-center gap-1 rounded-lg border border-rose-200 dark:border-rose-900/50 bg-rose-50 dark:bg-rose-950/40 px-2.5 py-1 text-xs font-semibold text-rose-700 dark:text-rose-300"
                >
                  <Flame className="h-3 w-3 text-rose-500" />
                  {courseCode}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Semester Tabs */}
        {roadmap.semesters && roadmap.semesters.length > 0 && (
          <div>
            <div className="flex items-center justify-between gap-2 text-xs font-bold text-slate-800 dark:text-slate-200 mb-2">
              <span className="flex items-center gap-1.5">
                <BookOpen className="h-3.5 w-3.5 text-blue-500" />
                {t('chat.roadmap_card.semesters_heading')}
              </span>
            </div>

            {/* Tab Buttons */}
            {roadmap.semesters.length > 1 && (
              <div className="flex items-center gap-1.5 overflow-x-auto pb-2 scrollbar-thin">
                {roadmap.semesters.map((sem, idx) => (
                  <button
                    key={sem.semester_number}
                    type="button"
                    onClick={() => setActiveTab(idx)}
                    className={`shrink-0 rounded-lg px-3 py-1.5 text-xs font-semibold transition-all cursor-pointer ${
                      activeTab === idx
                        ? 'bg-tlu-navy text-white dark:bg-tlu-cyan dark:text-slate-950 shadow-xs'
                        : 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700'
                    }`}
                  >
                    {sem.is_summer
                      ? t('chat.roadmap_card.summer_semester')
                      : t('chat.roadmap_card.semester_title', { sem: sem.semester_number })}
                  </button>
                ))}
              </div>
            )}

            {/* Active Semester Course Cards */}
            {currentSemesterPlan && (
              <div className="rounded-xl border border-slate-200/80 dark:border-slate-800 bg-white/70 dark:bg-slate-900/60 p-3.5">
                <div className="flex items-center justify-between text-xs mb-3 pb-2 border-b border-slate-100 dark:border-slate-800/80">
                  <span className="font-bold text-slate-800 dark:text-slate-200">
                    {currentSemesterPlan.is_summer
                      ? t('chat.roadmap_card.summer_semester')
                      : t('chat.roadmap_card.semester_title', { sem: currentSemesterPlan.semester_number })}
                  </span>
                  <span className="font-semibold text-blue-600 dark:text-blue-400">
                    {t('chat.roadmap_card.total_credits', {
                      credits: currentSemesterPlan.total_credits.toFixed(1),
                    })}
                  </span>
                </div>

                <div className="space-y-2">
                  {currentSemesterPlan.courses.map((c) => (
                    <div
                      key={c.course_code}
                      className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 rounded-lg border border-slate-100 dark:border-slate-800/60 bg-white dark:bg-slate-800/50 p-2.5 transition-colors hover:border-blue-200 dark:hover:border-blue-900/40"
                    >
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-bold text-xs text-blue-700 dark:text-cyan-400">
                            {c.course_code}
                          </span>
                          <span className="text-xs font-semibold text-slate-800 dark:text-slate-200 truncate">
                            {c.course_name}
                          </span>
                          {c.is_retake && (
                            <span className="inline-flex rounded-md bg-rose-100 dark:bg-rose-950/70 px-1.5 py-0.5 text-[10px] font-bold text-rose-700 dark:text-rose-300">
                              {t('chat.roadmap_card.retake_badge')}
                            </span>
                          )}
                          {c.is_elearning && (
                            <span className="inline-flex rounded-md bg-sky-100 dark:bg-sky-950/70 px-1.5 py-0.5 text-[10px] font-bold text-sky-700 dark:text-sky-300">
                              {t('chat.roadmap_card.elearning_badge')}
                            </span>
                          )}
                        </div>
                        {c.priority_reason && (
                          <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1 line-clamp-1">
                            {c.priority_reason}
                          </p>
                        )}
                      </div>

                      <div className="shrink-0 text-right">
                        <span className="rounded-md bg-slate-100 dark:bg-slate-800 px-2 py-0.5 text-xs font-semibold text-slate-700 dark:text-slate-300">
                          {t('common.credit_value', { credits: c.credits })}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        <RequirementProgressList items={roadmap.requirement_progress} />

        {/* Strategic Recommendations */}
        {roadmap.recommendations && roadmap.recommendations.length > 0 && (
          <div className="rounded-xl border border-blue-100 dark:border-blue-900/30 bg-blue-50/50 dark:bg-blue-950/20 p-3.5">
            <h5 className="text-xs font-bold text-blue-950 dark:text-blue-200 mb-2">
              {t('chat.roadmap_card.recommendations_title')}
            </h5>
            <ul className="space-y-1.5 text-xs text-slate-700 dark:text-slate-300 list-disc list-inside">
              {roadmap.recommendations.map((rec) => (
                <li key={rec} className="leading-relaxed">
                  {formatRoadmapRecommendation(rec, t)}
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* Multi-turn Conversational Negotiation Chips */}
        {onQuickAction && (
          <div className="rounded-xl border border-sky-100 dark:border-sky-900/30 bg-sky-50/40 dark:bg-sky-950/20 p-3.5 space-y-2.5">
            <div className="flex items-center gap-2 text-xs font-bold text-slate-800 dark:text-slate-200">
              <Sparkles className="h-3.5 w-3.5 text-sky-500" />
              <span>{t('chat.roadmap_card.quick_actions_title')}</span>
            </div>
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                onClick={() => onQuickAction(t('chat.roadmap_card.prompt_confirm_save'))}
                className="inline-flex items-center gap-1.5 rounded-lg border border-emerald-200 dark:border-emerald-800 bg-emerald-50 dark:bg-emerald-950/50 hover:bg-emerald-100 dark:hover:bg-emerald-900/50 px-2.5 py-1.5 text-xs font-semibold text-emerald-700 dark:text-emerald-300 transition-colors cursor-pointer shadow-2xs"
              >
                <span>✓</span>
                <span>{t('chat.roadmap_card.chip_confirm_save')}</span>
              </button>

              <button
                type="button"
                onClick={() => onQuickAction(t('chat.roadmap_card.prompt_target_3_5'))}
                className="inline-flex items-center gap-1.5 rounded-lg border border-blue-200 dark:border-blue-800 bg-white dark:bg-slate-800 hover:bg-blue-50 dark:hover:bg-blue-950/40 px-2.5 py-1.5 text-xs font-medium text-slate-700 dark:text-slate-200 transition-colors cursor-pointer shadow-2xs"
              >
                <span>⚡</span>
                <span>{t('chat.roadmap_card.chip_target_3_5')}</span>
              </button>

              <button
                type="button"
                onClick={() => onQuickAction(t('chat.roadmap_card.prompt_target_4_0'))}
                className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 hover:bg-slate-100 dark:hover:bg-slate-700/60 px-2.5 py-1.5 text-xs font-medium text-slate-700 dark:text-slate-200 transition-colors cursor-pointer shadow-2xs"
              >
                <span>📅</span>
                <span>{t('chat.roadmap_card.chip_target_4_0')}</span>
              </button>

              <button
                type="button"
                onClick={() => onQuickAction(t('chat.roadmap_card.prompt_reduce_workload'))}
                className="inline-flex items-center gap-1.5 rounded-lg border border-amber-200 dark:border-amber-800 bg-white dark:bg-slate-800 hover:bg-amber-50 dark:hover:bg-amber-950/40 px-2.5 py-1.5 text-xs font-medium text-slate-700 dark:text-slate-200 transition-colors cursor-pointer shadow-2xs"
              >
                <span>💼</span>
                <span>{t('chat.roadmap_card.chip_reduce_workload')}</span>
              </button>

              <button
                type="button"
                onClick={() => onQuickAction(t('chat.roadmap_card.prompt_summer_semester'))}
                className="inline-flex items-center gap-1.5 rounded-lg border border-teal-200 dark:border-teal-800 bg-white dark:bg-slate-800 hover:bg-teal-50 dark:hover:bg-teal-950/40 px-2.5 py-1.5 text-xs font-medium text-slate-700 dark:text-slate-200 transition-colors cursor-pointer shadow-2xs"
              >
                <span>☀️</span>
                <span>{t('chat.roadmap_card.chip_summer_semester')}</span>
              </button>

              <button
                type="button"
                onClick={() => onQuickAction(t('chat.roadmap_card.prompt_adjust_courses'))}
                className="inline-flex items-center gap-1.5 rounded-lg border border-indigo-200 dark:border-indigo-800 bg-white dark:bg-slate-800 hover:bg-indigo-50 dark:hover:bg-indigo-950/40 px-2.5 py-1.5 text-xs font-medium text-slate-700 dark:text-slate-200 transition-colors cursor-pointer shadow-2xs"
              >
                <span>🔄</span>
                <span>{t('chat.roadmap_card.chip_adjust_courses')}</span>
              </button>
            </div>
          </div>
        )}

        {/* Action Button to Open Full Student Roadmap Portal */}
        {onViewFullRoadmap && (
          <div className="pt-2 border-t border-slate-100 dark:border-slate-800">
            <button
              type="button"
              onClick={onViewFullRoadmap}
              className="w-full inline-flex items-center justify-center gap-2 rounded-xl bg-tlu-navy hover:bg-tlu-navy/90 text-white dark:bg-tlu-cyan dark:text-slate-950 dark:hover:bg-tlu-cyan/90 py-2.5 px-4 text-xs font-bold transition-all shadow-xs cursor-pointer"
            >
              <span>{t('chat.roadmap_card.view_full_portal')}</span>
              <ExternalLink className="h-3.5 w-3.5" />
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
