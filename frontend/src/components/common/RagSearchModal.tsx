import React, { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Search } from 'lucide-react';
import { Modal } from '@/components/common/Modal';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { ragSearchService, type RagSearchResultDTO } from '@/services/ragAdminService';
import { getApiErrorMessage } from '@/utils/apiError';

interface RagSearchModalProps {
  isOpen: boolean;
  onClose: () => void;
  onOpenDocument: (documentId: string) => void;
}

const inputClass =
  'w-full px-3 py-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-sm text-slate-800 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500/30';

// RagSearchModal lets an administrator test what the advisor would retrieve for a query and cohort.
export const RagSearchModal: React.FC<RagSearchModalProps> = ({ isOpen, onClose, onOpenDocument }) => {
  const { t } = useTranslation();
  const [query, setQuery] = useState('');
  const [cohort, setCohort] = useState('');
  const [topK, setTopK] = useState(5);
  const [results, setResults] = useState<RagSearchResultDTO[] | null>(null);

  const search = useMutation({
    mutationFn: () =>
      ragSearchService.search({
        query: query.trim(),
        student_cohort: cohort.trim() || undefined,
        top_k: topK,
      }),
    onSuccess: (items) => setResults(items),
  });

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={t('documents.rag_search_title')} maxWidth="4xl">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (query.trim()) search.mutate();
        }}
        className="grid grid-cols-1 sm:grid-cols-[2fr_1fr_6rem_auto] gap-2 mb-4"
      >
        <input
          className={inputClass}
          value={query}
          maxLength={4000}
          placeholder={t('documents.rag_search_placeholder')}
          onChange={(e) => setQuery(e.target.value)}
        />
        <input
          className={inputClass}
          value={cohort}
          maxLength={50}
          placeholder={t('documents.rag_search_cohort')}
          onChange={(e) => setCohort(e.target.value)}
        />
        <input
          type="number"
          min={1}
          max={50}
          className={inputClass}
          value={topK}
          onChange={(e) => setTopK(Math.min(50, Math.max(1, Number(e.target.value) || 1)))}
        />
        <button
          type="submit"
          disabled={search.isPending || !query.trim()}
          className="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-xl bg-tlu-blue text-white text-xs font-semibold disabled:opacity-50"
        >
          {search.isPending ? <LoadingSpinner size="sm" /> : <Search className="w-4 h-4" />}
          <span>{t('documents.rag_search_action')}</span>
        </button>
      </form>

      {search.isError && (
        <div className="text-xs text-rose-600 mb-3">
          {getApiErrorMessage(search.error, t('documents.rag_search_failed'))}
        </div>
      )}

      {results !== null && results.length === 0 && (
        <div className="text-xs text-slate-400 italic">{t('documents.rag_search_empty')}</div>
      )}

      <ul className="space-y-3">
        {(results ?? []).map((r) => (
          <li
            key={r.chunk_id}
            className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/80 dark:border-slate-800"
          >
            <div className="flex flex-wrap items-center justify-between gap-2 mb-1">
              <button
                type="button"
                onClick={() => onOpenDocument(r.document_id)}
                className="text-xs font-semibold text-tlu-blue hover:underline text-left"
              >
                {r.doc_title}
              </button>
              <span className="text-[10px] text-slate-500">
                {t('documents.rag_search_score', {
                  final: r.final_score.toFixed(3),
                  vector: r.vector_sim.toFixed(3),
                  bm25: r.bm25_score.toFixed(3),
                })}
              </span>
            </div>
            <div className="text-[10px] text-slate-400 mb-1">
              {r.doc_type} · {r.domain} · {r.applicable_cohort || 'ALL'}
            </div>
            <p className="text-xs text-slate-700 dark:text-slate-300 whitespace-pre-wrap line-clamp-6">{r.content}</p>
          </li>
        ))}
      </ul>
    </Modal>
  );
};
