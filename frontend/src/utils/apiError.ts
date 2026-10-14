interface ValidationErrorItem {
  failed_field?: string;
  tag?: string;
  value?: string;
  message?: string;
}

interface ApiResponsePayload {
  status?: boolean;
  message?: string;
  errors?: ValidationErrorItem[] | string[] | string | null;
}

interface ErrorWithResponse {
  response?: {
    data?: ApiResponsePayload;
  };
  message?: string;
}

/** Extracts a human-readable error message from an API error or returns fallback. */
export function getApiErrorMessage(error: unknown, fallback: string = ''): string {
  if (!error) {
    return fallback;
  }

  const err = error as ErrorWithResponse;
  const data = err.response?.data;

  if (data) {
    if (Array.isArray(data.errors) && data.errors.length > 0) {
      const messages = data.errors
        .map((item) => {
          if (typeof item === 'string') {
            return item;
          }
          return item.message || item.failed_field || '';
        })
        .filter((msg) => msg.trim() !== '');

      if (messages.length > 0) {
        return messages.join('; ');
      }
    }

    if (typeof data.errors === 'string' && data.errors.trim() !== '') {
      return data.errors;
    }

    if (typeof data.message === 'string' && data.message.trim() !== '') {
      return data.message;
    }
  }

  if (typeof err.message === 'string' && err.message.trim() !== '') {
    return err.message;
  }

  return fallback;
}
