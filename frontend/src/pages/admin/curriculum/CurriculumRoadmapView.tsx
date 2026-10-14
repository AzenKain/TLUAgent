import React, { useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Layers,
  ArrowRight,
  GitBranch,
  Search,
  BookOpen,
  CheckCircle2,
  Info,
  X,
  LayoutGrid,
  Network,
  ArrowLeftRight,
  Sparkles,
  ChevronRight,
  GraduationCap,
  Award,
} from 'lucide-react';
import type { CurriculumVisualizationData, CurriculumCourse, PrerequisiteEdge } from '@/types/curriculum';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';

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

const extractCreditBadge = (text: string): string | null => {
  const credits = extractCreditRequirementDetail(text);
  if (!credits) return null;
  return `≥ ${credits} TC`;
};


interface CurriculumRoadmapViewProps {
  data: CurriculumVisualizationData | null;
  isLoading: boolean;
}

interface RoadmapColumn {
  id: string;
  title: string;
  subtitle: string;
  icon: React.ReactNode;
  badgeStyle: string;
  headerBorder: string;
  courses: CurriculumCourse[];
  totalCredits: number;
}

const BLOCK_COLORS = [
  'bg-tlu-blue',
  'bg-tlu-navy',
  'bg-tlu-red',
  'bg-emerald-600',
  'bg-amber-500',
  'bg-tlu-cyan',
  'bg-slate-700',
  'bg-sky-600',
];

const isElectiveCourse = (course: CurriculumCourse): boolean => {
  const code = course.course_code.toUpperCase();
  const electivePrefixes = ['GF', 'GJ', 'GK', 'GI', 'GZ', 'VL101', 'CS102', 'NA151', 'EC102', 'SH131'];
  if (electivePrefixes.some((p) => code.startsWith(p))) return true;
  if (code.startsWith('GE') && !code.startsWith('GE111')) return true;
  return false;
};

const normalizeAscii = (str: string): string => {
  return str
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/\u0111/g, 'd')
    .replace(/\u0110/g, 'D')
    .toLowerCase()
    .trim();
};

const matchBlock = (
  block: string,
  type: 'general' | 'core' | 'major' | 'specialized' | 'graduation'
): boolean => {
  const norm = normalizeAscii(block);
  switch (type) {
    case 'general':
      return norm.includes('dai cuong');
    case 'core':
      return norm.includes('co so');
    case 'major':
      return norm.includes('nganh') && !norm.includes('chuyen nganh') && !norm.includes('co so');
    case 'specialized':
      return norm.includes('chuyen nganh');
    case 'graduation':
      return norm.includes('tot nghiep') || norm.includes('thuc tap') || norm.includes('khoa luan');
    default:
      return false;
  }
};

// CurriculumRoadmapView renders responsive visual credit distributions, customizable kanban stages, and prerequisite network flows.
export const CurriculumRoadmapView: React.FC<CurriculumRoadmapViewProps> = ({
  data,
  isLoading,
}) => {
  const { t } = useTranslation();
  const [selectedCourseCode, setSelectedCourseCode] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [selectedBlock, setSelectedBlock] = useState<string>('');
  const [selectedColumnFilter, setSelectedColumnFilter] = useState<string>('ALL');
  const [viewMode, setViewMode] = useState<'kanban' | 'dag' | 'table'>('kanban');

  const allCourses = useMemo(() => {
    if (data?.courses && data.courses.length > 0) return data.courses;
    if (!data?.semesters) return [];
    return data.semesters.flatMap((s) => s.courses);
  }, [data]);

  const courseMap = useMemo(() => {
    const map = new Map<string, CurriculumCourse>();
    allCourses.forEach((c) => {
      map.set(c.course_code.toUpperCase(), c);
    });
    return map;
  }, [allCourses]);

  const edges = useMemo(() => data?.prerequisite_edges ?? [], [data]);

  const incomingEdgesMap = useMemo(() => {
    const map = new Map<string, PrerequisiteEdge[]>();
    edges.forEach((edge) => {
      const toCode = (edge.to_course_code || edge.to || '').toUpperCase();
      if (!toCode) return;
      if (!map.has(toCode)) map.set(toCode, []);
      map.get(toCode)!.push(edge);
    });
    return map;
  }, [edges]);

  const outgoingEdgesMap = useMemo(() => {
    const map = new Map<string, PrerequisiteEdge[]>();
    edges.forEach((edge) => {
      const fromCode = (edge.from_course_code || edge.from || '').toUpperCase();
      if (!fromCode) return;
      if (!map.has(fromCode)) map.set(fromCode, []);
      map.get(fromCode)!.push(edge);
    });
    return map;
  }, [edges]);

  const activeDependencySets = useMemo(() => {
    if (!selectedCourseCode) {
      return {
        ancestors: new Set<string>(),
        descendants: new Set<string>(),
        chain: [] as string[],
      };
    }

    const sel = selectedCourseCode.toUpperCase();
    const ancestors = new Set<string>();
    const queueUp = [sel];
    while (queueUp.length > 0) {
      const curr = queueUp.shift()!;
      const parents = incomingEdgesMap.get(curr) || [];
      parents.forEach((edge) => {
        const pCode = (edge.from_course_code || edge.from || '').toUpperCase();
        if (pCode && !ancestors.has(pCode) && pCode !== sel) {
          ancestors.add(pCode);
          queueUp.push(pCode);
        }
      });
    }

    const descendants = new Set<string>();
    const queueDown = [sel];
    while (queueDown.length > 0) {
      const curr = queueDown.shift()!;
      const children = outgoingEdgesMap.get(curr) || [];
      children.forEach((edge) => {
        const cCode = (edge.to_course_code || edge.to || '').toUpperCase();
        if (cCode && !descendants.has(cCode) && cCode !== sel) {
          descendants.add(cCode);
          queueDown.push(cCode);
        }
      });
    }

    const chain: string[] = [...Array.from(ancestors), sel, ...Array.from(descendants)];
    return { ancestors, descendants, chain };
  }, [selectedCourseCode, incomingEdgesMap, outgoingEdgesMap]);

  const selectedCourse = useMemo(() => {
    if (!selectedCourseCode) return null;
    return courseMap.get(selectedCourseCode.toUpperCase()) || null;
  }, [selectedCourseCode, courseMap]);

  const filteredEdges = useMemo(() => {
    return edges.filter((edge) => {
      const fromCode = (edge.from_course_code || edge.from || '').toUpperCase();
      const toCode = (edge.to_course_code || edge.to || '').toUpperCase();
      const fromCourse = courseMap.get(fromCode);
      const toCourse = courseMap.get(toCode);
      const fromName = fromCourse?.course_name || edge.from_course_name || '';
      const toName = toCourse?.course_name || edge.to_course_name || '';

      if (selectedCourseCode) {
        const sel = selectedCourseCode.toUpperCase();
        if (fromCode !== sel && toCode !== sel) return false;
      }

      if (selectedBlock) {
        const matchFrom = fromCourse?.knowledge_block === selectedBlock;
        const matchTo = toCourse?.knowledge_block === selectedBlock;
        if (!matchFrom && !matchTo) return false;
      }

      if (searchQuery.trim()) {
        const q = searchQuery.trim().toUpperCase();
        const matchesCode = fromCode.includes(q) || toCode.includes(q);
        const matchesName =
          fromName.toUpperCase().includes(q) || toName.toUpperCase().includes(q);
        if (!matchesCode && !matchesName) return false;
      }

      return true;
    });
  }, [edges, selectedCourseCode, selectedBlock, searchQuery, courseMap]);

  const roadmapColumns = useMemo<RoadmapColumn[]>(() => {
    if (!allCourses.length) return [];

    const configs = [
      {
        type: 'general' as const,
        title: t('curriculum.viz.block_general_title'),
        subtitle: t('curriculum.viz.block_general_sub'),
        icon: <BookOpen className="h-4 w-4 text-tlu-blue dark:text-tlu-cyan" />,
        badgeStyle: 'bg-blue-50 text-tlu-blue dark:bg-blue-950 dark:text-tlu-cyan',
        headerBorder: 'border-t-4 border-t-tlu-blue',
      },
      {
        type: 'core' as const,
        title: t('curriculum.viz.block_core_title'),
        subtitle: t('curriculum.viz.block_core_sub'),
        icon: <Layers className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />,
        badgeStyle: 'bg-indigo-50 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300',
        headerBorder: 'border-t-4 border-t-indigo-600',
      },
      {
        type: 'major' as const,
        title: t('curriculum.viz.block_major_title'),
        subtitle: t('curriculum.viz.block_major_sub'),
        icon: <GitBranch className="h-4 w-4 text-amber-600 dark:text-amber-400" />,
        badgeStyle: 'bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300',
        headerBorder: 'border-t-4 border-t-amber-500',
      },
      {
        type: 'specialized' as const,
        title: t('curriculum.viz.block_specialized_title'),
        subtitle: t('curriculum.viz.block_specialized_sub'),
        icon: <Sparkles className="h-4 w-4 text-purple-600 dark:text-purple-400" />,
        badgeStyle: 'bg-purple-50 text-purple-700 dark:bg-purple-950 dark:text-purple-300',
        headerBorder: 'border-t-4 border-t-purple-600',
      },
      {
        type: 'graduation' as const,
        title: t('curriculum.viz.block_graduation_title'),
        subtitle: t('curriculum.viz.block_graduation_sub'),
        icon: <GraduationCap className="h-4 w-4 text-emerald-600 dark:text-emerald-400" />,
        badgeStyle: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300',
        headerBorder: 'border-t-4 border-t-emerald-600',
      },
    ];

    const columns: RoadmapColumn[] = [];
    const handledCourseIds = new Set<string>();

    configs.forEach((cfg, idx) => {
      const matches = allCourses.filter((c) => matchBlock(c.knowledge_block, cfg.type));
      if (matches.length > 0) {
        matches.forEach((c) => handledCourseIds.add(c.id));
        const credits = matches.reduce((sum, c) => sum + c.credits, 0);
        columns.push({
          id: `col-block-${idx}`,
          title: cfg.title,
          subtitle: cfg.subtitle,
          icon: cfg.icon,
          badgeStyle: cfg.badgeStyle,
          headerBorder: cfg.headerBorder,
          courses: matches,
          totalCredits: credits,
        });
      }
    });

    const unassigned = allCourses.filter((c) => !handledCourseIds.has(c.id));
    if (unassigned.length > 0) {
      const remainingBlocks = Array.from(new Set(unassigned.map((c) => c.knowledge_block)));
      remainingBlocks.forEach((blk, bIdx) => {
        const matches = unassigned.filter((c) => c.knowledge_block === blk);
        const credits = matches.reduce((sum, c) => sum + c.credits, 0);
        columns.push({
          id: `col-block-extra-${bIdx}`,
          title: blk || t('curriculum.filters.all_blocks'),
          subtitle: t('curriculum.viz.stage_major'),
          icon: <Layers className="h-4 w-4 text-slate-600" />,
          badgeStyle: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300',
          headerBorder: 'border-t-4 border-t-slate-500',
          courses: matches,
          totalCredits: credits,
        });
      });
    }

    return columns;
  }, [allCourses, t]);

  const dagStageGroups = useMemo(() => {
    const stages = [
      {
        id: 'dag-block-1',
        name: t('curriculum.viz.block_general_title'),
        desc: t('curriculum.viz.block_general_sub'),
        type: 'general' as const,
        courses: [] as CurriculumCourse[],
      },
      {
        id: 'dag-block-2',
        name: t('curriculum.viz.block_core_title'),
        desc: t('curriculum.viz.block_core_sub'),
        type: 'core' as const,
        courses: [] as CurriculumCourse[],
      },
      {
        id: 'dag-block-3',
        name: t('curriculum.viz.block_major_title'),
        desc: t('curriculum.viz.block_major_sub'),
        type: 'major' as const,
        courses: [] as CurriculumCourse[],
      },
      {
        id: 'dag-block-4',
        name: t('curriculum.viz.block_specialized_title'),
        desc: t('curriculum.viz.block_specialized_sub'),
        type: 'specialized' as const,
        courses: [] as CurriculumCourse[],
      },
      {
        id: 'dag-block-5',
        name: t('curriculum.viz.block_graduation_title'),
        desc: t('curriculum.viz.block_graduation_sub'),
        type: 'graduation' as const,
        courses: [] as CurriculumCourse[],
      },
    ];

    allCourses.forEach((c) => {
      const target = stages.find((s) => matchBlock(c.knowledge_block, s.type)) || stages[0];
      target.courses.push(c);
    });

    return stages.filter((s) => s.courses.length > 0);
  }, [allCourses, t]);

  const handleSelectCourse = (code: string) => {
    setSelectedCourseCode((prev) => (prev.toUpperCase() === code.toUpperCase() ? '' : code));
  };

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  if (!data) {
    return (
      <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-slate-300 p-12 text-center dark:border-slate-800">
        <BookOpen className="h-12 w-12 text-slate-400 dark:text-slate-600" />
        <p className="mt-3 text-sm font-medium text-slate-600 dark:text-slate-400">
          {t('curriculum.viz.select_hint')}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4 sm:space-y-6 w-full min-w-0 max-w-full">
      {/* Overview Program Summary Bar */}
      <div className="grid grid-cols-2 gap-2.5 sm:gap-4 lg:grid-cols-4">
        <div className="rounded-xl sm:rounded-2xl border border-slate-200 bg-white p-3.5 sm:p-4 shadow-xs dark:border-blue-900/30 dark:bg-[#111A35]">
          <div className="flex items-center justify-between">
            <span className="text-[10px] sm:text-xs font-semibold uppercase tracking-wider text-slate-400">
              {t('curriculum.viz.summary_courses')}
            </span>
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-50 text-tlu-blue dark:bg-blue-950/60 dark:text-tlu-cyan">
              <BookOpen className="h-3.5 w-3.5" />
            </div>
          </div>
          <div className="mt-1.5 text-lg sm:text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            {data.total_courses}{' '}
            <span className="text-xs font-normal text-slate-400">{t('curriculum.viz.course_count_unit')}</span>
          </div>
        </div>

        <div className="rounded-xl sm:rounded-2xl border border-slate-200 bg-white p-3.5 sm:p-4 shadow-xs dark:border-blue-900/30 dark:bg-[#111A35]">
          <div className="flex items-center justify-between">
            <span className="text-[10px] sm:text-xs font-semibold uppercase tracking-wider text-slate-400">
              {t('curriculum.viz.summary_credits')}
            </span>
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-50 text-emerald-600 dark:bg-emerald-950/60 dark:text-emerald-400">
              <GraduationCap className="h-3.5 w-3.5" />
            </div>
          </div>
          <div className="mt-1.5 text-lg sm:text-2xl font-bold tracking-tight text-emerald-600 dark:text-emerald-400">
            {data.total_credits}{' '}
            <span className="text-xs font-normal text-slate-400">{t('curriculum.viz.credits_unit')}</span>
          </div>
        </div>

        <div className="rounded-xl sm:rounded-2xl border border-slate-200 bg-white p-3.5 sm:p-4 shadow-xs dark:border-blue-900/30 dark:bg-[#111A35]">
          <div className="flex items-center justify-between">
            <span className="text-[10px] sm:text-xs font-semibold uppercase tracking-wider text-slate-400">
              {t('curriculum.viz.summary_prerequisites')}
            </span>
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-50 text-amber-600 dark:bg-amber-950/60 dark:text-amber-400">
              <GitBranch className="h-3.5 w-3.5" />
            </div>
          </div>
          <div className="mt-1.5 text-lg sm:text-2xl font-bold tracking-tight text-amber-600 dark:text-amber-400">
            {data.prerequisite_edges?.length ?? 0}
          </div>
        </div>

        <div className="rounded-xl sm:rounded-2xl border border-slate-200 bg-white p-3.5 sm:p-4 shadow-xs dark:border-blue-900/30 dark:bg-[#111A35]">
          <div className="flex items-center justify-between">
            <span className="text-[10px] sm:text-xs font-semibold uppercase tracking-wider text-slate-400">
              {t('curriculum.viz.summary_blocks')}
            </span>
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-purple-50 text-purple-600 dark:bg-purple-950/60 dark:text-purple-400">
              <Layers className="h-3.5 w-3.5" />
            </div>
          </div>
          <div className="mt-1.5 text-lg sm:text-2xl font-bold tracking-tight text-purple-600 dark:text-purple-400">
            {data.knowledge_blocks?.length ?? 0}
          </div>
        </div>
      </div>

      {/* Section A: Knowledge Block Distribution */}
      <div className="rounded-xl sm:rounded-2xl border border-slate-200 bg-white p-3.5 sm:p-5 shadow-xs dark:border-blue-900/30 dark:bg-[#111A35]">
        <div className="flex items-center space-x-2">
          <Layers className="h-4 w-4 sm:h-5 sm:w-5 text-tlu-blue dark:text-tlu-cyan" />
          <h3 className="text-sm sm:text-base font-bold text-slate-900 dark:text-slate-100">
            {t('curriculum.viz.block_distribution')}
          </h3>
        </div>

        {/* Multi-segment progress bar */}
        <div className="mt-3 sm:mt-4 flex h-3 sm:h-3.5 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800">
          {data.knowledge_blocks.map((block, idx) => {
            const bName = block.knowledge_block || (block as unknown as { name?: string }).name || '';
            const pct = typeof block.percentage === 'number' ? Number(block.percentage.toFixed(1)) : 0;
            return (
              <div
                key={bName || idx}
                style={{ width: `${pct}%` }}
                className={`${BLOCK_COLORS[idx % BLOCK_COLORS.length]} transition-all duration-300`}
                title={`${bName}: ${block.total_credits} ${t('curriculum.viz.credits_unit')} (${pct}%)`}
              />
            );
          })}
        </div>

        {/* Cards for each block */}
        <div className="mt-3 sm:mt-4 grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-5 gap-2 sm:gap-3">
          {data.knowledge_blocks.map((block, idx) => {
            const bName = block.knowledge_block || (block as unknown as { name?: string }).name || '';
            const pct = typeof block.percentage === 'number' ? Number(block.percentage.toFixed(1)) : 0;
            return (
              <div
                key={bName || idx}
                onClick={() =>
                  setSelectedBlock((prev) => (prev === bName ? '' : bName))
                }
                className={`flex items-center space-x-2.5 rounded-xl border p-2.5 sm:p-3 cursor-pointer transition-all ${
                  selectedBlock === bName
                    ? 'border-tlu-blue bg-blue-50/80 dark:bg-blue-950/50 ring-2 ring-blue-500/20'
                    : 'border-slate-100 bg-slate-50/60 dark:border-slate-800/60 dark:bg-slate-800/30 hover:border-blue-200 dark:hover:border-slate-700'
                }`}
              >
                <div
                  className={`h-3 w-3 shrink-0 rounded-full ${BLOCK_COLORS[idx % BLOCK_COLORS.length]}`}
                />
                <div className="min-w-0 flex-1">
                  <p
                    className="truncate text-[11px] sm:text-xs font-semibold text-slate-800 dark:text-slate-200"
                    title={bName}
                  >
                    {bName || t('curriculum.filters.all_blocks')}
                  </p>
                  <div className="mt-0.5 flex items-center justify-between text-[10px] sm:text-xs text-slate-500 dark:text-slate-400">
                    <span>
                      {block.total_credits} {t('curriculum.viz.credits_unit')}
                    </span>
                    <span className="font-bold text-slate-700 dark:text-slate-300">
                      {pct}%
                    </span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Main Interactive Workspace Header & Controls - Divided into 2 clean, spacious rows */}
      <div className="rounded-xl sm:rounded-2xl border border-slate-200 bg-white p-4 shadow-xs dark:border-blue-900/30 dark:bg-[#111A35] space-y-3.5">
        {/* Row 1: Dedicated Title, Icon & Description */}
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 sm:h-11 sm:w-11 items-center justify-center rounded-xl bg-blue-50 text-tlu-blue dark:bg-blue-950/60 dark:text-tlu-cyan shrink-0">
            <GitBranch className="h-5 w-5 sm:h-6 sm:w-6" />
          </div>
          <div>
            <h2 className="text-base sm:text-lg font-bold text-slate-900 dark:text-slate-100">
              {t('curriculum.viz.prerequisite_flow')}
            </h2>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
              {t('curriculum.viz.click_hint')}
            </p>
          </div>
        </div>

        {/* Row 2: Controls Toolbar (Flex-wrap to never squeeze inputs or truncate buttons) */}
        <div className="flex flex-wrap items-center justify-between gap-3 pt-3.5 border-t border-slate-100 dark:border-slate-800/80">
          {/* Left Controls: Search Input */}
          <div className="flex items-center gap-2.5 flex-1 min-w-[240px] max-w-md">
            {/* Search Input - Generous width with visible placeholder */}
            <div className="relative w-full">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-slate-400 pointer-events-none" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder={t('curriculum.viz.filter_search_placeholder')}
                className="w-full h-9 rounded-xl border border-slate-200 bg-slate-50/70 pl-8.5 pr-8 text-xs text-slate-900 placeholder:text-slate-400 focus:border-tlu-blue focus:bg-white focus:outline-none focus:ring-2 focus:ring-tlu-blue/20 dark:border-slate-700 dark:bg-slate-800/60 dark:text-slate-100"
              />
              {searchQuery && (
                <button
                  type="button"
                  onClick={() => setSearchQuery('')}
                  aria-label={t('common.clear')}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 cursor-pointer"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
          </div>

          {/* Right Controls: View Mode Switcher */}
          <div className="flex items-center rounded-xl border border-slate-200 bg-slate-100/70 p-0.5 dark:border-slate-800 dark:bg-slate-800/80 shrink-0 overflow-x-auto max-w-full scrollbar-none">
            <button
              type="button"
              onClick={() => setViewMode('kanban')}
              className={`flex items-center gap-1.5 rounded-lg px-2.5 sm:px-3 py-1.5 text-xs font-semibold transition cursor-pointer whitespace-nowrap ${
                viewMode === 'kanban'
                  ? 'bg-white text-tlu-blue shadow-xs dark:bg-slate-900 dark:text-tlu-cyan'
                  : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'
              }`}
            >
              <LayoutGrid className="h-3.5 w-3.5 shrink-0" />
              <span>{t('curriculum.viz.view_mode_kanban')}</span>
            </button>
            <button
              type="button"
              onClick={() => setViewMode('dag')}
              className={`flex items-center gap-1.5 rounded-lg px-2.5 sm:px-3 py-1.5 text-xs font-semibold transition cursor-pointer whitespace-nowrap ${
                viewMode === 'dag'
                  ? 'bg-white text-tlu-blue shadow-xs dark:bg-slate-900 dark:text-tlu-cyan'
                  : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'
              }`}
            >
              <Network className="h-3.5 w-3.5 shrink-0" />
              <span>{t('curriculum.viz.view_mode_tree')}</span>
            </button>
            <button
              type="button"
              onClick={() => setViewMode('table')}
              className={`flex items-center gap-1.5 rounded-lg px-2.5 sm:px-3 py-1.5 text-xs font-semibold transition cursor-pointer whitespace-nowrap ${
                viewMode === 'table'
                  ? 'bg-white text-tlu-blue shadow-xs dark:bg-slate-900 dark:text-tlu-cyan'
                  : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'
              }`}
            >
              <ArrowLeftRight className="h-3.5 w-3.5 shrink-0" />
              <span>{t('curriculum.viz.view_mode_table')}</span>
            </button>
          </div>
        </div>
      </div>

      {/* Selected Course Dependency & Unlock Inspector */}
      {selectedCourse && (
        <div className="rounded-xl sm:rounded-2xl border-2 border-tlu-blue/40 bg-gradient-to-br from-blue-50/70 via-white to-sky-50/40 p-4 sm:p-5 shadow-md dark:border-blue-700/50 dark:from-[#111A35] dark:via-[#0c152e] dark:to-[#080e22] animate-in fade-in slide-in-from-top-3 duration-200">
          <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-3 pb-3 sm:pb-4 border-b border-slate-200 dark:border-blue-900/40">
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-1.5 sm:gap-2">
                <span className="rounded-lg sm:rounded-xl bg-tlu-navy px-2.5 py-1 font-mono text-xs font-bold text-white dark:bg-tlu-blue">
                  {selectedCourse.course_code}
                </span>
                <span className="rounded-md sm:rounded-lg bg-blue-100/80 px-2 py-0.5 text-xs font-semibold text-tlu-navy dark:bg-blue-950 dark:text-blue-300">
                  {selectedCourse.credits} {t('curriculum.viz.credits_unit')}
                </span>
                <span className="rounded-md sm:rounded-lg bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-300">
                  {selectedCourse.knowledge_block}
                </span>
                {isElectiveCourse(selectedCourse) ? (
                  <span className="rounded-md bg-purple-100 px-2 py-0.5 text-xs font-semibold text-purple-800 dark:bg-purple-950/70 dark:text-purple-300">
                    {t('curriculum.viz.badge_elective')}
                  </span>
                ) : (
                  <span className="rounded-md bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300">
                    {t('curriculum.viz.badge_compulsory')}
                  </span>
                )}
              </div>
              <h3 className="mt-2 text-base sm:text-lg md:text-xl font-extrabold tracking-tight text-slate-900 dark:text-slate-100">
                {selectedCourse.course_name}
              </h3>
            </div>

            <button
              type="button"
              onClick={() => setSelectedCourseCode('')}
              className="self-start inline-flex items-center gap-1 rounded-xl bg-slate-100 px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700 transition cursor-pointer"
            >
              <X className="h-3.5 w-3.5" />
              <span>{t('curriculum.viz.inspector_clear')}</span>
            </button>
          </div>

          {/* Dual Panels: Prerequisites vs Unlocks */}
          <div className="mt-4 grid grid-cols-1 gap-3 sm:gap-4 md:grid-cols-2">
            {/* Panel 1: Prerequisites Needed */}
            <div className="rounded-xl border border-amber-200/80 bg-amber-50/40 p-3 sm:p-4 dark:border-amber-900/40 dark:bg-amber-950/20">
              <div className="flex items-center gap-2 mb-2.5">
                <div className="flex h-6 w-6 sm:h-7 sm:w-7 items-center justify-center rounded-lg bg-amber-500/20 text-amber-700 dark:text-amber-400">
                  <GitBranch className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
                </div>
                <h4 className="text-xs font-bold uppercase tracking-wider text-amber-800 dark:text-amber-300">
                  {t('curriculum.viz.inspector_prereqs')}
                </h4>
              </div>

              {(() => {
                const parents = incomingEdgesMap.get(selectedCourse.course_code.toUpperCase()) || [];
                const creditReq = extractCreditRequirementDetail(selectedCourse.prerequisites_text);

                interface UnpackedPrereq {
                  code: string;
                  name: string;
                  credits: number;
                  minGrade: number;
                  isOr: boolean;
                  isSelectable: boolean;
                }

                const unpackedList: UnpackedPrereq[] = [];
                parents.forEach((edge) => {
                  const rawCode = (edge.from_course_code || edge.from || '').toUpperCase().trim();
                  const sanitized = rawCode
                    .replace(/[\s,;]+(?:HO\u1EB6C|OR)[\s,;]*$/i, '')
                    .replace(/^[\s,;]+(?:HO\u1EB6C|OR)[\s,;]+/i, '')
                    .trim();

                  if (sanitized.includes('HO\u1EB6C') || sanitized.includes(' OR ')) {
                    const subCodes = sanitized.split(/\s+(?:HO\u1EB6C|OR)\s+/i).map((s) => s.trim()).filter(Boolean);

                    subCodes.forEach((sc) => {
                      const scCourse = courseMap.get(sc);
                      unpackedList.push({
                        code: sc,
                        name: scCourse?.course_name || edge.from_course_name || sc,
                        credits: scCourse?.credits || edge.from_credits || 0,
                        minGrade: edge.min_grade || 0,
                        isOr: true,
                        isSelectable: courseMap.has(sc),
                      });
                    });
                  } else {
                    const pCourse = courseMap.get(sanitized);
                    unpackedList.push({
                      code: sanitized,
                      name: pCourse?.course_name || edge.from_course_name || sanitized,
                      credits: pCourse?.credits || edge.from_credits || 0,
                      minGrade: edge.min_grade || 0,
                      isOr: edge.condition_type === 'OR',
                      isSelectable: courseMap.has(sanitized),
                    });
                  }
                });

                if (unpackedList.length === 0 && !creditReq) {
                  return (
                    <div className="flex items-center gap-2 rounded-lg bg-emerald-500/10 p-2.5 text-xs font-medium text-emerald-700 dark:text-emerald-300 border border-emerald-500/20">
                      <CheckCircle2 className="h-4 w-4 shrink-0" />
                      <span>{t('curriculum.viz.inspector_no_prereqs')}</span>
                    </div>
                  );
                }

                return (
                  <div className="space-y-2">
                    {creditReq && (
                      <div className="flex items-center gap-2 rounded-lg border border-amber-300/80 bg-amber-100/70 p-2.5 text-xs font-semibold text-amber-900 shadow-2xs dark:border-amber-700/60 dark:bg-amber-950/40 dark:text-amber-200">
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

                    {unpackedList.length === 0 && creditReq && (
                      <div className="flex items-center gap-2 rounded-lg bg-amber-500/10 p-2.5 text-xs font-medium text-amber-800 dark:text-amber-300 border border-amber-500/20">
                        <Info className="h-4 w-4 shrink-0 text-amber-600 dark:text-amber-400" />
                        <span>{t('curriculum.viz.only_credit_prereq_hint', { count: creditReq })}</span>
                      </div>
                    )}

                    {unpackedList.map((item, idx) => (
                      <React.Fragment key={`p-${item.code}-${idx}`}>
                        {idx > 0 && item.isOr && (
                          <div className="flex items-center gap-2 py-0.5 my-1">
                            <div className="h-px flex-1 bg-amber-300/60 dark:bg-amber-800/50" />
                            <span className="rounded-full border border-amber-300 bg-amber-100/90 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-amber-800 shadow-2xs dark:border-amber-700 dark:bg-amber-900/60 dark:text-amber-300">
                              {t('curriculum.viz.or_condition')}
                            </span>
                            <div className="h-px flex-1 bg-amber-300/60 dark:bg-amber-800/50" />
                          </div>
                        )}
                        <div
                          onClick={() => {
                            if (item.isSelectable) {
                              handleSelectCourse(item.code);
                            }
                          }}
                          className={`flex items-center justify-between rounded-lg border border-amber-200/90 bg-white p-2.5 text-xs shadow-2xs transition dark:border-amber-900/50 dark:bg-slate-900 ${
                            item.isSelectable
                              ? 'cursor-pointer hover:border-amber-400 hover:shadow-xs'
                              : 'cursor-default opacity-95'
                          }`}
                        >
                          <div className="min-w-0 flex-1 pr-2">
                            <div className="flex items-center gap-1.5">
                              <span className="font-mono font-bold text-tlu-navy dark:text-tlu-cyan">
                                {item.code}
                              </span>
                              <span className="text-slate-400">•</span>
                              <span className="font-semibold text-slate-800 dark:text-slate-200 truncate">
                                {item.name}
                              </span>
                            </div>
                            <div className="mt-1 flex items-center gap-2 text-[11px] text-slate-500 dark:text-slate-400">
                              <span>
                                {item.credits}{' '}
                                {t('curriculum.viz.credits_unit')}
                              </span>
                              {item.minGrade > 0 && (
                                <span className="rounded bg-amber-100 px-1 py-0.2 text-[10px] font-semibold text-amber-800 dark:bg-amber-950 dark:text-amber-300">
                                  {t('curriculum.viz.min_grade_label', { grade: item.minGrade })}
                                </span>
                              )}
                              {item.isOr && (
                                <span className="rounded bg-amber-100/80 px-1.5 py-0.2 text-[10px] font-medium text-amber-800 dark:bg-amber-950/80 dark:text-amber-300">
                                  {t('curriculum.viz.alternative_badge')}
                                </span>
                              )}
                            </div>
                          </div>
                          {item.isSelectable && <ChevronRight className="h-4 w-4 text-slate-400 shrink-0" />}
                        </div>
                      </React.Fragment>
                    ))}
                  </div>
                );
              })()}
            </div>

            {/* Panel 2: Unlocked Courses Next */}
            <div className="rounded-xl border border-emerald-200/80 bg-emerald-50/40 p-3 sm:p-4 dark:border-emerald-900/40 dark:bg-emerald-950/20">
              <div className="flex items-center gap-2 mb-2.5">
                <div className="flex h-6 w-6 sm:h-7 sm:w-7 items-center justify-center rounded-lg bg-emerald-500/20 text-emerald-700 dark:text-emerald-400">
                  <Sparkles className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
                </div>
                <h4 className="text-xs font-bold uppercase tracking-wider text-emerald-800 dark:text-emerald-300">
                  {t('curriculum.viz.inspector_unlocks')}
                </h4>
              </div>

              {(() => {
                const children = outgoingEdgesMap.get(selectedCourse.course_code.toUpperCase()) || [];
                if (children.length === 0) {
                  return (
                    <div className="flex items-center gap-2 rounded-lg bg-slate-500/10 p-2.5 text-xs font-medium text-slate-600 dark:text-slate-400 border border-slate-500/20">
                      <Info className="h-4 w-4 shrink-0" />
                      <span>{t('curriculum.viz.inspector_no_unlocks')}</span>
                    </div>
                  );
                }
                return (
                  <div className="space-y-2">
                    {children.map((edge, idx) => {
                      const cCode = (edge.to_course_code || edge.to || '').toUpperCase();
                      const cCourse = courseMap.get(cCode);
                      const isSelectable = courseMap.has(cCode);
                      return (
                        <div
                          key={`c-${cCode}-${idx}`}
                          onClick={() => {
                            if (isSelectable) {
                              handleSelectCourse(cCode);
                            }
                          }}
                          className={`flex items-center justify-between rounded-lg border border-emerald-200/90 bg-white p-2.5 text-xs shadow-2xs transition dark:border-emerald-900/50 dark:bg-slate-900 ${
                            isSelectable
                              ? 'cursor-pointer hover:border-emerald-400 hover:shadow-xs'
                              : 'cursor-default opacity-95'
                          }`}
                        >
                          <div className="min-w-0 flex-1 pr-2">
                            <div className="flex items-center gap-1.5">
                              <span className="font-mono font-bold text-emerald-700 dark:text-emerald-400">
                                {cCode}
                              </span>
                              <span className="text-slate-400">•</span>
                              <span className="font-semibold text-slate-800 dark:text-slate-200 truncate">
                                {cCourse?.course_name || edge.to_course_name || cCode}
                              </span>
                            </div>
                            <div className="mt-1 flex items-center gap-2 text-[11px] text-slate-500 dark:text-slate-400">
                              <span>
                                {cCourse?.credits || edge.to_credits || 0}{' '}
                                {t('curriculum.viz.credits_unit')}
                              </span>
                              {cCourse?.knowledge_block && (
                                <>
                                  <span className="text-slate-400">•</span>
                                  <span>{cCourse.knowledge_block}</span>
                                </>
                              )}
                            </div>
                          </div>
                          {isSelectable && <ChevronRight className="h-4 w-4 text-slate-400 shrink-0" />}
                        </div>
                      );
                    })}
                  </div>
                );
              })()}
            </div>
          </div>


          {/* Bottom Dependency Chain Breadcrumb */}
          {activeDependencySets.chain.length > 1 && (
            <div className="mt-3.5 pt-3 border-t border-slate-200/80 dark:border-blue-900/30">
              <span className="text-[11px] font-bold uppercase tracking-wider text-slate-400">
                {t('curriculum.viz.inspector_chain')}:
              </span>
              <div className="mt-2 flex items-center gap-1.5 overflow-x-auto pb-1 scrollbar-none">
                {activeDependencySets.chain.map((code, idx) => {
                  const cObj = courseMap.get(code);
                  const isCurrent = code === selectedCourse.course_code.toUpperCase();
                  const isAncestor = activeDependencySets.ancestors.has(code);
                  return (
                    <React.Fragment key={`ch-${code}-${idx}`}>
                      {idx > 0 && <ArrowRight className="h-3.5 w-3.5 text-slate-400 shrink-0" />}
                      <button
                        type="button"
                        onClick={() => handleSelectCourse(code)}
                        className={`inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-xs font-semibold shrink-0 transition cursor-pointer ${
                          isCurrent
                            ? 'bg-tlu-navy text-white shadow-xs dark:bg-tlu-blue'
                            : isAncestor
                              ? 'bg-amber-100 text-amber-800 hover:bg-amber-200 dark:bg-amber-950 dark:text-amber-300'
                              : 'bg-emerald-100 text-emerald-800 hover:bg-emerald-200 dark:bg-emerald-950 dark:text-emerald-300'
                        }`}
                      >
                        <span className="font-mono">{code}</span>
                        <span className="truncate max-w-36 font-normal">
                          {cObj?.course_name || code}
                        </span>
                      </button>
                    </React.Fragment>
                  );
                })}
              </div>
            </div>
          )}
        </div>
      )}

      {/* VIEW 1: Responsive Grid Kanban (Auto-wrapping down, NO horizontal scrolling) */}
      {viewMode === 'kanban' && (
        <div className="space-y-4">
          {/* Quick-Filter Pills Strip (Auto-wrap) */}
          <div className="flex flex-wrap items-center gap-1.5 rounded-xl border border-slate-200 bg-white p-2 dark:border-slate-800 dark:bg-slate-900 shadow-2xs">
            <button
              type="button"
              onClick={() => setSelectedColumnFilter('ALL')}
              className={`rounded-lg px-3 py-1.5 text-xs font-semibold transition cursor-pointer ${
                selectedColumnFilter === 'ALL'
                  ? 'bg-tlu-blue text-white shadow-xs'
                  : 'text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800'
              }`}
            >
              {t('curriculum.viz.group_all')}
            </button>
            {roadmapColumns.map((col) => (
              <button
                key={`pill-${col.id}`}
                type="button"
                onClick={() => setSelectedColumnFilter(col.id)}
                className={`inline-flex items-center gap-1 rounded-lg px-3 py-1.5 text-xs font-semibold transition cursor-pointer ${
                  selectedColumnFilter === col.id
                    ? 'bg-tlu-blue text-white shadow-xs'
                    : 'text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800'
                }`}
              >
                <span>{col.title}</span>
                <span className="text-[10px] opacity-75">({col.courses.length})</span>
              </button>
            ))}
          </div>

          {/* Responsive Columns Grid - Auto-wrapping naturally down the page, NO horizontal scroll */}
          <div className="grid grid-cols-[repeat(auto-fill,minmax(min(100%,270px),1fr))] gap-4 min-w-0">
            {roadmapColumns
              .filter((col) => selectedColumnFilter === 'ALL' || col.id === selectedColumnFilter)
              .map((col) => {
                const visibleCourses = col.courses.filter((c) => {
                  if (selectedBlock && c.knowledge_block !== selectedBlock) return false;
                  if (searchQuery.trim()) {
                    const q = searchQuery.trim().toUpperCase();
                    const codeMatch = c.course_code.toUpperCase().includes(q);
                    const nameMatch = c.course_name.toUpperCase().includes(q);
                    if (!codeMatch && !nameMatch) return false;
                  }
                  return true;
                });

                return (
                  <div
                    key={`kanban-col-${col.id}`}
                    className={`flex flex-col rounded-2xl border border-slate-200 bg-slate-50/70 shadow-xs dark:border-blue-900/30 dark:bg-[#111A35]/60 overflow-hidden min-w-0 ${col.headerBorder}`}
                  >
                    {/* Column Header */}
                    <div className="border-b border-slate-200 bg-white px-3.5 py-3 dark:border-slate-800 dark:bg-slate-900">
                      <div className="flex items-center justify-between gap-2 min-w-0">
                        <div className="flex items-center gap-2 min-w-0 flex-1">
                          <div className="shrink-0">{col.icon}</div>
                          <span
                            className="font-bold text-slate-900 dark:text-slate-100 text-sm truncate"
                            title={col.title}
                          >
                            {col.title}
                          </span>
                        </div>
                        <span className={`rounded-full px-2 py-0.5 text-xs font-bold shrink-0 whitespace-nowrap ${col.badgeStyle}`}>
                          {col.totalCredits} {t('curriculum.viz.credits_unit')}
                        </span>
                      </div>
                      <p className="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400 truncate" title={col.subtitle}>
                        {col.subtitle}
                      </p>
                    </div>

                    {/* Cards Container with clean vertical scrolling */}
                    <div className="flex-1 space-y-2.5 p-3 overflow-y-auto max-h-[520px] min-h-[140px] scrollbar-thin">
                      {visibleCourses.length === 0 ? (
                        <div className="py-8 text-center text-xs text-slate-400">
                          {t('common.no_data')}
                        </div>
                      ) : (
                        visibleCourses.map((c) => {
                          const codeUpper = c.course_code.toUpperCase();
                          const isSelected = selectedCourseCode.toUpperCase() === codeUpper;
                          const isAncestor = activeDependencySets.ancestors.has(codeUpper);
                          const isDescendant = activeDependencySets.descendants.has(codeUpper);
                          const isDimmed =
                            selectedCourseCode && !isSelected && !isAncestor && !isDescendant;

                          const prereqCount = (incomingEdgesMap.get(codeUpper) || []).length;
                          const unlockCount = (outgoingEdgesMap.get(codeUpper) || []).length;
                          const isElective = isElectiveCourse(c);
                          const creditBadge = extractCreditBadge(c.prerequisites_text);

                          return (
                            <div
                              key={c.id}
                              onClick={() => handleSelectCourse(c.course_code)}
                              className={`cursor-pointer rounded-xl border p-3.5 transition-all shadow-2xs ${
                                isSelected
                                  ? 'border-tlu-blue bg-blue-50/90 ring-2 ring-blue-500/40 shadow-md dark:border-tlu-cyan dark:bg-blue-950/80'
                                  : isAncestor
                                    ? 'border-amber-400 bg-amber-50/80 ring-1 ring-amber-400/50 dark:border-amber-700 dark:bg-amber-950/60'
                                    : isDescendant
                                      ? 'border-emerald-400 bg-emerald-50/80 ring-1 ring-emerald-400/50 dark:border-emerald-700 dark:bg-emerald-950/60'
                                      : 'border-slate-200/90 bg-white hover:border-blue-300 hover:shadow-xs dark:border-slate-800 dark:bg-slate-900 dark:hover:border-slate-700'
                              } ${isDimmed ? 'opacity-35 hover:opacity-100' : 'opacity-100'}`}
                            >
                              <div className="flex items-center justify-between gap-1.5">
                                <span className="rounded-md bg-blue-50 px-2 py-0.5 font-mono text-xs font-bold text-tlu-blue dark:bg-blue-950/80 dark:text-tlu-cyan">
                                  {c.course_code}
                                </span>
                                <div className="flex items-center gap-1.5">
                                  {isElective && (
                                    <span className="rounded bg-purple-50 px-1.5 py-0.5 text-[10px] font-semibold text-purple-700 dark:bg-purple-950/70 dark:text-purple-300">
                                      {t('curriculum.viz.badge_elective')}
                                    </span>
                                  )}
                                  <span className="text-xs font-bold text-slate-600 dark:text-slate-300">
                                    {c.credits} TC
                                  </span>
                                </div>
                              </div>

                              <h4
                                className="mt-1.5 text-xs font-bold leading-snug text-slate-900 dark:text-slate-100 line-clamp-2"
                                title={c.course_name}
                              >
                                {c.course_name}
                              </h4>

                              <div className="mt-2.5 flex flex-wrap items-center gap-1.5">
                                <span className="rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300 truncate max-w-36">
                                  {c.knowledge_block}
                                </span>

                                {creditBadge && (
                                  <span
                                    title={c.prerequisites_text}
                                    className="inline-flex items-center rounded-md bg-amber-500/10 border border-amber-500/25 px-1.5 py-0.5 text-[10px] font-bold text-amber-800 dark:bg-amber-950/50 dark:border-amber-800/60 dark:text-amber-300"
                                  >
                                    <Award className="mr-1 h-3 w-3 text-amber-600 dark:text-amber-400" />
                                    {creditBadge}
                                  </span>
                                )}

                                {prereqCount > 0 && (
                                  <span className="inline-flex items-center rounded-md bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold text-amber-800 dark:bg-amber-950 dark:text-amber-300">
                                    <GitBranch className="mr-1 h-3 w-3" />
                                    {t('curriculum.viz.badge_prereq_count', { count: prereqCount })}
                                  </span>
                                )}


                                {unlockCount > 0 && (
                                  <span className="inline-flex items-center rounded-md bg-emerald-100 px-1.5 py-0.5 text-[10px] font-semibold text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300">
                                    <Sparkles className="mr-1 h-3 w-3" />
                                    {t('curriculum.viz.badge_unlock_count', { count: unlockCount })}
                                  </span>
                                )}
                              </div>
                            </div>
                          );
                        })
                      )}
                    </div>
                  </div>
                );
              })}
          </div>
        </div>
      )}

      {/* VIEW 2: Visual Stage-Based DAG Flowchart (Responsive auto-wrapping grid, NO horizontal scroll) */}
      {viewMode === 'dag' && (
        <div className="rounded-xl sm:rounded-2xl border border-slate-200 bg-white p-3.5 sm:p-5 shadow-xs dark:border-blue-900/30 dark:bg-[#111A35] min-w-0">
          <div className="grid grid-cols-[repeat(auto-fill,minmax(min(100%,270px),1fr))] gap-4 min-w-0">
            {dagStageGroups.map((stage) => (
              <div
                key={`dag-col-${stage.id}`}
                className="flex flex-col rounded-2xl border border-slate-200 bg-slate-50/70 p-3.5 shadow-xs dark:border-blue-900/30 dark:bg-[#111A35]/60 space-y-3 min-w-0"
              >
                <div className="border-b border-slate-200 pb-2.5 dark:border-slate-800">
                  <h4 className="font-bold text-sm text-slate-900 dark:text-slate-100">
                    {stage.name}
                  </h4>
                  <p className="text-xs text-slate-400 mt-0.5">
                    {stage.desc}
                  </p>
                </div>

                <div className="space-y-2 max-h-[520px] overflow-y-auto pr-1 scrollbar-thin">
                  {stage.courses
                    .filter((c) => {
                      if (selectedBlock && c.knowledge_block !== selectedBlock) return false;
                      if (searchQuery.trim()) {
                        const q = searchQuery.trim().toUpperCase();
                        return (
                          c.course_code.toUpperCase().includes(q) ||
                          c.course_name.toUpperCase().includes(q)
                        );
                      }
                      return true;
                    })
                    .map((c) => {
                      const codeUpper = c.course_code.toUpperCase();
                      const isSelected = selectedCourseCode.toUpperCase() === codeUpper;
                      const isAncestor = activeDependencySets.ancestors.has(codeUpper);
                      const isDescendant = activeDependencySets.descendants.has(codeUpper);
                      const isDimmed =
                        selectedCourseCode && !isSelected && !isAncestor && !isDescendant;
                      const creditBadge = extractCreditBadge(c.prerequisites_text);

                      return (
                        <div
                          key={`dag-node-${c.id}`}
                          onClick={() => handleSelectCourse(c.course_code)}
                          className={`rounded-xl border p-3 transition-all cursor-pointer shadow-2xs ${
                            isSelected
                              ? 'border-tlu-blue bg-blue-50 ring-2 ring-blue-500/40 dark:border-tlu-cyan dark:bg-blue-950'
                              : isAncestor
                                ? 'border-amber-400 bg-amber-50 ring-1 ring-amber-400 dark:border-amber-700 dark:bg-amber-950'
                                : isDescendant
                                  ? 'border-emerald-400 bg-emerald-50 ring-1 ring-emerald-400 dark:border-emerald-700 dark:bg-emerald-950'
                                  : 'border-slate-200 bg-white hover:border-blue-300 dark:border-slate-800 dark:bg-slate-900'
                          } ${isDimmed ? 'opacity-30' : 'opacity-100'}`}
                        >
                          <div className="flex items-center justify-between">
                            <span className="font-mono text-xs font-bold text-tlu-navy dark:text-tlu-cyan">
                              {c.course_code}
                            </span>
                            <span className="text-[11px] font-bold text-slate-500">
                              {c.credits} TC
                            </span>
                          </div>
                          <p className="mt-1 text-xs font-semibold text-slate-800 dark:text-slate-200 line-clamp-2">
                            {c.course_name}
                          </p>
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
                        </div>
                      );
                    })}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* VIEW 3: Prerequisite Matrix List (With Full Names) */}
      {viewMode === 'table' && (
        <div className="rounded-xl sm:rounded-2xl border border-slate-200 bg-white p-3.5 sm:p-5 shadow-xs dark:border-blue-900/30 dark:bg-[#111A35]">
          <div className="flex items-center justify-between mb-4 pb-2 border-b border-slate-100 dark:border-slate-800">
            <div className="text-xs font-bold uppercase tracking-wider text-slate-400">
              {t('curriculum.viz.all_dependencies', { count: filteredEdges.length })}
            </div>
            {selectedCourseCode && (
              <button
                type="button"
                onClick={() => setSelectedCourseCode('')}
                className="text-xs text-tlu-blue hover:underline cursor-pointer"
              >
                {t('curriculum.viz.inspector_clear')}
              </button>
            )}
          </div>

          {filteredEdges.length === 0 ? (
            <p className="py-12 text-center text-xs text-slate-400 dark:text-slate-500">
              {t('curriculum.viz.no_edges')}
            </p>
          ) : (
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              {filteredEdges.map((edge, idx) => {
                const fromCode = (edge.from_course_code || edge.from || '').toUpperCase();
                const toCode = (edge.to_course_code || edge.to || '').toUpperCase();
                const fromObj = courseMap.get(fromCode);
                const toObj = courseMap.get(toCode);

                const isEdgeSelected =
                  selectedCourseCode &&
                  (fromCode === selectedCourseCode.toUpperCase() ||
                    toCode === selectedCourseCode.toUpperCase());

                return (
                  <div
                    key={`mat-${fromCode}-${toCode}-${idx}`}
                    className={`flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-xl border p-3 sm:p-3.5 transition-all shadow-2xs ${
                      isEdgeSelected
                        ? 'border-tlu-blue bg-blue-50/70 ring-2 ring-blue-500/20 dark:border-blue-700 dark:bg-blue-950/60'
                        : 'border-slate-200/80 bg-slate-50/40 hover:border-blue-200 hover:bg-white dark:border-slate-800 dark:bg-slate-900/60 dark:hover:bg-slate-900'
                    }`}
                  >
                    {/* From Course (Prerequisite) */}
                    <div
                      onClick={() => handleSelectCourse(fromCode)}
                      className="min-w-0 flex-1 cursor-pointer"
                    >
                      <div className="flex items-center gap-1.5">
                        <span className="rounded bg-blue-100 px-2 py-0.5 font-mono text-xs font-bold text-tlu-navy dark:bg-blue-950/80 dark:text-blue-300">
                          {fromCode}
                        </span>
                        <span className="text-xs text-slate-400 font-medium">
                          ({fromObj?.credits || edge.from_credits || 0} TC)
                        </span>
                      </div>
                      <p
                        className="mt-1 text-xs font-bold text-slate-800 dark:text-slate-200 truncate"
                        title={fromObj?.course_name || edge.from_course_name || fromCode}
                      >
                        {fromObj?.course_name || edge.from_course_name || fromCode}
                      </p>
                    </div>

                    <div className="flex items-center justify-center shrink-0 text-slate-400 px-1">
                      <ArrowRight className="h-4 w-4" />
                    </div>

                    {/* To Course (Target Course) */}
                    <div
                      onClick={() => handleSelectCourse(toCode)}
                      className="min-w-0 flex-1 cursor-pointer"
                    >
                      <div className="flex items-center gap-1.5">
                        <span className="rounded bg-emerald-100 px-2 py-0.5 font-mono text-xs font-bold text-emerald-800 dark:bg-emerald-950/80 dark:text-emerald-300">
                          {toCode}
                        </span>
                        <span className="text-xs text-slate-400 font-medium">
                          ({toObj?.credits || edge.to_credits || 0} TC)
                        </span>
                      </div>
                      <p
                        className="mt-1 text-xs font-bold text-slate-800 dark:text-slate-200 truncate"
                        title={toObj?.course_name || edge.to_course_name || toCode}
                      >
                        {toObj?.course_name || edge.to_course_name || toCode}
                      </p>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
