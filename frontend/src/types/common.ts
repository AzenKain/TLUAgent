export interface CommonResponse<T = unknown> {
  status: boolean;
  message?: string;
  data?: T;
  errors?: unknown;
}

export interface PaginationMeta {
  current_page: number;
  page_size: number;
  total_records: number;
  total_pages: number;
}

export interface PaginatedResponse<T = unknown> {
  status: boolean;
  message?: string;
  data: T[];
  errors?: unknown;
  pagination: PaginationMeta;
}
