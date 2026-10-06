import { apiClient } from '@/lib/axios';

export interface JobDTO {
  id: string;
  type: string;
  status: 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED' | 'CANCELLED';
  progress: number;
  payload_json?: string;
  result_json?: string;
  error_message?: string;
  started_at?: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
}

export interface JobScheduleDTO {
  id: string;
  name: string;
  task_type?: string;
  job_type: string;
  cron_expr?: string;
  interval_minutes?: number;
  interval_sec?: number;
  payload_json?: string;
  enabled?: boolean;
  is_active: boolean;
  last_run_at?: string;
  next_run_at?: string;
  created_at: string;
  updated_at: string;
}

export interface UpsertJobScheduleDTO {
  name: string;
  task_type?: string;
  job_type: string;
  cron_expr?: string;
  interval_minutes?: number;
  interval_sec?: number;
  payload_json?: string;
  enabled?: boolean;
  is_active: boolean;
}

export interface TriggerJobDTO {
  type: string;
  payload_json?: string;
}

export interface TaskDTO {
  type: string;
  name: string;
  description: string;
}

export const jobService = {
  async listJobs(params?: { status?: string; type?: string; limit?: number; offset?: number }) {
    const response = await apiClient.get<{ status: boolean; data: { items: JobDTO[]; total: number } }>('/admin/jobs', {
      params,
    });
    return response.data.data;
  },

  async getJob(id: string) {
    const response = await apiClient.get<{ status: boolean; data: JobDTO }>(`/admin/jobs/${id}`);
    return response.data.data;
  },

  async listTasks() {
    const response = await apiClient.get<{ status: boolean; data: TaskDTO[] }>('/admin/jobs/tasks');
    return response.data.data;
  },

  async triggerJob(data: TriggerJobDTO) {
    const response = await apiClient.post<{ status: boolean; data: JobDTO }>('/admin/jobs/trigger', data);
    return response.data.data;
  },

  async listSchedules() {
    const response = await apiClient.get<{ status: boolean; data: JobScheduleDTO[] }>('/admin/jobs/schedules');
    return response.data.data;
  },

  async createSchedule(data: UpsertJobScheduleDTO) {
    const response = await apiClient.post<{ status: boolean; data: JobScheduleDTO }>('/admin/jobs/schedules', data);
    return response.data.data;
  },

  async updateSchedule(id: string, data: UpsertJobScheduleDTO) {
    const response = await apiClient.put<{ status: boolean; data: JobScheduleDTO }>(`/admin/jobs/schedules/${id}`, data);
    return response.data.data;
  },

  async deleteSchedule(id: string) {
    const response = await apiClient.delete<{ status: boolean }>(`/admin/jobs/schedules/${id}`);
    return response.data;
  },

  async runScheduleNow(id: string) {
    const response = await apiClient.post<{ status: boolean; data: JobDTO }>(`/admin/jobs/schedules/${id}/run-now`);
    return response.data.data;
  },
};
