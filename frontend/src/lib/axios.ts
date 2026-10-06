import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';
import { useAuthStore } from '@/stores/authStore';
import type { CommonResponse, AuthResponse } from '@/types';

export const CSRF_HEADER_NAME = 'X-CSRF-Token';

const CSRF_COOKIE_NAME = 'csrf_token';

export const getCsrfToken = (): string => {
  const entry = document.cookie
    .split('; ')
    .find((cookie) => cookie.startsWith(`${CSRF_COOKIE_NAME}=`));
  if (!entry) {
    return '';
  }
  const raw = entry.slice(CSRF_COOKIE_NAME.length + 1);
  try {
    return decodeURIComponent(raw);
  } catch {
    return raw;
  }
};

const attachCsrfHeader = (config: InternalAxiosRequestConfig): InternalAxiosRequestConfig => {
  if (config.url?.startsWith('/api/')) {
    config.url = config.url.slice(4);
  }
  const method = config.method?.toLowerCase();
  const isUnsafeMethod = method === 'post' || method === 'put' || method === 'patch' || method === 'delete';
  if (isUnsafeMethod && config.headers) {
    config.headers[CSRF_HEADER_NAME] = getCsrfToken();
  }
  return config;
};

export const apiClient = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
});

apiClient.interceptors.request.use(attachCsrfHeader, (error) => Promise.reject(error));

let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value?: unknown) => void;
  reject: (reason?: unknown) => void;
}> = [];

const processQueue = (error: unknown) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(undefined);
    }
  });
  failedQueue = [];
};

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<CommonResponse>) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { retry?: boolean };

    if (
      error.response?.status === 401 &&
      originalRequest &&
      !originalRequest.retry &&
      !originalRequest.url?.includes('/auth/login') &&
      !originalRequest.url?.includes('/setup') &&
      !originalRequest.url?.includes('/auth/refresh')
    ) {
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        })
          .then(() => apiClient(originalRequest))
          .catch((err) => Promise.reject(err));
      }

      originalRequest.retry = true;
      isRefreshing = true;

      try {
        const response = await axios.post<CommonResponse<AuthResponse>>('/api/auth/refresh', null, {
          headers: {
            [CSRF_HEADER_NAME]: getCsrfToken(),
          },
        });

        const refreshedUser = response.data.data?.user;
        if (refreshedUser) {
          useAuthStore.getState().setUser(refreshedUser);
        }
        processQueue(null);
        return apiClient(originalRequest);
      } catch (refreshErr) {
        processQueue(refreshErr);
        useAuthStore.getState().logout();
        return Promise.reject(refreshErr);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);
