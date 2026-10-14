import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  CheckCircle2,
  XCircle,
  Check,
  X,
  MessageSquare,
  Clock,
  Layers,
  ArrowRight,
  ShieldAlert,
} from 'lucide-react';
import type { TLUClawPreviewCardData } from '@/types/tluclaw';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';

interface TLUClawPreviewCardProps {
  card: TLUClawPreviewCardData;
  onConfirm?: (snapshotId: string, nonce: string, selectedKeys?: string[]) => Promise<void>;
  onCancel?: (snapshotId: string, nonce: string) => Promise<void>;
  onRefinePrompt?: (prompt: string) => void;
}

function actionBadgeClass(action: string): string {
  switch (action) {
    case 'CREATE':
    case 'INGEST':
      return 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20';
    case 'UPDATE':
    case 'UPSERT':
      return 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20';
    case 'DELETE':
      return 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20';
    case 'SUPERSEDE':
      return 'bg-cyan-500/10 text-cyan-700 dark:text-cyan-300 border-cyan-500/30';
    case 'PARTIALLY_AMEND':
      return 'bg-purple-500/10 text-purple-700 dark:text-purple-300 border-purple-500/30';
    case 'RETAIN':
      return 'bg-blue-500/10 text-blue-700 dark:text-blue-300 border-blue-500/30';
    default:
      return 'bg-muted text-muted-foreground border-border';
  }
}

export const TLUClawPreviewCard: React.FC<TLUClawPreviewCardProps> = ({
  card,
  onConfirm,
  onCancel,
  onRefinePrompt,
}) => {
  const { t } = useTranslation();
  const [status, setStatus] = useState<'STAGED' | 'COMMITTED' | 'CANCELLED'>('STAGED');
  const [loadingAction, setLoadingAction] = useState<'confirm' | 'cancel' | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(() => {
    return new Set((card.diff_items || []).map((item) => item.key));
  });

  const toggleSelectAll = () => {
    if (selectedKeys.size === (card.diff_items || []).length) {
      setSelectedKeys(new Set());
    } else {
      setSelectedKeys(new Set((card.diff_items || []).map((item) => item.key)));
    }
  };

  const toggleKey = (key: string) => {
    setSelectedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const handleConfirm = async () => {
    if (!onConfirm || loadingAction || selectedKeys.size === 0) return;
    setLoadingAction('confirm');
    try {
      await onConfirm(card.snapshot_id, card.nonce, Array.from(selectedKeys));
      setStatus('COMMITTED');
    } finally {
      setLoadingAction(null);
    }
  };

  const handleCancel = async () => {
    if (!onCancel || loadingAction) return;
    setLoadingAction('cancel');
    try {
      await onCancel(card.snapshot_id, card.nonce);
      setStatus('CANCELLED');
    } finally {
      setLoadingAction(null);
    }
  };

  const handleRefine = () => {
    if (onRefinePrompt) {
      onRefinePrompt(t('tluclaw.preview.refine_template', { batchId: card.batch_id }));
    }
  };

  const riskBadgeClass = () => {
    switch (card.blast_radius?.risk_level) {
      case 'HIGH':
        return 'bg-red-500/10 text-red-500 border-red-500/30';
      case 'MEDIUM':
        return 'bg-amber-500/10 text-amber-500 border-amber-500/30';
      default:
        return 'bg-emerald-500/10 text-emerald-500 border-emerald-500/30';
    }
  };

  return (
    <div className="my-4 rounded-xl border border-border bg-card p-5 shadow-sm transition-all duration-200">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border/60 pb-3">
        <div className="flex items-center gap-2">
          <Layers className="h-5 w-5 text-primary" />
          <h4 className="text-base font-semibold text-foreground">
            {t('tluclaw.preview.title')}
          </h4>
          <span className="rounded-md border border-primary/20 bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
            {card.action_type}
          </span>
          <span className="font-mono text-xs text-muted-foreground">
            #{card.batch_id}
          </span>
        </div>

        <div className="flex items-center gap-2">
          {status === 'STAGED' && (
            <span className="inline-flex items-center gap-1 rounded-full border border-amber-500/30 bg-amber-500/10 px-2.5 py-0.5 text-xs font-medium text-amber-500">
              <Clock className="h-3.5 w-3.5" />
              {t('tluclaw.preview.status_staged')}
            </span>
          )}
          {status === 'COMMITTED' && (
            <span className="inline-flex items-center gap-1 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-0.5 text-xs font-medium text-emerald-500">
              <CheckCircle2 className="h-3.5 w-3.5" />
              {t('tluclaw.preview.status_committed')}
            </span>
          )}
          {status === 'CANCELLED' && (
            <span className="inline-flex items-center gap-1 rounded-full border border-muted-foreground/30 bg-muted/40 px-2.5 py-0.5 text-xs font-medium text-muted-foreground">
              <XCircle className="h-3.5 w-3.5" />
              {t('tluclaw.preview.status_cancelled')}
            </span>
          )}
        </div>
      </div>

      {/* Summary */}
      <p className="mt-3 text-sm text-foreground/90 font-medium leading-relaxed">
        {card.summary}
      </p>

      {/* Blast Radius Box */}
      {card.blast_radius && (
        <div className="mt-4 rounded-lg border border-border/80 bg-muted/30 p-3.5">
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <ShieldAlert className="h-4 w-4 text-primary" />
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {t('tluclaw.preview.blast_radius_label')}
              </span>
            </div>
            <span className={`rounded-md border px-2 py-0.5 text-xs font-semibold ${riskBadgeClass()}`}>
              {t('tluclaw.preview.risk_level')}: {card.blast_radius.risk_level}
            </span>
          </div>

          <p className="mt-2 text-xs text-foreground/80 leading-relaxed">
            {card.blast_radius.impact_summary}
          </p>

          <div className="mt-2.5 flex flex-wrap items-center gap-2">
            {card.blast_radius.affected_cohorts?.map((coh) => (
              <span
                key={coh}
                className="rounded bg-primary/10 px-2 py-0.5 font-mono text-[11px] font-medium text-primary"
              >
                {t('tluclaw.preview.cohort')}: {coh}
              </span>
            ))}
            {card.blast_radius.affected_majors?.map((maj) => (
              <span
                key={maj}
                className="rounded bg-sky-500/10 px-2 py-0.5 font-mono text-[11px] font-medium text-sky-600 dark:text-sky-400"
              >
                {t('tluclaw.preview.major')}: {maj}
              </span>
            ))}
          </div>

          {/* Superseded Documents */}
          {card.blast_radius.superseded_doc_names && card.blast_radius.superseded_doc_names.length > 0 && (
            <div className="mt-3 rounded border border-blue-500/20 bg-blue-500/5 p-2.5">
              <span className="text-xs font-semibold text-blue-600 dark:text-blue-400">
                {t('tluclaw.preview.superseded_docs_warning')}:
              </span>
              <ul className="mt-1 list-disc list-inside space-y-0.5 text-xs text-muted-foreground">
                {card.blast_radius.superseded_doc_names.map((doc, idx) => (
                  <li key={idx} className="font-medium text-foreground/90">
                    {doc}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      {/* Diff Table */}
      {card.diff_items && card.diff_items.length > 0 && (
        <div className="mt-4 overflow-hidden rounded-lg border border-border">
          <div className="max-h-64 overflow-y-auto overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="sticky top-0 bg-muted/80 backdrop-blur-sm text-muted-foreground">
                <tr className="border-b border-border">
                  <th className="py-2 px-3 w-8">
                    <input
                      type="checkbox"
                      checked={selectedKeys.size > 0 && selectedKeys.size === (card.diff_items || []).length}
                      onChange={toggleSelectAll}
                      className="rounded border-border text-primary focus:ring-primary cursor-pointer"
                    />
                  </th>
                  <th className="py-2 px-3 font-semibold">{t('tluclaw.diff.col_entity')}</th>
                  <th className="py-2 px-3 font-semibold">{t('tluclaw.diff.col_action')}</th>
                  <th className="py-2 px-3 font-semibold">{t('tluclaw.diff.col_key')}</th>
                  <th className="py-2 px-3 font-semibold">{t('tluclaw.diff.col_name')}</th>
                  <th className="py-2 px-3 font-semibold">{t('tluclaw.diff.col_change')}</th>
                  <th className="py-2 px-3 font-semibold">{t('tluclaw.diff.col_notes')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/50">
                {card.diff_items.map((item, idx) => (
                  <tr key={idx} className={`hover:bg-muted/30 transition-colors ${!selectedKeys.has(item.key) ? 'opacity-40' : ''}`}>
                    <td className="py-2 px-3 w-8">
                      <input
                        type="checkbox"
                        checked={selectedKeys.has(item.key)}
                        onChange={() => toggleKey(item.key)}
                        className="rounded border-border text-primary focus:ring-primary cursor-pointer"
                      />
                    </td>
                    <td className="py-2 px-3 font-mono font-medium text-muted-foreground">
                      {item.entity_type}
                    </td>
                    <td className="py-2 px-3">
                      <span className={`inline-block rounded border px-1.5 py-0.5 text-[10px] font-semibold ${actionBadgeClass(item.action)}`}>
                        {item.action}
                      </span>
                    </td>
                    <td className="py-2 px-3 font-mono font-semibold text-foreground">
                      {item.key}
                    </td>
                    <td className="py-2 px-3 text-foreground/90 font-medium">
                      {item.name || '-'}
                    </td>
                    <td className="py-2 px-3">
                      {item.old_value || item.new_value ? (
                        <span className="flex items-center gap-1.5 font-mono text-[11px]">
                          <span className="text-muted-foreground line-through">{item.old_value || '-'}</span>
                          <ArrowRight className="h-3 w-3 text-muted-foreground" />
                          <span className="font-semibold text-foreground">{item.new_value || '-'}</span>
                        </span>
                      ) : (
                        '-'
                      )}
                    </td>
                    <td className="py-2 px-3 text-muted-foreground">
                      {item.note || '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Action Footer */}
      <div className="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-border/60 pt-3">
        {status === 'STAGED' && (
          <>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={handleConfirm}
                disabled={loadingAction !== null || selectedKeys.size === 0}
                className="inline-flex items-center gap-1.5 rounded-lg bg-emerald-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/50 disabled:opacity-50 transition-colors"
              >
                {loadingAction === 'confirm' ? (
                  <LoadingSpinner className="h-3.5 w-3.5 text-white" />
                ) : (
                  <Check className="h-3.5 w-3.5" />
                )}
                {t('tluclaw.preview.btn_confirm')} ({selectedKeys.size}/{card.diff_items.length})
              </button>

              <button
                type="button"
                onClick={handleCancel}
                disabled={loadingAction !== null}
                className="inline-flex items-center gap-1.5 rounded-lg border border-border bg-card px-3.5 py-2 text-xs font-semibold text-foreground hover:bg-muted focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50 transition-colors"
              >
                {loadingAction === 'cancel' ? (
                  <LoadingSpinner className="h-3.5 w-3.5 text-muted-foreground" />
                ) : (
                  <X className="h-3.5 w-3.5" />
                )}
                {t('tluclaw.preview.btn_cancel')}
              </button>
            </div>

            <button
              type="button"
              onClick={handleRefine}
              className="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline transition-colors"
            >
              <MessageSquare className="h-3.5 w-3.5" />
              {t('tluclaw.preview.btn_refine')}
            </button>
          </>
        )}

        {status === 'COMMITTED' && (
          <div className="flex items-center gap-2 text-xs font-semibold text-emerald-600 dark:text-emerald-400">
            <CheckCircle2 className="h-4 w-4" />
            <span>{t('tluclaw.preview.committed_msg')}</span>
          </div>
        )}

        {status === 'CANCELLED' && (
          <div className="flex items-center gap-2 text-xs font-semibold text-muted-foreground">
            <XCircle className="h-4 w-4" />
            <span>{t('tluclaw.preview.cancelled_msg')}</span>
          </div>
        )}
      </div>
    </div>
  );
};
