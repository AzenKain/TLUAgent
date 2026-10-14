import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  CheckCircle2,
  XCircle,
  MessageSquare,
  TrendingUp,
  Plus,
  RefreshCw,
  AlertCircle,
} from 'lucide-react';
import type { TranscriptDiffResultResponse, StudentCourseGradeItem } from '@/types/transcript';
import { transcriptService } from '@/services/transcriptService';

// Props for the TranscriptConfirmationCard component.
export interface TranscriptConfirmationCardProps {
  diff: TranscriptDiffResultResponse;
  extractedGrades?: StudentCourseGradeItem[];
  actionState?: 'pending' | 'confirmed' | 'cancelled';
  onConfirmSuccess?: () => void;
  onCancel?: () => void;
  onChatOption?: (courseCodes: string[]) => void;
}

// TranscriptConfirmationCard displays OCR transcript diffs with options to overwrite, dismiss, or chat.
export const TranscriptConfirmationCard: React.FC<TranscriptConfirmationCardProps> = ({
  diff,
  extractedGrades,
  actionState = 'pending',
  onConfirmSuccess,
  onCancel,
  onChatOption,
}) => {
  const { t } = useTranslation();
  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const handleConfirm = async () => {
    if (!extractedGrades || extractedGrades.length === 0) {
      return;
    }
    setIsSaving(true);
    setErrorMessage(null);
    try {
      await transcriptService.batchSaveTranscript(extractedGrades);
      if (onConfirmSuccess) {
        onConfirmSuccess();
      }
    } catch {
      setErrorMessage(t('transcript.save_error'));
    } finally {
      setIsSaving(false);
    }
  };

  const handleChat = () => {
    if (onChatOption) {
      const changedCodes = diff.items
        .filter((item) => item.change_type === 'NEW' || item.change_type === 'UPGRADED')
        .map((item) => item.course_code);
      onChatOption(changedCodes);
    }
  };

  return (
    <div className="mt-3.5 w-full rounded-2xl border border-blue-200/80 dark:border-blue-900/40 bg-linear-to-br from-blue-50/50 via-white to-slate-50/50 dark:from-[#111A35] dark:via-[#111A35]/90 dark:to-[#0B1021] p-4 shadow-2xs text-slate-800 dark:text-slate-100">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-3 border-b border-blue-100 dark:border-blue-900/30">
        <div>
          <div className="flex items-center gap-2 font-semibold text-sm text-tlu-navy dark:text-tlu-cyan">
            <TrendingUp className="w-4 h-4 text-tlu-blue dark:text-tlu-cyan" />
            <span>{t('transcript.card_title')}</span>
          </div>
          <div className="text-xs text-slate-600 dark:text-slate-400 mt-0.5 leading-relaxed">
            {t('transcript.card_desc')}
          </div>
        </div>
        <div className="flex items-center gap-1.5 flex-wrap">
          {diff.new_courses_count > 0 && (
            <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-100 dark:bg-emerald-950/70 text-emerald-700 dark:text-emerald-300">
              <Plus className="w-3 h-3" />
              <span>{t('transcript.new_courses_badge', { count: diff.new_courses_count })}</span>
            </span>
          )}
          {diff.upgraded_courses_count > 0 && (
            <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-blue-100 dark:bg-blue-950/70 text-blue-700 dark:text-blue-300">
              <RefreshCw className="w-3 h-3" />
              <span>{t('transcript.upgraded_courses_badge', { count: diff.upgraded_courses_count })}</span>
            </span>
          )}
        </div>
      </div>

      <div className="mt-3 max-h-64 overflow-y-auto rounded-xl border border-slate-200/80 dark:border-slate-800 bg-white/70 dark:bg-slate-950/50 shadow-inner">
        <table className="w-full text-left text-xs border-collapse">
          <thead className="sticky top-0 bg-slate-100/90 dark:bg-slate-900/90 backdrop-blur-xs text-slate-500 dark:text-slate-400 border-b border-slate-200 dark:border-slate-800">
            <tr>
              <th className="py-2 px-3 font-semibold">{t('transcript.course_code')}</th>
              <th className="py-2 px-3 font-semibold">{t('transcript.course_name')}</th>
              <th className="py-2 px-2 text-center font-semibold">{t('transcript.credits')}</th>
              <th className="py-2 px-2 text-center font-semibold">{t('transcript.old_grade')}</th>
              <th className="py-2 px-2 text-center font-semibold">{t('transcript.new_grade')}</th>
              <th className="py-2 px-3 text-center font-semibold">{t('transcript.change_type')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 font-mono text-[11px]">
            {diff.items.map((item, idx) => (
              <tr key={idx} className="hover:bg-blue-50/40 dark:hover:bg-blue-950/20 transition-colors">
                <td className="py-2 px-3 font-bold text-tlu-blue dark:text-tlu-cyan">
                  {item.course_code}
                </td>
                <td className="py-2 px-3 font-sans text-slate-700 dark:text-slate-300 truncate max-w-40">
                  {item.course_name}
                </td>
                <td className="py-2 px-2 text-center text-slate-600 dark:text-slate-400">
                  {item.credits}
                </td>
                <td className="py-2 px-2 text-center text-slate-500 dark:text-slate-400">
                  {item.change_type === 'NEW' ? '-' : `${item.old_score_10} (${item.old_grade_letter || '-'})`}
                </td>
                <td className="py-2 px-2 text-center font-semibold text-emerald-600 dark:text-emerald-400">
                  {item.new_score_10} ({item.new_grade_letter || '-'})
                </td>
                <td className="py-2 px-3 text-center font-sans">
                  {item.change_type === 'NEW' ? (
                    <span className="px-2 py-0.5 rounded-md bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300 text-[10px] font-medium border border-emerald-200 dark:border-emerald-800">
                      {t('transcript.type_new')}
                    </span>
                  ) : item.change_type === 'UPGRADED' ? (
                    <span className="px-2 py-0.5 rounded-md bg-blue-50 dark:bg-blue-950/60 text-tlu-blue dark:text-tlu-cyan text-[10px] font-medium border border-blue-200 dark:border-blue-800">
                      {t('transcript.type_upgraded')}
                    </span>
                  ) : item.change_type === 'LOWER' ? (
                    <span className="px-2 py-0.5 rounded-md bg-amber-50 dark:bg-amber-950/60 text-amber-700 dark:text-amber-300 text-[10px] font-medium border border-amber-200 dark:border-amber-800">
                      {t('transcript.type_lower')}
                    </span>
                  ) : (
                    <span className="px-2 py-0.5 rounded-md bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 text-[10px] font-medium">
                      {t('transcript.type_unchanged')}
                    </span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {errorMessage && (
        <div className="mt-2 text-xs text-rose-500 flex items-center gap-1.5">
          <AlertCircle className="w-3.5 h-3.5 shrink-0" />
          <span>{errorMessage}</span>
        </div>
      )}

      <div className="mt-3.5 pt-3 border-t border-blue-100 dark:border-blue-900/30 flex flex-wrap items-center justify-between gap-2.5">
        {actionState === 'confirmed' ? (
          <div className="w-full flex items-center gap-2 px-3 py-2 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200/80 dark:border-emerald-800 text-xs text-emerald-800 dark:text-emerald-200 font-medium">
            <CheckCircle2 className="w-4 h-4 text-emerald-600 dark:text-emerald-400 shrink-0" />
            <span>{t('transcript.confirmed_saved_badge')}</span>
          </div>
        ) : actionState === 'cancelled' ? (
          <div className="w-full flex items-center gap-2 px-3 py-2 rounded-xl bg-slate-100 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700 text-xs text-slate-700 dark:text-slate-300 font-medium">
            <XCircle className="w-4 h-4 text-slate-500 shrink-0" />
            <span>{t('transcript.cancelled_badge')}</span>
          </div>
        ) : (
          <div className="w-full flex flex-wrap items-center justify-end gap-2">
            <button
              type="button"
              onClick={handleChat}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-xs font-semibold text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition shadow-2xs cursor-pointer"
            >
              <MessageSquare className="w-3.5 h-3.5 text-tlu-blue" />
              <span>{t('transcript.btn_chat_option')}</span>
            </button>
            <button
              type="button"
              onClick={onCancel}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-slate-100 dark:bg-slate-800/80 text-xs font-semibold text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700 transition shadow-2xs cursor-pointer"
            >
              <XCircle className="w-3.5 h-3.5 text-slate-500" />
              <span>{t('transcript.btn_cancel_overwrite')}</span>
            </button>
            <button
              type="button"
              disabled={isSaving}
              onClick={handleConfirm}
              className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-tlu-navy hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-[#1541B8] text-white text-xs font-semibold shadow-xs transition disabled:opacity-50 cursor-pointer"
            >
              {isSaving ? (
                <>
                  <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                  <span>{t('transcript.saving')}</span>
                </>
              ) : (
                <>
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  <span>{t('transcript.btn_confirm_overwrite')}</span>
                </>
              )}
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
