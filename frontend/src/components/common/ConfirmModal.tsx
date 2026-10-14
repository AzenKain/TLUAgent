import React from 'react';
import { useTranslation } from 'react-i18next';
import { AlertTriangle, Info, CheckCircle2, Trash2 } from 'lucide-react';
import { Modal } from './Modal';
import { LoadingSpinner } from './LoadingSpinner';

export interface ConfirmModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm?: () => void;
  title: string;
  message: string | React.ReactNode;
  confirmText?: string;
  cancelText?: string;
  variant?: 'danger' | 'warning' | 'info' | 'success';
  isLoading?: boolean;
  maxWidth?: 'sm' | 'md' | 'lg';
}

// ConfirmModal provides a standard confirmation or alert dialog with clean light dark overlay.
export const ConfirmModal: React.FC<ConfirmModalProps> = ({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText,
  cancelText,
  variant = 'danger',
  isLoading = false,
  maxWidth = 'sm',
}) => {
  const { t } = useTranslation();

  const isAlertOnly = !onConfirm;

  const iconClasses = {
    danger: 'bg-rose-100 text-rose-600 dark:bg-rose-950/50 dark:text-rose-400',
    warning: 'bg-amber-100 text-amber-600 dark:bg-amber-950/50 dark:text-amber-400',
    info: 'bg-blue-100 text-tlu-blue dark:bg-blue-950/50 dark:text-tlu-cyan',
    success: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-950/50 dark:text-emerald-400',
  };

  const confirmBtnClasses = {
    danger: 'bg-rose-600 hover:bg-rose-700 text-white shadow-rose-600/20',
    warning: 'bg-amber-600 hover:bg-amber-700 text-white shadow-amber-600/20',
    info: 'bg-tlu-blue hover:bg-tlu-navy-light text-white shadow-tlu-blue/20',
    success: 'bg-emerald-600 hover:bg-emerald-700 text-white shadow-emerald-600/20',
  };

  const renderIcon = () => {
    switch (variant) {
      case 'danger':
        return <Trash2 className="h-5 w-5" />;
      case 'warning':
        return <AlertTriangle className="h-5 w-5" />;
      case 'success':
        return <CheckCircle2 className="h-5 w-5" />;
      case 'info':
      default:
        return <Info className="h-5 w-5" />;
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={title} maxWidth={maxWidth}>
      <div className="space-y-4">
        <div className="flex items-start gap-3.5">
          <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-xl ${iconClasses[variant]}`}>
            {renderIcon()}
          </div>
          <div className="min-w-0 flex-1 pt-1 text-sm leading-relaxed text-slate-600 dark:text-slate-300">
            {typeof message === 'string' ? <p>{message}</p> : message}
          </div>
        </div>

        <div className="flex flex-col-reverse gap-2 border-t border-slate-100 pt-4 dark:border-slate-800 sm:flex-row sm:justify-end sm:gap-3">
          {!isAlertOnly && (
            <button
              type="button"
              onClick={onClose}
              disabled={isLoading}
              className="inline-flex min-h-11 w-full items-center justify-center rounded-lg border border-slate-200 px-4 py-2.5 text-sm font-medium text-slate-700 transition hover:bg-slate-100 disabled:opacity-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800 sm:w-auto sm:min-h-0 sm:py-2"
            >
              {cancelText || t('common.cancel')}
            </button>
          )}
          <button
            type="button"
            onClick={isAlertOnly ? onClose : onConfirm}
            disabled={isLoading}
            className={`inline-flex min-h-11 w-full items-center justify-center gap-2 rounded-lg px-4 py-2.5 text-sm font-semibold shadow-md transition disabled:opacity-50 sm:w-auto sm:min-h-0 sm:py-2 ${confirmBtnClasses[variant]}`}
          >
            {isLoading && <LoadingSpinner size="sm" />}
            <span>
              {isAlertOnly
                ? confirmText || t('common.close')
                : confirmText || (variant === 'danger' ? t('common.delete') : t('common.confirm'))}
            </span>
          </button>
        </div>
      </div>
    </Modal>
  );
};
