import React, { useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  BookOpen,
  Plus,
  Pencil,
  Trash2,
  Search,
  ArrowLeftRight,
  ChevronLeft,
  ChevronRight,
  GitBranch,
  GraduationCap,
  RotateCcw,
  X,
} from 'lucide-react';
import { curriculumAdminService } from '@/services/curriculumAdminService';
import { CourseModal } from './CourseModal';
import { SubstitutionModal } from './SubstitutionModal';
import { CurriculumRoadmapView } from './CurriculumRoadmapView';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { ConfirmModal } from '@/components/common/ConfirmModal';
import { PermissionGate } from '@/components/common/PermissionGate';
import { PageHeader } from '@/components/common/PageHeader';
import { RefreshButton } from '@/components/common/RefreshButton';
import { StatCard } from '@/components/common/StatCard';
import type {
  CurriculumCourse,
  CourseSubstitution,
  CreateCoursePayload,
  UpdateCoursePayload,
  CreateSubstitutionPayload,
  UpdateSubstitutionPayload,
} from '@/types/curriculum';

type TabType = 'courses' | 'visualization' | 'substitutions';

const renderCohortBadge = (sub: CourseSubstitution) => {
  const min = sub.min_cohort && sub.min_cohort > 0 ? sub.min_cohort : 1;
  const max = sub.max_cohort && sub.max_cohort > 0 ? sub.max_cohort : 999;

  if (min <= 1 && max >= 999) {
    return (
      <span className="inline-flex items-center rounded-full border border-slate-200 bg-slate-100 px-2.5 py-0.5 text-xs font-semibold text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300">
        ALL
      </span>
    );
  }

  if (min === max) {
    return (
      <div className="flex flex-col items-start gap-0.5">
        <span className="inline-flex items-center rounded-full border border-indigo-300 bg-indigo-50 px-2.5 py-0.5 text-xs font-semibold text-indigo-800 dark:border-indigo-800/60 dark:bg-indigo-950/40 dark:text-indigo-300">
          K{min}
        </span>
        <span className="font-mono text-[10px] text-slate-400">
          (Min: {min}, Max: {max})
        </span>
      </div>
    );
  }

  const maxLabel = max >= 999 ? '∞' : `K${max}`;
  const displayMax = max >= 999 ? '∞' : String(max);
  return (
    <div className="flex flex-col items-start gap-0.5">
      <span className="inline-flex items-center rounded-full border border-purple-300 bg-purple-50 px-2.5 py-0.5 text-xs font-semibold text-purple-800 dark:border-purple-800/60 dark:bg-purple-950/40 dark:text-purple-300">
        K{min} - {maxLabel}
      </span>
      <span className="font-mono text-[10px] text-slate-400">
        (Min: {min}, Max: {displayMax})
      </span>
    </div>
  );
};

const formatPrerequisitesDisplay = (text?: string): string => {
  if (!text) return '-';
  const cleaned = text
    .replace(/,\s*,+/g, ',')
    .replace(new RegExp(',\\s*ho\\u1EB7c\\s*,?', 'gi'), ' ho\u1EB7c ')
    .replace(/[\s,;]+$/g, '')
    .replace(new RegExp('\\s+(?:ho\\u1EB7c|or)$', 'i'), '')
    .replace(/[\s,;]+$/g, '')
    .trim();
  return cleaned || '-';
};

// CurriculumManagePage coordinates full administrative management and visual exploration of courses and substitutions.
export const CurriculumManagePage: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [activeTab, setActiveTab] = useState<TabType>('courses');
  const [selectedCohort, setSelectedCohort] = useState<string>('K36');
  const [selectedMajor, setSelectedMajor] = useState<string>('');

  const [courseSearch, setCourseSearch] = useState<string>('');
  const [blockFilter, setBlockFilter] = useState<string>('');
  const [semesterFilter, setSemesterFilter] = useState<number>(0);
  const [elearningFilter, setElearningFilter] = useState<number>(-1);
  const [coursePage, setCoursePage] = useState<number>(1);
  const courseLimit = 20;

  const [subSearch, setSubSearch] = useState<string>('');
  const [subScopeCohort, setSubScopeCohort] = useState<string>('ALL');
  const [subScopeMajor, setSubScopeMajor] = useState<string>('ALL');
  const [subPage, setSubPage] = useState<number>(1);
  const subLimit = 20;

  const [isCourseModalOpen, setIsCourseModalOpen] = useState(false);
  const [editingCourse, setEditingCourse] = useState<CurriculumCourse | null>(null);
  const [deletingCourse, setDeletingCourse] = useState<CurriculumCourse | null>(null);

  const [isSubModalOpen, setIsSubModalOpen] = useState(false);
  const [editingSub, setEditingSub] = useState<CourseSubstitution | null>(null);
  const [deletingSub, setDeletingSub] = useState<CourseSubstitution | null>(null);

  const [toastMessage, setToastMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const showToast = (type: 'success' | 'error', text: string) => {
    setToastMessage({ type, text });
    setTimeout(() => setToastMessage(null), 4000);
  };

  const { data: statsData, isFetching: isStatsFetching, refetch: refetchStats } = useQuery({
    queryKey: ['curriculumStats'],
    queryFn: () => curriculumAdminService.getStats(),
  });

  const { data: cohortsData, refetch: refetchCohorts } = useQuery({
    queryKey: ['curriculumCohorts'],
    queryFn: () => curriculumAdminService.getCohorts(),
  });

  const { data: majorsData, refetch: refetchMajors } = useQuery({
    queryKey: ['curriculumMajors', selectedCohort],
    queryFn: () => curriculumAdminService.getMajors(selectedCohort),
  });

  const { data: allMajorsData } = useQuery({
    queryKey: ['curriculumAllMajors'],
    queryFn: () => curriculumAdminService.getMajors(),
  });

  const uniqueMajors = useMemo(() => {
    if (!allMajorsData) return [];
    const seen = new Set<string>();
    return allMajorsData.filter((m) => {
      const key = (m.major_code || m.major_name).toUpperCase();
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
  }, [allMajorsData]);

  const effectiveMajor =
    selectedMajor || (majorsData && majorsData.length > 0 ? majorsData[0].major_name : '');

  const {
    data: coursesData,
    isLoading: isCoursesLoading,
    refetch: refetchCourses,
  } = useQuery({
    queryKey: [
      'curriculumCourses',
      selectedCohort,
      effectiveMajor,
      blockFilter,
      semesterFilter,
      elearningFilter,
      courseSearch,
      coursePage,
    ],
    queryFn: () =>
      curriculumAdminService.getCourses({
        cohort: selectedCohort,
        major: effectiveMajor,
        knowledge_block: blockFilter,
        semester: semesterFilter > 0 ? semesterFilter : undefined,
        is_elearning: elearningFilter === -1 ? undefined : elearningFilter,
        keyword: courseSearch.trim(),
        page: coursePage,
        limit: courseLimit,
      }),
  });

  const {
    data: vizData,
    isLoading: isVizLoading,
    refetch: refetchViz,
  } = useQuery({
    queryKey: ['curriculumVisualization', selectedCohort, effectiveMajor],
    queryFn: () => curriculumAdminService.getVisualization(selectedCohort, effectiveMajor),
    enabled: Boolean(selectedCohort && effectiveMajor && activeTab === 'visualization'),
  });

  const effectiveSubCohort = subScopeCohort === 'ALL' || !subScopeCohort ? '' : subScopeCohort;
  const effectiveSubMajor = subScopeMajor === 'ALL' || !subScopeMajor ? '' : subScopeMajor;

  const {
    data: subsData,
    isLoading: isSubsLoading,
    refetch: refetchSubs,
  } = useQuery({
    queryKey: ['curriculumSubstitutions', effectiveSubCohort, effectiveSubMajor, subSearch, subPage],
    queryFn: () =>
      curriculumAdminService.getSubstitutions({
        cohort: effectiveSubCohort,
        major: effectiveSubMajor,
        keyword: subSearch.trim(),
        page: subPage,
        limit: subLimit,
      }),
    enabled: activeTab === 'substitutions',
  });

  const saveCourseMutation = useMutation({
    mutationFn: async (payload: CreateCoursePayload | UpdateCoursePayload) => {
      if (editingCourse) {
        return curriculumAdminService.updateCourse(editingCourse.id, payload as UpdateCoursePayload);
      }
      return curriculumAdminService.createCourse(payload as CreateCoursePayload);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['curriculumCourses'] });
      queryClient.invalidateQueries({ queryKey: ['curriculumStats'] });
      queryClient.invalidateQueries({ queryKey: ['curriculumVisualization'] });
      showToast(
        'success',
        editingCourse
          ? t('curriculum.courses.update_success')
          : t('curriculum.courses.create_success')
      );
    },
    onError: () => {
      showToast('error', t('curriculum.courses.save_error'));
    },
  });

  const deleteCourseMutation = useMutation({
    mutationFn: (id: string) => curriculumAdminService.deleteCourse(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['curriculumCourses'] });
      queryClient.invalidateQueries({ queryKey: ['curriculumStats'] });
      queryClient.invalidateQueries({ queryKey: ['curriculumVisualization'] });
      setDeletingCourse(null);
      showToast('success', t('curriculum.courses.delete_success'));
    },
    onError: () => {
      showToast('error', t('curriculum.courses.delete_error'));
    },
  });

  const saveSubMutation = useMutation({
    mutationFn: async (payload: CreateSubstitutionPayload | UpdateSubstitutionPayload) => {
      if (editingSub) {
        return curriculumAdminService.updateSubstitution(
          editingSub.id,
          payload as UpdateSubstitutionPayload
        );
      }
      return curriculumAdminService.createSubstitution(payload as CreateSubstitutionPayload);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['curriculumSubstitutions'] });
      queryClient.invalidateQueries({ queryKey: ['curriculumStats'] });
      showToast(
        'success',
        editingSub
          ? t('curriculum.substitutions.update_success')
          : t('curriculum.substitutions.create_success')
      );
    },
    onError: () => {
      showToast('error', t('curriculum.substitutions.save_error'));
    },
  });

  const deleteSubMutation = useMutation({
    mutationFn: (id: string) => curriculumAdminService.deleteSubstitution(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['curriculumSubstitutions'] });
      queryClient.invalidateQueries({ queryKey: ['curriculumStats'] });
      setDeletingSub(null);
      showToast('success', t('curriculum.substitutions.delete_success'));
    },
    onError: () => {
      showToast('error', t('curriculum.substitutions.delete_error'));
    },
  });

  const handleRefreshAll = () => {
    refetchStats();
    refetchCohorts();
    refetchMajors();
    if (activeTab === 'courses') refetchCourses();
    if (activeTab === 'visualization') refetchViz();
    if (activeTab === 'substitutions') refetchSubs();
  };

  const totalCoursePages = Math.ceil((coursesData?.total || 0) / courseLimit) || 1;
  const totalSubPages = Math.ceil((subsData?.total || 0) / subLimit) || 1;

  const vizBlocks = vizData?.knowledge_blocks;
  const courseItems = coursesData?.items;

  const availableBlocks = useMemo(() => {
    const blockSet = new Set<string>();
    if (vizBlocks) {
      vizBlocks.forEach((b) => {
        if (b.knowledge_block) blockSet.add(b.knowledge_block);
      });
    }
    if (courseItems) {
      courseItems.forEach((c) => {
        if (c.knowledge_block) blockSet.add(c.knowledge_block);
      });
    }
    return Array.from(blockSet);
  }, [vizBlocks, courseItems]);

  return (
    <div className="space-y-6 max-w-7xl mx-auto pb-10">
      {/* Toast Alert */}
      {toastMessage && (
        <div
          className={`fixed bottom-5 right-5 z-50 flex items-center rounded-xl p-4 shadow-xl transition-all ${
            toastMessage.type === 'success'
              ? 'bg-emerald-600 text-white'
              : 'bg-rose-600 text-white'
          }`}
        >
          <span className="text-sm font-medium">{toastMessage.text}</span>
        </div>
      )}

      {/* Header and Top Actions */}
      <PageHeader
        icon={<GraduationCap className="h-6 w-6" />}
        title={t('curriculum.title')}
        subtitle={t('curriculum.subtitle')}
        actions={
          <RefreshButton
            onClick={handleRefreshAll}
            isLoading={isStatsFetching || isCoursesLoading || isVizLoading || isSubsLoading}
          />
        }
      />

      {/* Quick Stats Grid */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4 sm:gap-4">
        <StatCard
          icon={<GraduationCap className="h-5 w-5" />}
          label={t('curriculum.stats.programs')}
          value={statsData?.curriculums?.toLocaleString() ?? 0}
          variant="blue"
        />
        <StatCard
          icon={<BookOpen className="h-5 w-5" />}
          label={t('curriculum.stats.courses')}
          value={statsData?.courses?.toLocaleString() ?? 0}
          variant="sky"
        />
        <StatCard
          icon={<ArrowLeftRight className="h-5 w-5" />}
          label={t('curriculum.stats.substitutions')}
          value={statsData?.substitutions?.toLocaleString() ?? 0}
          variant="rose"
        />
        <StatCard
          icon={<GitBranch className="h-5 w-5" />}
          label={t('curriculum.stats.prerequisites')}
          value={statsData?.prerequisites?.toLocaleString() ?? 0}
          variant="emerald"
        />
      </div>

      {/* Primary Module Navigation Tabs */}
      <div className="flex border-b border-slate-200 dark:border-slate-800 overflow-x-auto scrollbar-none">
        <button
          type="button"
          onClick={() => setActiveTab('courses')}
          className={`flex items-center gap-2 border-b-2 px-4 py-3 text-sm font-semibold transition-colors shrink-0 cursor-pointer ${
            activeTab === 'courses'
              ? 'border-tlu-navy text-tlu-navy dark:border-tlu-blue dark:text-tlu-cyan'
              : 'border-transparent text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200'
          }`}
        >
          <BookOpen className="h-4 w-4" />
          <span>{t('curriculum.tabs.courses')}</span>
        </button>

        <button
          type="button"
          onClick={() => setActiveTab('visualization')}
          className={`flex items-center gap-2 border-b-2 px-4 py-3 text-sm font-semibold transition-colors shrink-0 cursor-pointer ${
            activeTab === 'visualization'
              ? 'border-tlu-navy text-tlu-navy dark:border-tlu-blue dark:text-tlu-cyan'
              : 'border-transparent text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200'
          }`}
        >
          <GraduationCap className="h-4 w-4" />
          <span>{t('curriculum.tabs.visualization')}</span>
        </button>

        <button
          type="button"
          onClick={() => setActiveTab('substitutions')}
          className={`flex items-center gap-2 border-b-2 px-4 py-3 text-sm font-semibold transition-colors shrink-0 cursor-pointer ${
            activeTab === 'substitutions'
              ? 'border-tlu-navy text-tlu-navy dark:border-tlu-blue dark:text-tlu-cyan'
              : 'border-transparent text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200'
          }`}
        >
          <ArrowLeftRight className="h-4 w-4" />
          <span>{t('curriculum.tabs.substitutions')}</span>
        </button>
      </div>

      {/* Cohort and Major Scope Filter Bar */}
      {activeTab !== 'substitutions' && (
        <div className="flex flex-wrap items-center gap-3 rounded-xl border border-slate-200 bg-white p-3.5 shadow-xs dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center space-x-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.filters.cohort')}:
            </span>
            <select
              value={selectedCohort}
              onChange={(e) => {
                setSelectedCohort(e.target.value);
                setSelectedMajor('');
                setCoursePage(1);
              }}
              className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm font-medium text-slate-900 focus:border-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            >
              <option value="">{t('curriculum.filters.all_cohorts')}</option>
              {cohortsData?.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          </div>

          <div className="flex items-center space-x-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.filters.major')}:
            </span>
            <select
              value={effectiveMajor}
              onChange={(e) => {
                setSelectedMajor(e.target.value);
                setCoursePage(1);
              }}
              className="max-w-xs truncate rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm font-medium text-slate-900 focus:border-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            >
              <option value="">{t('curriculum.filters.all_majors')}</option>
              {majorsData?.map((m) => (
                <option key={m.major_name} value={m.major_name}>
                  {m.major_name}
                </option>
              ))}
            </select>
          </div>
        </div>
      )}

      {/* Tab 1: Courses Directory */}
      {activeTab === 'courses' && (
        <div className="space-y-4">
          {/* Filters and Actions */}
          <div className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-800 dark:bg-slate-900 lg:flex-row lg:items-center lg:justify-between">
            <div className="flex flex-1 flex-wrap items-center gap-3">
              <div className="relative min-w-50 flex-1">
                <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
                <input
                  type="text"
                  value={courseSearch}
                  onChange={(e) => {
                    setCourseSearch(e.target.value);
                    setCoursePage(1);
                  }}
                  placeholder={t('curriculum.filters.search_placeholder')}
                  className="w-full rounded-lg border border-slate-300 bg-white py-2 pl-9 pr-3 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
                />
              </div>

              <select
                value={blockFilter}
                onChange={(e) => {
                  setBlockFilter(e.target.value);
                  setCoursePage(1);
                }}
                className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
              >
                <option value="">{t('curriculum.filters.all_blocks')}</option>
                {availableBlocks.map((blk) => (
                  <option key={blk} value={blk}>
                    {blk}
                  </option>
                ))}
              </select>

              <select
                value={semesterFilter}
                onChange={(e) => {
                  setSemesterFilter(parseInt(e.target.value, 10) || 0);
                  setCoursePage(1);
                }}
                className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
              >
                <option value={0}>{t('curriculum.filters.all_semesters')}</option>
                {[1, 2, 3, 4, 5, 6, 7, 8].map((s) => (
                  <option key={s} value={s}>
                    {t('curriculum.filters.semester_num', { num: s })}
                  </option>
                ))}
              </select>

              <select
                value={elearningFilter}
                onChange={(e) => {
                  setElearningFilter(parseInt(e.target.value, 10));
                  setCoursePage(1);
                }}
                className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
              >
                <option value="-1">{t('curriculum.filters.all_modes')}</option>
                <option value="0">{t('curriculum.filters.mode_in_person')}</option>
                <option value="1">{t('curriculum.filters.mode_elearning')}</option>
              </select>

              {(courseSearch || blockFilter || semesterFilter > 0 || elearningFilter !== -1) && (
                <button
                  type="button"
                  onClick={() => {
                    setCourseSearch('');
                    setBlockFilter('');
                    setSemesterFilter(0);
                    setElearningFilter(-1);
                    setCoursePage(1);
                  }}
                  className="rounded-lg border border-slate-300 px-3 py-2 text-xs font-semibold text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                >
                  {t('curriculum.filters.clear')}
                </button>
              )}
            </div>

            <PermissionGate permission="curriculum.manage">
              <button
                type="button"
                onClick={() => {
                  setEditingCourse(null);
                  setIsCourseModalOpen(true);
                }}
                className="inline-flex items-center rounded-lg bg-tlu-navy hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-blue-600 px-4 py-2 text-sm font-semibold text-white shadow-md shadow-blue-900/20"
              >
                <Plus className="mr-1.5 h-4 w-4" />
                {t('curriculum.courses.add_btn')}
              </button>
            </PermissionGate>
          </div>

          {/* Table Container */}
          <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-900">
            {isCoursesLoading ? (
              <div className="flex h-64 items-center justify-center">
                <LoadingSpinner size="lg" />
              </div>
            ) : coursesData?.items.length === 0 ? (
              <div className="p-12 text-center">
                <BookOpen className="mx-auto h-12 w-12 text-slate-400 dark:text-slate-600" />
                <p className="mt-3 text-sm font-medium text-slate-600 dark:text-slate-400">
                  {t('curriculum.courses.empty')}
                </p>
              </div>
            ) : (
              <>
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="border-b border-slate-200 bg-slate-50/75 text-xs uppercase text-slate-500 dark:border-slate-800 dark:bg-slate-800/50 dark:text-slate-400">
                      <tr>
                        <th className="px-4 py-3">{t('curriculum.courses.code')}</th>
                        <th className="px-4 py-3">{t('curriculum.courses.name')}</th>
                        <th className="px-4 py-3 text-center">{t('curriculum.courses.credits')}</th>
                        <th className="px-4 py-3 text-center">{t('curriculum.courses.weight')}</th>
                        <th className="px-4 py-3 text-center">{t('curriculum.courses.hours')}</th>
                        <th className="px-4 py-3">{t('curriculum.courses.block')}</th>
                        <th className="px-4 py-3 text-center">{t('curriculum.courses.semester')}</th>
                        <th className="px-4 py-3">{t('curriculum.courses.prereqs')}</th>
                        <th className="px-4 py-3 text-right">{t('common.actions')}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                      {coursesData?.items.map((course) => (
                        <tr
                          key={course.id}
                          className="hover:bg-slate-50/50 dark:hover:bg-slate-800/50 transition-colors"
                        >
                          <td className="whitespace-nowrap px-4 py-3 font-mono font-bold text-tlu-navy dark:text-blue-400">
                            {course.course_code}
                          </td>
                          <td className="px-4 py-3 font-medium text-slate-900 dark:text-slate-100">
                            <div className="flex items-center space-x-2">
                              <span>{course.course_name}</span>
                              {course.is_elearning && (
                                <span className="inline-flex items-center rounded-full bg-blue-100 px-2 py-0.5 text-[10px] font-semibold text-tlu-navy dark:bg-blue-950/60 dark:text-blue-300">
                                  {t('curriculum.courses.badge_elearning')}
                                </span>
                              )}
                            </div>
                          </td>
                          <td className="whitespace-nowrap px-4 py-3 text-center font-semibold text-slate-700 dark:text-slate-300">
                            {course.credits}
                          </td>
                          <td className="whitespace-nowrap px-4 py-3 text-center text-slate-500 dark:text-slate-400">
                            {course.weight}
                          </td>
                          <td className="whitespace-nowrap px-4 py-3 text-center text-slate-500 dark:text-slate-400">
                            {course.hours || '-'}
                          </td>
                          <td className="px-4 py-3 text-xs text-slate-600 dark:text-slate-300">
                            <span className="rounded bg-slate-100 px-2 py-0.5 dark:bg-slate-800">
                              {course.knowledge_block}
                            </span>
                          </td>
                          <td className="whitespace-nowrap px-4 py-3 text-center font-medium text-slate-700 dark:text-slate-300">
                            {course.suggested_semester > 0 ? course.suggested_semester : '-'}
                          </td>
                          <td className="px-4 py-3 text-xs font-mono text-amber-700 dark:text-amber-400">
                            {formatPrerequisitesDisplay(course.prerequisites_text)}
                          </td>
                          <td className="whitespace-nowrap px-4 py-3 text-right">
                            <div className="flex items-center justify-end space-x-2">
                              <PermissionGate permission="curriculum.manage">
                                <button
                                  type="button"
                                  onClick={() => {
                                    setEditingCourse(course);
                                    setIsCourseModalOpen(true);
                                  }}
                                  className="rounded p-1 text-slate-400 hover:text-tlu-blue dark:hover:text-blue-400"
                                  title={t('common.edit')}
                                >
                                  <Pencil className="h-4 w-4" />
                                </button>
                                <button
                                  type="button"
                                  onClick={() => setDeletingCourse(course)}
                                  className="rounded p-1 text-slate-400 hover:text-rose-600 dark:hover:text-rose-400"
                                  title={t('common.delete')}
                                >
                                  <Trash2 className="h-4 w-4" />
                                </button>
                              </PermissionGate>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                {/* Pagination */}
                <div className="flex items-center justify-between border-t border-slate-200 px-4 py-3 dark:border-slate-800 sm:px-6">
                  <div className="text-xs text-slate-500 dark:text-slate-400">
                    {t('common.page', {
                      current: coursePage,
                      total: totalCoursePages,
                      count: coursesData?.total ?? 0,
                    })}
                  </div>

                  <div className="flex items-center space-x-2">
                    <button
                      type="button"
                      disabled={coursePage <= 1}
                      onClick={() => setCoursePage((p) => Math.max(1, p - 1))}
                      className="rounded-lg border border-slate-300 p-1.5 text-slate-600 hover:bg-slate-100 disabled:opacity-40 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      disabled={coursePage >= totalCoursePages}
                      onClick={() => setCoursePage((p) => p + 1)}
                      className="rounded-lg border border-slate-300 p-1.5 text-slate-600 hover:bg-slate-100 disabled:opacity-40 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                    >
                      <ChevronRight className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* Tab 2: Visual Roadmap and Prerequisite Network */}
      {activeTab === 'visualization' && (
        <CurriculumRoadmapView data={vizData ?? null} isLoading={isVizLoading} />
      )}

      {/* Tab 3: Substitutions */}
      {activeTab === 'substitutions' && (
        <div className="space-y-4">
          {/* Filters and Actions */}
          <div className="rounded-xl border border-slate-200 bg-white p-3.5 shadow-xs dark:border-slate-800 dark:bg-slate-900">
            <div className="flex flex-col gap-2.5 lg:flex-row lg:items-center lg:justify-between lg:flex-nowrap">
              {/* Search input */}
              <div className="relative flex-1 min-w-[180px]">
                <Search className="absolute left-3 top-3 h-4 w-4 text-slate-400 dark:text-slate-500" />
                <input
                  type="text"
                  value={subSearch}
                  onChange={(e) => {
                    setSubSearch(e.target.value);
                    setSubPage(1);
                  }}
                  placeholder={t('curriculum.filters.search_sub_placeholder')}
                  className="h-10 w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 py-2 pl-9 pr-9 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 dark:placeholder:text-slate-500 focus:border-tlu-blue focus:ring-2 focus:ring-tlu-blue/20 focus:outline-none"
                />
                {subSearch && (
                  <button
                    type="button"
                    onClick={() => {
                      setSubSearch('');
                      setSubPage(1);
                    }}
                    className="absolute right-2.5 top-2.5 rounded-md p-0.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                    title={t('common.clear')}
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>

              {/* Filters & Actions Group */}
              <div className="flex flex-wrap lg:flex-nowrap items-center gap-2 shrink-0">
                <div className="flex items-center gap-1.5 shrink-0">
                  <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400 shrink-0">
                    {t('curriculum.filters.cohort')}:
                  </span>
                  <select
                    value={subScopeCohort}
                    onChange={(e) => {
                      setSubScopeCohort(e.target.value);
                      setSubPage(1);
                    }}
                    className="h-10 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-2.5 text-sm font-medium text-slate-900 dark:text-slate-100 focus:border-tlu-blue focus:ring-2 focus:ring-tlu-blue/20 focus:outline-none"
                  >
                    <option value="ALL">{t('curriculum.substitutions.filter_all_cohorts')}</option>
                    {cohortsData?.map((c) => (
                      <option key={c} value={c}>
                        {c}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="flex items-center gap-1.5 shrink-0">
                  <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400 shrink-0">
                    {t('curriculum.filters.major')}:
                  </span>
                  <select
                    value={subScopeMajor}
                    onChange={(e) => {
                      setSubScopeMajor(e.target.value);
                      setSubPage(1);
                    }}
                    className="h-10 w-48 xl:w-56 truncate rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-2.5 text-sm font-medium text-slate-900 dark:text-slate-100 focus:border-tlu-blue focus:ring-2 focus:ring-tlu-blue/20 focus:outline-none"
                  >
                    <option value="ALL">{t('curriculum.substitutions.filter_all_majors')}</option>
                    {uniqueMajors.map((m) => (
                      <option key={m.major_code || m.major_name} value={m.major_code || m.major_name}>
                        {m.major_code ? `${m.major_code} - ` : ''}{m.major_name}
                      </option>
                    ))}
                  </select>
                </div>

                {(subSearch.trim() !== '' || subScopeCohort !== 'ALL' || subScopeMajor !== 'ALL') && (
                  <button
                    type="button"
                    onClick={() => {
                      setSubSearch('');
                      setSubScopeCohort('ALL');
                      setSubScopeMajor('ALL');
                      setSubPage(1);
                    }}
                    className="h-10 inline-flex items-center gap-1.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/80 px-2.5 text-xs font-semibold text-slate-600 hover:text-slate-900 hover:bg-slate-100 dark:text-slate-300 dark:hover:text-slate-100 dark:hover:bg-slate-800 transition-colors shrink-0 whitespace-nowrap"
                    title={t('curriculum.filters.clear')}
                  >
                    <RotateCcw className="h-3.5 w-3.5" />
                    <span className="hidden xl:inline">{t('curriculum.filters.clear')}</span>
                  </button>
                )}

                <PermissionGate permission="curriculum.manage">
                  <button
                    type="button"
                    onClick={() => {
                      setEditingSub(null);
                      setIsSubModalOpen(true);
                    }}
                    className="h-10 inline-flex items-center gap-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white px-3.5 text-sm font-semibold shadow-xs transition-colors shrink-0 whitespace-nowrap"
                  >
                    <Plus className="h-4 w-4" />
                    <span>{t('curriculum.substitutions.add_btn')}</span>
                  </button>
                </PermissionGate>
              </div>
            </div>
          </div>

          {/* Substitutions Table */}
          <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-900">
            {isSubsLoading ? (
              <div className="flex h-64 items-center justify-center">
                <LoadingSpinner size="lg" />
              </div>
            ) : subsData?.items.length === 0 ? (
              <div className="p-12 text-center">
                <ArrowLeftRight className="mx-auto h-12 w-12 text-slate-400 dark:text-slate-600" />
                <p className="mt-3 text-sm font-medium text-slate-600 dark:text-slate-400">
                  {t('curriculum.substitutions.empty')}
                </p>
              </div>
            ) : (
              <>
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="border-b border-slate-200 bg-slate-50/75 text-xs uppercase text-slate-500 dark:border-slate-800 dark:bg-slate-800/50 dark:text-slate-400">
                      <tr>
                        <th className="px-4 py-3">{t('curriculum.substitutions.original_course')}</th>
                        <th className="px-4 py-3 text-center">{t('curriculum.substitutions.direction')}</th>
                        <th className="px-4 py-3">{t('curriculum.substitutions.replacement_course')}</th>
                        <th className="px-4 py-3">{t('curriculum.substitutions.scope_cohort')}</th>
                        <th className="px-4 py-3">{t('curriculum.substitutions.scope_major')}</th>
                        <th className="px-4 py-3">{t('curriculum.substitutions.rule_type')}</th>
                        <th className="px-4 py-3">{t('curriculum.substitutions.source_reference')}</th>
                        <th className="px-4 py-3 text-right">{t('common.actions')}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                      {subsData?.items.map((sub) => (
                        <tr
                          key={sub.id}
                          className="hover:bg-slate-50/50 dark:hover:bg-slate-800/50 transition-colors"
                        >
                          <td className="px-4 py-3">
                            <span className="font-mono font-bold text-tlu-navy dark:text-blue-400">
                              {sub.original_course_code}
                            </span>
                            <span className="ml-2 font-medium text-slate-900 dark:text-slate-100">
                              {sub.original_course_name}
                            </span>
                            <span className="ml-1 text-xs text-slate-400">({sub.credits_original} TC)</span>
                          </td>
                          <td className="whitespace-nowrap px-4 py-3 text-center">
                            {sub.is_bidirectional ? (
                              <span className="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-semibold text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400">
                                ⇄
                              </span>
                            ) : (
                              <span className="inline-flex items-center rounded-full bg-blue-50 px-2 py-0.5 text-xs font-semibold text-blue-700 dark:bg-blue-950/60 dark:text-blue-400">
                                →
                              </span>
                            )}
                          </td>
                          <td className="px-4 py-3">
                            <span className="font-mono font-bold text-emerald-600 dark:text-emerald-400">
                              {sub.replacement_course_code}
                            </span>
                            <span className="ml-2 font-medium text-slate-900 dark:text-slate-100">
                              {sub.replacement_course_name}
                            </span>
                            <span className="ml-1 text-xs text-slate-400">({sub.credits_replacement} TC)</span>
                          </td>
                          <td className="whitespace-nowrap px-4 py-3">
                            {renderCohortBadge(sub)}
                          </td>
                          <td className="whitespace-nowrap px-4 py-3">
                            <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-800 dark:bg-slate-800 dark:text-slate-300">
                              {sub.scope_major}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-xs text-slate-500 dark:text-slate-400">
                            {sub.rule_type}
                          </td>
                          <td className="px-4 py-3 text-xs text-slate-500 dark:text-slate-400">
                            {sub.source_reference || '-'}
                          </td>
                          <td className="whitespace-nowrap px-4 py-3 text-right">
                            <div className="flex items-center justify-end space-x-2">
                              <PermissionGate permission="curriculum.manage">
                                <button
                                  type="button"
                                  onClick={() => {
                                    setEditingSub(sub);
                                    setIsSubModalOpen(true);
                                  }}
                                  className="rounded p-1 text-slate-400 hover:text-tlu-blue dark:hover:text-blue-400"
                                  title={t('common.edit')}
                                >
                                  <Pencil className="h-4 w-4" />
                                </button>
                                <button
                                  type="button"
                                  onClick={() => setDeletingSub(sub)}
                                  className="rounded p-1 text-slate-400 hover:text-rose-600 dark:hover:text-rose-400"
                                  title={t('common.delete')}
                                >
                                  <Trash2 className="h-4 w-4" />
                                </button>
                              </PermissionGate>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                {/* Pagination */}
                <div className="flex items-center justify-between border-t border-slate-200 px-4 py-3 dark:border-slate-800 sm:px-6">
                  <div className="text-xs text-slate-500 dark:text-slate-400">
                    {t('common.page', {
                      current: subPage,
                      total: totalSubPages,
                      count: subsData?.total ?? 0,
                    })}
                  </div>

                  <div className="flex items-center space-x-2">
                    <button
                      type="button"
                      disabled={subPage <= 1}
                      onClick={() => setSubPage((p) => Math.max(1, p - 1))}
                      className="rounded-lg border border-slate-300 p-1.5 text-slate-600 hover:bg-slate-100 disabled:opacity-40 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      disabled={subPage >= totalSubPages}
                      onClick={() => setSubPage((p) => p + 1)}
                      className="rounded-lg border border-slate-300 p-1.5 text-slate-600 hover:bg-slate-100 disabled:opacity-40 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                    >
                      <ChevronRight className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* Course Modal */}
      {isCourseModalOpen && (
        <CourseModal
          key={editingCourse?.id || 'new-course'}
          isOpen={isCourseModalOpen}
          onClose={() => {
            setIsCourseModalOpen(false);
            setEditingCourse(null);
          }}
          onSubmit={async (payload) => {
            await saveCourseMutation.mutateAsync(payload);
          }}
          initialCourse={editingCourse}
          defaultCohort={selectedCohort}
          defaultMajor={effectiveMajor}
          isSubmitting={saveCourseMutation.isPending}
        />
      )}

      {/* Substitution Modal */}
      {isSubModalOpen && (
        <SubstitutionModal
          key={editingSub?.id || 'new-sub'}
          isOpen={isSubModalOpen}
          onClose={() => {
            setIsSubModalOpen(false);
            setEditingSub(null);
          }}
          onSubmit={async (payload) => {
            await saveSubMutation.mutateAsync(payload);
          }}
          initialSubstitution={editingSub}
          defaultCohort={effectiveSubCohort}
          defaultMajor={effectiveSubMajor}
          availableMajors={uniqueMajors.map((m) => ({
            major_name: m.major_name,
            major_code: m.major_code || '',
          }))}
          isSubmitting={saveSubMutation.isPending}
        />
      )}

      {/* Course Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={Boolean(deletingCourse)}
        onClose={() => setDeletingCourse(null)}
        onConfirm={() => {
          if (deletingCourse) {
            deleteCourseMutation.mutate(deletingCourse.id);
          }
        }}
        title={t('curriculum.courses.delete_confirm_title')}
        message={
          deletingCourse
            ? t('curriculum.courses.delete_confirm_desc', {
                code: deletingCourse.course_code,
                name: deletingCourse.course_name,
              })
            : ''
        }
        variant="danger"
        confirmText={t('common.delete')}
        cancelText={t('common.cancel')}
        isLoading={deleteCourseMutation.isPending}
      />

      {/* Substitution Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={Boolean(deletingSub)}
        onClose={() => setDeletingSub(null)}
        onConfirm={() => {
          if (deletingSub) {
            deleteSubMutation.mutate(deletingSub.id);
          }
        }}
        title={t('curriculum.substitutions.delete_confirm_title')}
        message={
          deletingSub
            ? t('curriculum.substitutions.delete_confirm_desc', {
                orig: deletingSub.original_course_code,
                repl: deletingSub.replacement_course_code,
              })
            : ''
        }
        variant="danger"
        confirmText={t('common.delete')}
        cancelText={t('common.cancel')}
        isLoading={deleteSubMutation.isPending}
      />
    </div>
  );
};
