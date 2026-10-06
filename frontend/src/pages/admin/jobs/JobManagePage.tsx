import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  RefreshCw,
  Play,
  Plus,
  Trash2,
  Eye,
  Sliders,
  Calendar,
  Layers,
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

  const [scheduleModalOpen, setScheduleModalOpen] = useState(false);
  const [editingSchedule, setEditingSchedule] = useState<JobScheduleDTO | null>(null);
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

  const availableJobTypes = [
    { value: 'crawl_notices', labelKey: 'jobs.type_crawl_notices' },
    { value: 'crawl_news', labelKey: 'jobs.type_crawl_news' },
    { value: 'crawl_static_pages', labelKey: 'jobs.type_crawl_static_pages' },
    { value: 'pipeline_clean_docs', labelKey: 'jobs.type_pipeline_clean_docs' },
    { value: 'vectorize_knowledge', labelKey: 'jobs.type_vectorize_knowledge' },
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
    setScheduleForm({
      name: '',
      job_type: 'crawl_notices',
      cron_expr: '',
      interval_sec: 3600,
      payload_json: '{}',
      is_active: true,
    });
    setScheduleModalOpen(true);
  };

  const handleOpenEditSchedule = (s: JobScheduleDTO) => {
    setEditingSchedule(s);
    setScheduleForm({
      name: s.name,
      job_type: s.task_type || s.job_type,
      cron_expr: s.cron_expr || '',
      interval_sec: s.interval_sec || (s.interval_minutes ? s.interval_minutes * 60 : 3600),
      payload_json: s.payload_json || '{}',
      is_active: s.enabled ?? s.is_active,
    });
    setScheduleModalOpen(true);
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
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
    <div className="space-y-6">
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

      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-100">{t('jobs.title')}</h1>
          <p className="mt-1 text-sm text-slate-400">{t('jobs.sub')}</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => {
              if (activeTab === 'jobs') refetchJobs();
              if (activeTab === 'schedules') refetchSchedules();
            }}
            className="flex items-center gap-2 rounded-xl border border-slate-700 bg-slate-800/80 px-4 py-2.5 text-sm font-medium text-slate-300 transition-colors hover:bg-slate-700/80 hover:text-white"
          >
            <RefreshCw className="h-4 w-4" />
            <span>{t('common.loading')}</span>
          </button>
          <button
            onClick={() => setTriggerModalOpen(true)}
            className="flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white shadow-lg shadow-indigo-600/20 transition-all hover:bg-indigo-500"
          >
            <Play className="h-4 w-4" />
            <span>{t('jobs.trigger_task')}</span>
          </button>
        </div>
      </div>

      <div className="flex border-b border-slate-800">
        <button
          onClick={() => setActiveTab('jobs')}
          className={`flex items-center gap-2 border-b-2 px-5 py-3 text-sm font-medium transition-colors ${
            activeTab === 'jobs'
              ? 'border-indigo-500 text-indigo-400'
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
              ? 'border-indigo-500 text-indigo-400'
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
              ? 'border-indigo-500 text-indigo-400'
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
              className="rounded-xl border border-slate-700 bg-slate-800/80 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
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
              className="rounded-xl border border-slate-700 bg-slate-800/80 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            >
              <option value="">{t('jobs.all_types')}</option>
              {availableJobTypes.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {t(opt.labelKey)}
                </option>
              ))}
            </select>
          </div>

          <div className="overflow-hidden rounded-2xl border border-slate-800 bg-slate-900/60 shadow-xl backdrop-blur-sm">
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
                        <td className="px-6 py-4 font-mono text-xs text-indigo-400">{job.id}</td>
                        <td className="px-6 py-4 font-medium text-slate-200">{getJobTypeLabel(job.type)}</td>
                        <td className="px-6 py-4">{getStatusBadge(job.status)}</td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-2">
                            <div className="h-2 w-24 overflow-hidden rounded-full bg-slate-800">
                              <div
                                className="h-full bg-indigo-500 transition-all duration-300"
                                style={{ width: `${Math.min(100, Math.max(0, job.progress))}%` }}
                              />
                            </div>
                            <span className="text-xs text-slate-400">{job.progress}%</span>
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
                            <span>{t('common.edit')}</span>
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
              className="flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white shadow-lg shadow-indigo-600/20 transition-all hover:bg-indigo-500"
            >
              <Plus className="h-4 w-4" />
              <span>{t('jobs.create_schedule')}</span>
            </button>
          </div>

          <div className="overflow-hidden rounded-2xl border border-slate-800 bg-slate-900/60 shadow-xl backdrop-blur-sm">
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
                        <td className="px-6 py-4 font-mono text-xs text-indigo-400">
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
                              className="inline-flex items-center gap-1 rounded-lg border border-indigo-700/60 bg-indigo-900/30 px-2.5 py-1.5 text-xs font-medium text-indigo-300 transition-colors hover:bg-indigo-800/50 hover:text-white"
                            >
                              <Play className="h-3.5 w-3.5" />
                              <span>{t('jobs.run_now')}</span>
                            </button>
                            <button
                              onClick={() => handleOpenEditSchedule(s)}
                              className="inline-flex items-center gap-1 rounded-lg border border-slate-700 bg-slate-800/60 px-2.5 py-1.5 text-xs font-medium text-slate-300 transition-colors hover:bg-slate-700 hover:text-white"
                            >
                              <Eye className="h-3.5 w-3.5" />
                              <span>{t('common.edit')}</span>
                            </button>
                            <button
                              onClick={() => {
                                if (window.confirm(t('jobs.delete_confirm'))) {
                                  deleteScheduleMutation.mutate(s.id);
                                }
                              }}
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
              className="flex flex-col justify-between rounded-2xl border border-slate-800 bg-slate-900/60 p-5 shadow-xl backdrop-blur-sm"
            >
              <div>
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold text-slate-100">{getJobTypeLabel(tItem.type)}</h3>
                  <span className="font-mono text-xs text-indigo-400">{tItem.type}</span>
                </div>
                <p className="mt-2 text-sm text-slate-400">{tItem.description}</p>
              </div>
              <div className="mt-4 flex justify-end">
                <button
                  onClick={() => {
                    setTriggerType(tItem.type);
                    setTriggerPayload('{}');
                    setTriggerModalOpen(true);
                  }}
                  className="flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-xs font-medium text-white transition-all hover:bg-indigo-500"
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
              <div className="mt-1 font-mono text-sm text-indigo-300">
                {getJobTypeLabel(selectedJob.type)} <span className="text-xs text-slate-500">({selectedJob.type})</span>
              </div>
            </div>
            <div>
              <label className="text-xs font-semibold uppercase text-slate-400">{t('jobs.status')}</label>
              <div className="mt-1">{getStatusBadge(selectedJob.status)}</div>
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
            {selectedJob.error_message && (
              <div>
                <label className="text-xs font-semibold uppercase text-slate-400">{t('jobs.error')}</label>
                <pre className="mt-1 max-h-40 overflow-y-auto rounded-lg bg-slate-950 p-3 font-mono text-xs text-rose-400">
                  {selectedJob.error_message}
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
                className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>
            <div>
              <label className="text-xs font-medium text-slate-300">{t('jobs.type')}</label>
              <select
                value={scheduleForm.job_type}
                onChange={(e) => setScheduleForm({ ...scheduleForm, job_type: e.target.value })}
                className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                {availableJobTypes.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {t(opt.labelKey)}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-xs font-medium text-slate-300">{t('jobs.interval_sec')}</label>
              <input
                type="number"
                value={scheduleForm.interval_sec || 3600}
                onChange={(e) =>
                  setScheduleForm({ ...scheduleForm, interval_sec: parseInt(e.target.value, 10) || 0 })
                }
                className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>
            <div className="flex items-center gap-3">
              <input
                type="checkbox"
                id="is_active_check"
                checked={scheduleForm.is_active}
                onChange={(e) => setScheduleForm({ ...scheduleForm, is_active: e.target.checked })}
                className="h-4 w-4 rounded border-slate-700 bg-slate-900 text-indigo-600 focus:ring-indigo-500"
              />
              <label htmlFor="is_active_check" className="text-sm text-slate-300">
                {t('jobs.active')}
              </label>
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
                className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500"
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
                onChange={(e) => setTriggerType(e.target.value)}
                className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                {availableJobTypes.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {t(opt.labelKey)}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-xs font-medium text-slate-300">{t('jobs.payload')}</label>
              <textarea
                rows={4}
                value={triggerPayload}
                onChange={(e) => setTriggerPayload(e.target.value)}
                className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 p-3 font-mono text-xs text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
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
                className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500"
              >
                {t('jobs.run_now')}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};
