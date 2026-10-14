import React from 'react';
import { useTranslation } from 'react-i18next';
import { RefreshCw } from 'lucide-react';

interface RefreshButtonProps {
  onClick: () => void;
  isLoading?: boolean;
  label?: string;
  disabled?: boolean;
  className?: string;
  iconOnlyOnMobile?: boolean;
}

// RefreshButton provides a consistent, standardized reload button across all administrative and portal pages.
export const RefreshButton: React.FC<RefreshButtonProps> = ({
  onClick,
  isLoading = false,
  label,
  disabled = false,
  className = '',
  iconOnlyOnMobile = false,
}) => {
  const { t } = useTranslation();
  const text = label || t('common.refresh');

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled || isLoading}
      title={text}
      aria-label={text}
      className={`inline-flex items-center justify-center gap-1.5 px-3.5 py-2 text-xs font-semibold rounded-xl border border-slate-200 bg-white text-slate-700 hover:bg-slate-50 hover:text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700 shadow-xs transition-colors cursor-pointer disabled:opacity-50 min-h-10 shrink-0 ${className}`}
    >
      <RefreshCw
        className={`h-3.5 w-3.5 shrink-0 ${
          isLoading ? 'animate-spin text-tlu-blue' : 'text-slate-500 dark:text-slate-400'
        }`}
      />
      <span className={iconOnlyOnMobile ? 'hidden sm:inline' : 'inline'}>{text}</span>
    </button>
  );
};
