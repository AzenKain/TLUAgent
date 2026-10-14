import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Play,
  Plus,
  Trash2,
  Eye,
  Pencil,
  Sliders,
  Calendar,
  Layers,
  Activity,
  ChevronDown,
  ChevronUp,
} from 'lucide-react';
import {
  jobService,
  type JobDTO,
  type JobScheduleDTO,
  type UpsertJobScheduleDTO,
} from '@/services/jobService';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { Badge } from '@/components/common/Badge';
import { Modal } from '@/components/common/Modal';
import { ConfirmModal } from '@/components/common/ConfirmModal';
import { PageHeader } from '@/components/common/PageHeader';
import { RefreshButton } from '@/components/common/RefreshButton';

const isCrawlerType = (type: string): boolean => type === 'crawl_notices' || type === 'crawl_news';

const getProgressPercent = (job: JobDTO): number => {
  if (job.status?.toUpperCase() === 'COMPLETED') return 100;
  if (!job.total || job.total <= 0) return 0;
  return Math.min(100, Math.max(0, Math.round((job.progress / job.total) * 100)));
};

export const JobManagePage: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [activeTab, setActiveTab] = useState<'jobs' | 'schedules' | 'tasks'>('jobs');
  const [page, setPage] = useState(1);
  const [pageSize] = useState(15);
  const [statusFilter, setStatusFilter] = useState('');
  const [typeFilter, setTypeFilter] = useState('');

  const [inspectModalOpen, setInspectModalOpen] = useState(false);
  const [selectedJob, setSelectedJob] = useState<JobDTO | null>(null);
  const [deletingScheduleId, setDeletingScheduleId] = useState<string | null>(null);

  const [scheduleModalOpen, setScheduleModalOpen] = useState(false);
  const [editingSchedule, setEditingSchedule] = useState<JobScheduleDTO | null>(null);
  const [scheduleMaxTimeDays, setScheduleMaxTimeDays] = useState(7);
  const [scheduleMaxPages, setScheduleMaxPages] = useState(5);
  const [scheduleShowAdvanced, setScheduleShowAdvanced] = useState(false);
  const [scheduleForm, setScheduleForm] = useState<UpsertJobScheduleDTO>({
    name: '',
    job_type: 'crawl_notices',
    cron_expr: '',
    interval_sec: 3600,
    payload_json: '{}',
    is_active: true,
  });

  const [triggerModalOpen, setTriggerModalOpen] = useState(false);
  const [triggerType, setTriggerType] = useState('crawl_notices');
  const [triggerPayload, setTriggerPayload] = useState('{}');
  const [triggerMaxTimeDays, setTriggerMaxTimeDays] = useState(7);
  const [triggerMaxPages, setTriggerMaxPages] = useState(5);
  const [triggerShowAdvanced, setTriggerShowAdvanced] = useState(false);

  const availableJobTypes = [
    { value: 'crawl_notices', labelKey: 'jobs.type_crawl_notices' },
    { value: 'crawl_news', labelKey: 'jobs.type_crawl_news' },
    { value: 'crawl_static_pages', labelKey: 'jobs.type_crawl_static_pages' },
    { value: 'clean_corpus', labelKey: 'jobs.type_clean_corpus' },
    { value: 'supersede_check', labelKey: 'jobs.type_supersede_check' },
  ];

  const getJobTypeLabel = (type: string) => {
    const normalized = type.replace('.', '_');
    const key = `jobs.type_${normalized}`;
    const translated = t(key);
    return translated !== key ? translated : type;
  };

  const [toastMessage, setToastMessage] = useState<string | null>(null);
  const [toastType, setToastType] = useState<'success' | 'error'>('success');

  const showToast = (message: string, type: 'success' | 'error' = 'success') => {
    setToastMessage(message);
    setToastType(type);
    setTimeout(() => setToastMessage(null), 4000);
  };

  const {
    data: jobsData,
    isLoading: isJobsLoading,
    refetch: refetchJobs,
  } = useQuery({
    queryKey: ['admin-jobs', statusFilter, typeFilter, page, pageSize],
    queryFn: () =>
      jobService.listJobs({
        status: statusFilter || undefined,
        type: typeFilter || undefined,
        limit: pageSize,
        offset: (page - 1) * pageSize,
      }),
    refetchInterval: 5000,
  });

  const {
    data: schedulesData,
    isLoading: isSchedulesLoading,
    refetch: refetchSchedules,
  } = useQuery({
    queryKey: ['admin-job-schedules'],
    queryFn: () => jobService.listSchedules(),
  });

  const { data: tasksData } = useQuery({
    queryKey: ['admin-job-tasks'],
    queryFn: () => jobService.listTasks(),
  });

  const triggerMutation = useMutation({
    mutationFn: (data: { type: string; payload_json?: string }) => jobService.triggerJob(data),
    onSuccess: () => {
      showToast(t('jobs.trigger_success'));
      setTriggerModalOpen(false);
      queryClient.invalidateQueries({ queryKey: ['admin-jobs'] });
    },
    onError: () => {
      showToast(t('common.error'), 'error');
    },
  });

  const runScheduleNowMutation = useMutation({
    mutationFn: (id: string) => jobService.runScheduleNow(id),
    onSuccess: () => {
      showToast(t('jobs.run_now_success'));
      queryClient.invalidateQueries({ queryKey: ['admin-jobs'] });
    },
    onError: () => {
      showToast(t('common.error'), 'error');
    },
  });

  const saveScheduleMutation = useMutation({
    mutationFn: (data: { id?: string; body: UpsertJobScheduleDTO }) => {
      if (data.id) {
        return jobService.updateSchedule(data.id, data.body);
      }
      return jobService.createSchedule(data.body);
    },
    onSuccess: () => {
      showToast(editingSchedule ? t('jobs.update_success') : t('jobs.create_success'));
      setScheduleModalOpen(false);
      setEditingSchedule(null);
      queryClient.invalidateQueries({ queryKey: ['admin-job-schedules'] });
    },
    onError: () => {
      showToast(t('common.error'), 'error');
    },
  });

  const deleteScheduleMutation = useMutation({
    mutationFn: (id: string) => jobService.deleteSchedule(id),
    onSuccess: () => {
      showToast(t('jobs.delete_success'));
      queryClient.invalidateQueries({ queryKey: ['admin-job-schedules'] });
    },
    onError: () => {
      showToast(t('common.error'), 'error');
    },
  });

  const handleOpenCreateSchedule = () => {
    setEditingSchedule(null);
    const defaultType = 'crawl_notices';
    const defaultPayload = JSON.stringify({ max_time_days: 7, max_pages: 5 }, null, 2);
    setScheduleMaxTimeDays(7);
    setScheduleMaxPages(5);
    setScheduleShowAdvanced(false);
    setScheduleForm({
      name: '',
      job_type: defaultType,
      cron_expr: '',
      interval_sec: 3600,
      payload_json: defaultPayload,
      is_active: true,
    });
    setScheduleModalOpen(true);
  };

  const handleOpenEditSchedule = (s: JobScheduleDTO) => {
    setEditingSchedule(s);
    const targetType = s.task_type || s.job_type;
    let parsedDays = targetType === 'crawl_news' ? 14 : 7;
    let parsedPages = 5;
    if (s.payload_json) {
      try {
        const p = JSON.parse(s.payload_json);
        if (typeof p.max_time_days === 'number') parsedDays = p.max_time_days;
        else if (typeof p.max_time === 'number') parsedDays = p.max_time;
        if (typeof p.max_pages === 'number') parsedPages = p.max_pages;
        else if (typeof p.limit === 'number') parsedPages = p.limit;
      } catch {
        // ignore
      }
    }
    setScheduleMaxTimeDays(parsedDays);
    setScheduleMaxPages(parsedPages);
    setScheduleShowAdvanced(false);
    setScheduleForm({
      name: s.name,
      job_type: targetType,
      cron_expr: s.cron_expr || '',
      interval_sec: s.interval_sec || (s.interval_minutes ? s.interval_minutes * 60 : 3600),
      payload_json: s.payload_json || '{}',
      is_active: s.enabled ?? s.is_active,
    });
    setScheduleModalOpen(true);
  };

  const handleScheduleTypeChange = (type: string) => {
    let newPayload = scheduleForm.payload_json;
    if (isCrawlerType(type)) {
      const days = type === 'crawl_news' ? 14 : 7;
      setScheduleMaxTimeDays(days);
      setScheduleMaxPages(5);
      newPayload = JSON.stringify({ max_time_days: days, max_pages: 5 }, null, 2);
    }
    setScheduleForm({ ...scheduleForm, job_type: type, payload_json: newPayload });
  };

  const handleScheduleScopeChange = (days: number, pages: number) => {
    setScheduleMaxTimeDays(days);
    setScheduleMaxPages(pages);
    const newPayload = JSON.stringify({ max_time_days: days, max_pages: pages }, null, 2);
    setScheduleForm({ ...scheduleForm, payload_json: newPayload });
  };

  const handleSchedulePayloadTextChange = (val: string) => {
    setScheduleForm({ ...scheduleForm, payload_json: val });
    try {
      const parsed = JSON.parse(val);
      if (typeof parsed.max_time_days === 'number') setScheduleMaxTimeDays(parsed.max_time_days);
      else if (typeof parsed.max_time === 'number') setScheduleMaxTimeDays(parsed.max_time);
      if (typeof parsed.max_pages === 'number') setScheduleMaxPages(parsed.max_pages);
      else if (typeof parsed.limit === 'number') setScheduleMaxPages(parsed.limit);
    } catch {
      // ignore
    }
  };

  const handleOpenTrigger = (type: string = 'crawl_notices') => {
    setTriggerType(type);
    if (isCrawlerType(type)) {
      const days = type === 'crawl_news' ? 14 : 7;
      setTriggerMaxTimeDays(days);
      setTriggerMaxPages(5);
      setTriggerPayload(JSON.stringify({ max_time_days: days, max_pages: 5 }, null, 2));
    } else {
      setTriggerMaxTimeDays(0);
      setTriggerMaxPages(0);
      setTriggerPayload('{}');
    }
    setTriggerShowAdvanced(false);
    setTriggerModalOpen(true);
  };

  const handleTriggerTypeChange = (type: string) => {
    setTriggerType(type);
    if (isCrawlerType(type)) {
      const days = type === 'crawl_news' ? 14 : 7;
      setTriggerMaxTimeDays(days);
      setTriggerMaxPages(5);
      setTriggerPayload(JSON.stringify({ max_time_days: days, max_pages: 5 }, null, 2));
    } else {
      setTriggerPayload('{}');
    }
  };

  const handleTriggerScopeChange = (days: number, pages: number) => {
    setTriggerMaxTimeDays(days);
    setTriggerMaxPages(pages);
    const newPayload = JSON.stringify({ max_time_days: days, max_pages: pages }, null, 2);
    setTriggerPayload(newPayload);
  };

  const handleTriggerPayloadTextChange = (val: string) => {
    setTriggerPayload(val);
    try {
      const parsed = JSON.parse(val);
      if (typeof parsed.max_time_days === 'number') setTriggerMaxTimeDays(parsed.max_time_days);
      else if (typeof parsed.max_time === 'number') setTriggerMaxTimeDays(parsed.max_time);
      if (typeof parsed.max_pages === 'number') setTriggerMaxPages(parsed.max_pages);
      else if (typeof parsed.limit === 'number') setTriggerMaxPages(parsed.limit);
    } catch {
      // ignore
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status?.toUpperCase()) {
      case 'COMPLETED':
        return <Badge variant="success">{t('jobs.status_completed')}</Badge>;
      case 'RUNNING':
        return <Badge variant="warning">{t('jobs.status_running')}</Badge>;
      case 'FAILED':
        return <Badge variant="danger">{t('jobs.status_failed')}</Badge>;
      case 'CANCELLED':
        return <Badge variant="info">{t('jobs.status_cancelled')}</Badge>;
      default:
        return <Badge variant="info">{t('jobs.status_pending')}</Badge>;
    }
  };

  return (
    <div className="space-y-6 max-w-7xl mx-auto pb-10">
      {toastMessage && (
        <div
          className={`fixed bottom-6 right-6 z-50 rounded-xl px-5 py-3 shadow-xl backdrop-blur-md transition-all ${
            toastType === 'success'
              ? 'bg-emerald-600/90 text-white'
              : 'bg-rose-600/90 text-white'
          }`}
        >
          {toastMessage}
        </div>
      )}

      <PageHeader
        icon={<Activity className="h-6 w-6" />}
        title={t('jobs.title')}
        subtitle={t('jobs.sub')}
        actions={
          <div className="flex items-center gap-2">
            <RefreshButton
              onClick={() => {
                if (activeTab === 'jobs') void refetchJobs();
                if (activeTab === 'schedules') void refetchSchedules();
              }}
              isLoading={isJobsLoading || isSchedulesLoading}
            />
            <button
              onClick={() => handleOpenTrigger('crawl_notices')}
              className="inline-flex min-h-10 items-center justify-center gap-2 rounded-xl bg-tlu-navy px-4 py-2 text-xs sm:text-sm font-semibold text-white shadow-xs hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-blue-600 transition cursor-pointer"
            >
              <Play className="h-4 w-4" />
              <span>{t('jobs.trigger_task')}</span>
            </button>
          </div>
        }
      />

      <div className="flex border-b border-slate-800">
        <button
          onClick={() => setActiveTab('jobs')}
          className={`flex items-center gap-2 border-b-2 px-5 py-3 text-sm font-medium transition-colors ${
            activeTab === 'jobs'
              ? 'border-tlu-navy text-tlu-navy dark:border-tlu-blue dark:text-blue-400'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <Layers className="h-4 w-4" />
          <span>{t('jobs.tabs.jobs')}</span>
        </button>
        <button
          onClick={() => setActiveTab('schedules')}
          className={`flex items-center gap-2 border-b-2 px-5 py-3 text-sm font-medium transition-colors ${
            activeTab === 'schedules'
              ? 'border-tlu-navy text-tlu-navy dark:border-tlu-blue dark:text-blue-400'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <Calendar className="h-4 w-4" />
          <span>{t('jobs.tabs.schedules')}</span>
        </button>
        <button
          onClick={() => setActiveTab('tasks')}
          className={`flex items-center gap-2 border-b-2 px-5 py-3 text-sm font-medium transition-colors ${
            activeTab === 'tasks'
              ? 'border-tlu-navy text-tlu-navy dark:border-tlu-blue dark:text-blue-400'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <Sliders className="h-4 w-4" />
          <span>{t('jobs.tabs.tasks')}</span>
        </button>
      </div>

      {activeTab === 'jobs' && (
        <div className="space-y-4">
          <div className="flex flex-wrap items-center gap-3">
            <select
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setPage(1);
              }}
              className="rounded-xl border border-slate-700 bg-slate-800/80 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-tlu-blue"
            >
              <option value="">{t('jobs.all_statuses')}</option>
              <option value="PENDING">{t('jobs.status_pending')}</option>
              <option value="RUNNING">{t('jobs.status_running')}</option>
              <option value="COMPLETED">{t('jobs.status_completed')}</option>
              <option value="FAILED">{t('jobs.status_failed')}</option>
              <option value="CANCELLED">{t('jobs.status_cancelled')}</option>
            </select>
            <select
              value={typeFilter}
              onChange={(e) => {
                setTypeFilter(e.target.value);
                setPage(1);
              }}
              className="rounded-xl border border-slate-700 bg-slate-800/80 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-tlu-blue"
            >
              <option value="">{t('jobs.all_types')}</option>
              {availableJobTypes.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {t(opt.labelKey)}
                </option>
              ))}
            </select>
          </div>

          <div className="overflow-hidden rounded-2xl border border-slate-800 bg-slate-900/60 shadow-xl backdrop-blur-sm dark:border-blue-900/30 dark:bg-[#111A35]">
            {isJobsLoading ? (
              <div className="flex h-64 items-center justify-center">
                <LoadingSpinner size="lg" />
              </div>
            ) : !jobsData?.items || jobsData.items.length === 0 ? (
              <div className="flex h-64 flex-col items-center justify-center text-slate-500">
                <Layers className="h-10 w-10 stroke-[1.5]" />
                <p className="mt-3 text-sm">{t('jobs.empty_jobs')}</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm text-slate-300">
                  <thead className="border-b border-slate-800 bg-slate-800/40 text-xs uppercase text-slate-400">
                    <tr>
                      <th className="px-6 py-4">{t('jobs.job_id')}</th>
                      <th className="px-6 py-4">{t('jobs.type')}</th>
                      <th className="px-6 py-4">{t('jobs.status')}</th>
                      <th className="px-6 py-4">{t('jobs.progress')}</th>
                      <th className="px-6 py-4">{t('jobs.created_at')}</th>
                      <th className="px-6 py-4 text-right">{t('common.actions')}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/60">
                    {jobsData.items.map((job) => (
                      <tr key={job.id} className="transition-colors hover:bg-slate-800/30">
                        <td className="px-6 py-4 font-mono text-xs text-tlu-blue dark:text-blue-400">{job.id}</td>
                        <td className="px-6 py-4 font-medium text-slate-200">{getJobTypeLabel(job.type)}</td>
                        <td className="px-6 py-4">{getStatusBadge(job.status)}</td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-2.5">
                            <div className="h-2 w-24 overflow-hidden rounded-full bg-slate-800">
                              <div
                                className="h-full bg-tlu-blue transition-all duration-300"
                                style={{ width: `${getProgressPercent(job)}%` }}
                              />
                            </div>
                            <div className="flex items-baseline gap-1.5">
                              <span className="text-xs font-medium text-slate-300">{getProgressPercent(job)}%</span>
                              {job.total !== undefined && job.total > 0 && (
                                <span className="text-[11px] text-slate-500 font-mono">
                                  ({job.progress}/{job.total})
                                </span>
                              )}
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4 text-xs text-slate-400">
                          {new Date(job.created_at).toLocaleString()}
                        </td>
                        <td className="px-6 py-4 text-right">
                          <button
                            onClick={() => {
                              setSelectedJob(job);
                              setInspectModalOpen(true);
                            }}
                            className="inline-flex items-center gap-1 rounded-lg border border-slate-700 bg-slate-800/60 px-2.5 py-1.5 text-xs font-medium text-slate-300 transition-colors hover:bg-slate-700 hover:text-white"
                          >
                            <Eye className="h-3.5 w-3.5" />
                            <span>{t('common.view')}</span>
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}

      {activeTab === 'schedules' && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={handleOpenCreateSchedule}
              className="flex items-center gap-2 rounded-xl bg-tlu-navy px-4 py-2.5 text-sm font-medium text-white shadow-lg shadow-blue-900/20 transition-all hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-blue-600"
            >
              <Plus className="h-4 w-4" />
              <span>{t('jobs.create_schedule')}</span>
            </button>
          </div>

          <div className="overflow-hidden rounded-2xl border border-slate-800 bg-slate-900/60 shadow-xl backdrop-blur-sm dark:border-blue-900/30 dark:bg-[#111A35]">
            {isSchedulesLoading ? (
              <div className="flex h-64 items-center justify-center">
                <LoadingSpinner size="lg" />
              </div>
            ) : !schedulesData || schedulesData.length === 0 ? (
              <div className="flex h-64 flex-col items-center justify-center text-slate-500">
                <Calendar className="h-10 w-10 stroke-[1.5]" />
                <p className="mt-3 text-sm">{t('jobs.empty_schedules')}</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm text-slate-300">
                  <thead className="border-b border-slate-800 bg-slate-800/40 text-xs uppercase text-slate-400">
                    <tr>
                      <th className="px-6 py-4">{t('jobs.schedule_name')}</th>
                      <th className="px-6 py-4">{t('jobs.type')}</th>
                      <th className="px-6 py-4">{t('jobs.interval_sec')}</th>
                      <th className="px-6 py-4">{t('jobs.status')}</th>
                      <th className="px-6 py-4">{t('jobs.last_run')}</th>
                      <th className="px-6 py-4">{t('jobs.next_run')}</th>
                      <th className="px-6 py-4 text-right">{t('common.actions')}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/60">
                    {schedulesData.map((s) => (
                      <tr key={s.id} className="transition-colors hover:bg-slate-800/30">
                        <td className="px-6 py-4 font-medium text-slate-200">{s.name}</td>
                        <td className="px-6 py-4 font-mono text-xs text-tlu-blue dark:text-blue-400">
                          {getJobTypeLabel(s.task_type || s.job_type)}
                        </td>
                        <td className="px-6 py-4 text-xs text-slate-300">
                          {s.interval_minutes ? `${s.interval_minutes}m` : s.interval_sec ? `${Math.round(s.interval_sec / 60)}m` : s.cron_expr || '-'}
                        </td>
                        <td className="px-6 py-4">
                          {(s.enabled ?? s.is_active) ? (
                            <Badge variant="success">{t('jobs.active')}</Badge>
                          ) : (
                            <Badge variant="info">{t('jobs.inactive')}</Badge>
                          )}
                        </td>
                        <td className="px-6 py-4 text-xs text-slate-400">
                          {s.last_run_at ? new Date(s.last_run_at).toLocaleString() : '-'}
                        </td>
                        <td className="px-6 py-4 text-xs text-slate-400">
                          {s.next_run_at ? new Date(s.next_run_at).toLocaleString() : '-'}
                        </td>
                        <td className="px-6 py-4 text-right">
                          <div className="flex items-center justify-end gap-2">
                            <button
                              onClick={() => runScheduleNowMutation.mutate(s.id)}
                              className="inline-flex items-center gap-1 rounded-lg border border-blue-700/60 bg-blue-900/30 px-2.5 py-1.5 text-xs font-medium text-blue-300 transition-colors hover:bg-blue-800/50 hover:text-white"
                            >
                              <Play className="h-3.5 w-3.5" />
                              <span>{t('jobs.run_now')}</span>
                            </button>
                            <button
                              onClick={() => handleOpenEditSchedule(s)}
                              className="inline-flex items-center gap-1 rounded-lg border border-slate-700 bg-slate-800/60 px-2.5 py-1.5 text-xs font-medium text-slate-300 transition-colors hover:bg-slate-700 hover:text-white"
                            >
                              <Pencil className="h-3.5 w-3.5" />
                              <span>{t('common.edit')}</span>
                            </button>
                            <button
                              onClick={() => setDeletingScheduleId(s.id)}
                              className="inline-flex items-center gap-1 rounded-lg border border-rose-900/50 bg-rose-950/30 px-2.5 py-1.5 text-xs font-medium text-rose-400 transition-colors hover:bg-rose-900/50 hover:text-rose-200"
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}

      {activeTab === 'tasks' && (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          {tasksData?.map((tItem) => (
            <div
              key={tItem.type}
              className="flex flex-col justify-between rounded-2xl border border-slate-800 bg-slate-900/60 p-5 shadow-xl backdrop-blur-sm dark:border-blue-900/30 dark:bg-[#111A35]"
            >
              <div>
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold text-slate-100">{getJobTypeLabel(tItem.type)}</h3>
                  <span className="font-mono text-xs text-tlu-blue dark:text-blue-400">{tItem.type}</span>
                </div>
                <p className="mt-2 text-sm text-slate-400">{tItem.description}</p>
              </div>
              <div className="mt-4 flex justify-end">
                <button
                  onClick={() => handleOpenTrigger(tItem.type)}
                  className="flex items-center gap-2 rounded-xl bg-tlu-navy px-4 py-2 text-xs font-medium text-white transition-all hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-blue-600"
                >
                  <Play className="h-3.5 w-3.5" />
                  <span>{t('jobs.trigger_task')}</span>
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {inspectModalOpen && selectedJob && (
        <Modal
          isOpen={inspectModalOpen}
          onClose={() => setInspectModalOpen(false)}
          title={`${t('jobs.job_id')}: ${selectedJob.id}`}
        >
          <div className="space-y-4 text-sm text-slate-300">
            <div>
              <label className="text-xs font-semibold uppercase text-slate-400">{t('jobs.type')}</label>
              <div className="mt-1 font-mono text-sm text-blue-300">
                {getJobTypeLabel(selectedJob.type)} <span className="text-xs text-slate-500">({selectedJob.type})</span>
              </div>
            </div>
            <div>
              <label className="text-xs font-semibold uppercase text-slate-400">{t('jobs.status')}</label>
              <div className="mt-1">{getStatusBadge(selectedJob.status)}</div>
            </div>
            <div>
              <label className="text-xs font-semibold uppercase text-slate-400">{t('jobs.progress')}</label>
              <div className="mt-1.5 flex items-center gap-3">
                <div className="h-2.5 w-36 overflow-hidden rounded-full bg-slate-800">
                  <div
                    className="h-full bg-tlu-blue transition-all duration-300"
                    style={{ width: `${getProgressPercent(selectedJob)}%` }}
                  />
                </div>
                <span className="text-xs font-semibold text-slate-200">
                  {getProgressPercent(selectedJob)}%
                </span>
                {selectedJob.total !== undefined && selectedJob.total > 0 && (
                  <span className="text-xs text-slate-400 font-mono">
                    ({selectedJob.progress} / {selectedJob.total})
                  </span>
                )}
              </div>
            </div>
            {selectedJob.payload_json && (
              <div>
                <label className="text-xs font-semibold uppercase text-slate-400">{t('jobs.payload')}</label>
                <pre className="mt-1 max-h-40 overflow-y-auto rounded-lg bg-slate-950 p-3 font-mono text-xs text-slate-300">
                  {selectedJob.payload_json}
                </pre>
              </div>
            )}
            {selectedJob.result_json && (
              <div>
                <label className="text-xs font-semibold uppercase text-slate-400">{t('jobs.result')}</label>
                <pre className="mt-1 max-h-40 overflow-y-auto rounded-lg bg-slate-950 p-3 font-mono text-xs text-emerald-400">
                  {selectedJob.result_json}
                </pre>
              </div>
            )}
            {(selectedJob.error_msg || selectedJob.error_message) && (
              <div>
                <label className="text-xs font-semibold uppercase text-slate-400">{t('jobs.error')}</label>
                <pre className="mt-1 max-h-40 overflow-y-auto rounded-lg bg-slate-950 p-3 font-mono text-xs text-rose-400">
                  {selectedJob.error_msg || selectedJob.error_message}
                </pre>
              </div>
            )}
          </div>
        </Modal>
      )}

      {scheduleModalOpen && (
        <Modal
          isOpen={scheduleModalOpen}
          onClose={() => setScheduleModalOpen(false)}
          title={editingSchedule ? t('jobs.edit_schedule') : t('jobs.create_schedule')}
        >
          <div className="space-y-4">
            <div>
              <label className="text-xs font-medium text-slate-300">{t('jobs.schedule_name')}</label>
              <input
                type="text"
                value={scheduleForm.name}
                onChange={(e) => setScheduleForm({ ...scheduleForm, name: e.target.value })}
                className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-tlu-blue"
              />
            </div>
            <div>
              <label className="text-xs font-medium text-slate-300">{t('jobs.type')}</label>
              <select
                value={scheduleForm.job_type}
                onChange={(e) => handleScheduleTypeChange(e.target.value)}
                className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-tlu-blue"
              >
                {availableJobTypes.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {t(opt.labelKey)}
                  </option>
                ))}
              </select>
            </div>

            {isCrawlerType(scheduleForm.job_type || '') && (
              <div className="rounded-xl border border-slate-700/60 bg-slate-800/40 p-4 space-y-3">
                <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-tlu-blue">
                  <Sliders className="h-3.5 w-3.5" />
                  <span>{t('jobs.crawler_settings')}</span>
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <label className="text-xs font-medium text-slate-300">
                      {t('jobs.max_time_days')}
                    </label>
                    <input
                      type="number"
                      min={0}
                      value={scheduleMaxTimeDays}
                      onChange={(e) =>
                        handleScheduleScopeChange(
                          Math.max(0, parseInt(e.target.value, 10) || 0),
                          scheduleMaxPages
                        )
                      }
                      className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-tlu-blue"
                    />
                    <p className="mt-1 text-[11px] text-slate-400">
                      {t('jobs.max_time_days_help')}
                    </p>
                  </div>
                  <div>
                    <label className="text-xs font-medium text-slate-300">
                      {t('jobs.max_pages')}
                    </label>
                    <input
                      type="number"
                      min={0}
                      value={scheduleMaxPages}
                      onChange={(e) =>
                        handleScheduleScopeChange(
                          scheduleMaxTimeDays,
                          Math.max(0, parseInt(e.target.value, 10) || 0)
                        )
                      }
                      className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-tlu-blue"
                    />
                    <p className="mt-1 text-[11px] text-slate-400">
                      {t('jobs.max_pages_help')}
                    </p>
                  </div>
                </div>
              </div>
            )}

            <div>
              <label className="text-xs font-medium text-slate-300">{t('jobs.interval_sec')}</label>
              <input
                type="number"
                value={scheduleForm.interval_sec || 3600}
                onChange={(e) =>
                  setScheduleForm({ ...scheduleForm, interval_sec: parseInt(e.target.value, 10) || 0 })
                }
                className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-tlu-blue"
              />
            </div>
            <div className="flex items-center gap-3">
              <input
                type="checkbox"
                id="is_active_check"
                checked={scheduleForm.is_active}
                onChange={(e) => setScheduleForm({ ...scheduleForm, is_active: e.target.checked })}
                className="h-4 w-4 rounded border-slate-700 bg-slate-900 text-tlu-blue focus:ring-tlu-blue"
              />
              <label htmlFor="is_active_check" className="text-sm text-slate-300">
                {t('jobs.active')}
              </label>
            </div>

            <div className="pt-1">
              <button
                type="button"
                onClick={() => setScheduleShowAdvanced(!scheduleShowAdvanced)}
                className="flex items-center gap-1.5 text-xs font-medium text-slate-400 hover:text-slate-200 cursor-pointer"
              >
                <Sliders className="h-3.5 w-3.5" />
                <span>{t('jobs.advanced_payload')}</span>
                {scheduleShowAdvanced ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
              </button>
              {scheduleShowAdvanced && (
                <div className="mt-2 space-y-1">
                  <p className="text-[11px] text-slate-400">
                    {t('jobs.advanced_payload_desc')}
                  </p>
                  <textarea
                    rows={4}
                    value={scheduleForm.payload_json || '{}'}
                    onChange={(e) => handleSchedulePayloadTextChange(e.target.value)}
                    className="w-full rounded-xl border border-slate-700 bg-slate-900 p-3 font-mono text-xs text-slate-200 focus:outline-none focus:ring-2 focus:ring-tlu-blue"
                  />
                </div>
              )}
            </div>

            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={() => setScheduleModalOpen(false)}
                className="rounded-xl border border-slate-700 px-4 py-2 text-sm font-medium text-slate-300 hover:bg-slate-800"
              >
                {t('common.cancel')}
              </button>
              <button
                onClick={() =>
                  saveScheduleMutation.mutate({
                    id: editingSchedule?.id,
                    body: scheduleForm,
                  })
                }
                className="rounded-xl bg-tlu-navy px-4 py-2 text-sm font-medium text-white hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-blue-600"
              >
                {t('common.save')}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {triggerModalOpen && (
        <Modal
          isOpen={triggerModalOpen}
          onClose={() => setTriggerModalOpen(false)}
          title={t('jobs.trigger_task')}
        >
          <div className="space-y-4">
            <div>
              <label className="text-xs font-medium text-slate-300">{t('jobs.type')}</label>
              <select
                value={triggerType}
                onChange={(e) => handleTriggerTypeChange(e.target.value)}
                className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-tlu-blue"
              >
                {availableJobTypes.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {t(opt.labelKey)}
                  </option>
                ))}
              </select>
            </div>

            {isCrawlerType(triggerType) && (
              <div className="rounded-xl border border-slate-700/60 bg-slate-800/40 p-4 space-y-3">
                <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-tlu-blue">
                  <Sliders className="h-3.5 w-3.5" />
                  <span>{t('jobs.crawler_settings')}</span>
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <label className="text-xs font-medium text-slate-300">
                      {t('jobs.max_time_days')}
                    </label>
                    <input
                      type="number"
                      min={0}
                      value={triggerMaxTimeDays}
                      onChange={(e) =>
                        handleTriggerScopeChange(
                          Math.max(0, parseInt(e.target.value, 10) || 0),
                          triggerMaxPages
                        )
                      }
                      className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-tlu-blue"
                    />
                    <p className="mt-1 text-[11px] text-slate-400">
                      {t('jobs.max_time_days_help')}
                    </p>
                  </div>
                  <div>
                    <label className="text-xs font-medium text-slate-300">
                      {t('jobs.max_pages')}
                    </label>
                    <input
                      type="number"
                      min={0}
                      value={triggerMaxPages}
                      onChange={(e) =>
                        handleTriggerScopeChange(
                          triggerMaxTimeDays,
                          Math.max(0, parseInt(e.target.value, 10) || 0)
                        )
                      }
                      className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-tlu-blue"
                    />
                    <p className="mt-1 text-[11px] text-slate-400">
                      {t('jobs.max_pages_help')}
                    </p>
                  </div>
                </div>
              </div>
            )}

            <div className="pt-1">
              <button
                type="button"
                onClick={() => setTriggerShowAdvanced(!triggerShowAdvanced)}
                className="flex items-center gap-1.5 text-xs font-medium text-slate-400 hover:text-slate-200 cursor-pointer"
              >
                <Sliders className="h-3.5 w-3.5" />
                <span>{t('jobs.advanced_payload')}</span>
                {triggerShowAdvanced ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
              </button>
              {triggerShowAdvanced && (
                <div className="mt-2 space-y-1">
                  <p className="text-[11px] text-slate-400">
                    {t('jobs.advanced_payload_desc')}
                  </p>
                  <textarea
                    rows={4}
                    value={triggerPayload}
                    onChange={(e) => handleTriggerPayloadTextChange(e.target.value)}
                    className="w-full rounded-xl border border-slate-700 bg-slate-900 p-3 font-mono text-xs text-slate-200 focus:outline-none focus:ring-2 focus:ring-tlu-blue"
                  />
                </div>
              )}
            </div>

            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={() => setTriggerModalOpen(false)}
                className="rounded-xl border border-slate-700 px-4 py-2 text-sm font-medium text-slate-300 hover:bg-slate-800"
              >
                {t('common.cancel')}
              </button>
              <button
                onClick={() =>
                  triggerMutation.mutate({
                    type: triggerType,
                    payload_json: triggerPayload,
                  })
                }
                className="rounded-xl bg-tlu-navy px-4 py-2 text-sm font-medium text-white hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-blue-600"
              >
                {t('jobs.run_now')}
              </button>
            </div>
          </div>
        </Modal>
      )}

      <ConfirmModal
        isOpen={Boolean(deletingScheduleId)}
        onClose={() => setDeletingScheduleId(null)}
        onConfirm={() => {
          if (deletingScheduleId) {
            deleteScheduleMutation.mutate(deletingScheduleId);
            setDeletingScheduleId(null);
          }
        }}
        title={t('common.delete')}
        message={t('jobs.delete_confirm')}
        variant="danger"
        isLoading={deleteScheduleMutation.isPending}
      />
    </div>
  );
};
