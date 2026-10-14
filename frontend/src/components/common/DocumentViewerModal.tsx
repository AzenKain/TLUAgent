import React, { useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';
import {
  BookOpen,
  Search,
  Copy,
  Check,
  AlertTriangle,
  Layers,
  Info,
  ArrowLeft,
  X,
  History,
} from 'lucide-react';
import { documentService } from '@/services/documentService';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { MarkdownRenderer } from '@/components/common/MarkdownRenderer';
import type { DocumentDetailDTO } from '@/services/ragAdminService';

interface DocumentViewerModalProps {
  isOpen: boolean;
  onClose: () => void;
  documentIdentifier?: string | null;
  initialDocDetail?: DocumentDetailDTO | null;
}

export const DocumentViewerModal: React.FC<DocumentViewerModalProps> = ({
  isOpen,
  onClose,
  documentIdentifier,
  initialDocDetail,
}) => {
  const { t } = useTranslation();
  const [navigatedId, setNavigatedId] = useState<string | null>(null);
  const [prevDocProp, setPrevDocProp] = useState<string | null | undefined>(documentIdentifier);
  const [historyStack, setHistoryStack] = useState<string[]>([]);
  const [activeTab, setActiveTab] = useState<'content' | 'metadata' | 'chunks' | 'audit'>('content');
  const [searchInDoc, setSearchInDoc] = useState('');
  const [isCopied, setIsCopied] = useState(false);

  if (documentIdentifier !== prevDocProp) {
    setPrevDocProp(documentIdentifier);
    setNavigatedId(null);
    setHistoryStack([]);
    setSearchInDoc('');
    setActiveTab('content');
  }

  const targetIdentifier = navigatedId || documentIdentifier;

  const {
    data: fetchedDetail,
    isLoading,
    isError,
    refetch,
  } = useQuery({
    queryKey: ['document-viewer-detail', targetIdentifier],
    queryFn: () => (targetIdentifier ? documentService.getDocumentDetail(targetIdentifier) : null),
    enabled: isOpen && !!targetIdentifier && (!initialDocDetail || !!navigatedId),
    staleTime: 60 * 1000,
  });

  const docDetail = (navigatedId ? fetchedDetail : initialDocDetail) || fetchedDetail;

  const handleNavigateToDoc = (newDocId: string) => {
    if (targetIdentifier) {
      setHistoryStack((prev) => [...prev, targetIdentifier]);
    }
    setNavigatedId(newDocId);
    setSearchInDoc('');
  };

  const handleGoBack = () => {
    if (historyStack.length === 0) return;
    const prev = historyStack[historyStack.length - 1];
    setHistoryStack((prevStack) => prevStack.slice(0, -1));
    setNavigatedId(prev);
    setSearchInDoc('');
  };

  const fullContent = useMemo(() => {
    if (!docDetail) return '';
    if (docDetail.full_content && docDetail.full_content.trim().length > 0) {
      return docDetail.full_content;
    }
    if (docDetail.chunks && docDetail.chunks.length > 0) {
      return docDetail.chunks.map((c) => c.content).join('\n\n');
    }
    return '';
  }, [docDetail]);

  const searchMatchCount = useMemo(() => {
    if (!searchInDoc.trim() || !fullContent) return 0;
    try {
      const q = searchInDoc.trim().replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
      const matches = fullContent.match(new RegExp(q, 'gi'));
      return matches ? matches.length : 0;
    } catch {
      return 0;
    }
  }, [fullContent, searchInDoc]);

  const handleCopyContent = async () => {
    if (!fullContent) return;
    try {
      await navigator.clipboard.writeText(fullContent);
      setIsCopied(true);
      setTimeout(() => setIsCopied(false), 2000);
    } catch {}
  };

  if (!isOpen) return null;

  const doc = docDetail?.document;

  const renderStatusBadge = (status?: string) => {
    if (status === 'ACTIVE') {
      return (
        <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800">
          <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
          {t('documents.status_active')}
        </span>
      );
    }
    if (status === 'SUPERSEDED') {
      return (
        <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-amber-50 text-amber-700 dark:bg-amber-950/60 dark:text-amber-300 border border-amber-200 dark:border-amber-800">
          <AlertTriangle className="h-3 w-3" />
          {t('documents.status_superseded')}
        </span>
      );
    }
    return (
      <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300 border border-slate-200 dark:border-slate-700">
        {status || t('documents.status_archived')}
      </span>
    );
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-2 sm:p-4 animate-in fade-in duration-150">
      <div
        role="dialog"
        aria-modal="true"
        className="flex max-h-[92dvh] w-full max-w-5xl flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl dark:border-slate-800 dark:bg-[#111A35]"
      >
        {/* Modal Header */}
        <div className="flex items-start justify-between border-b border-slate-200 p-4 dark:border-slate-800/80 bg-slate-50/60 dark:bg-slate-900/40">
          <div className="flex items-start gap-3 min-w-0 pr-2">
            <div className="rounded-xl bg-blue-50 p-2 text-tlu-blue dark:bg-blue-950/60 dark:text-tlu-cyan shrink-0 mt-0.5">
              <BookOpen className="h-5 w-5" />
            </div>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                {historyStack.length > 0 && (
                  <button
                    onClick={handleGoBack}
                    className="inline-flex items-center gap-1 text-xs text-tlu-blue hover:underline font-medium cursor-pointer"
                  >
                    <ArrowLeft className="w-3.5 h-3.5" />
                    <span>{t('common.back')}</span>
                  </button>
                )}
                {doc && renderStatusBadge(doc.status)}
                {doc?.domain && (
                  <span className="rounded-md bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-300">
                    {doc.domain}
                  </span>
                )}
                {doc?.applicable_cohort && (
                  <span className="rounded-md bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-800 dark:text-slate-400">
                    {t('documents.col_cohort')}: {doc.applicable_cohort}
                  </span>
                )}
              </div>
              <h2 className="mt-1 text-base sm:text-lg font-bold text-slate-900 dark:text-slate-100 line-clamp-2 leading-snug">
                {doc?.title || targetIdentifier || t('documents.inspect_title')}
              </h2>
              {doc && (
                <div className="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
                  <span className="font-mono font-medium">{doc.doc_code || doc.id}</span>
                  <span>•</span>
                  <span>{doc.type}</span>
                  {doc.publish_date && (
                    <>
                      <span>•</span>
                      <span>{doc.publish_date}</span>
                    </>
                  )}
                </div>
              )}
            </div>
          </div>

          <button
            onClick={onClose}
            className="rounded-lg p-1.5 text-slate-400 hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-slate-800 dark:hover:text-slate-200 cursor-pointer shrink-0"
            aria-label={t('common.close')}
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Superseded Warning Banner if applicable */}
        {docDetail?.superseded_by && (
          <div className="border-b border-amber-200 bg-amber-50/90 px-4 py-2.5 dark:border-amber-900/50 dark:bg-amber-950/40 flex items-center justify-between gap-3 text-xs">
            <div className="flex items-center gap-2 text-amber-800 dark:text-amber-200">
              <AlertTriangle className="h-4 w-4 shrink-0 text-amber-600 dark:text-amber-400" />
              <span>
                {t('documents.superseded_by', { doc: docDetail.superseded_by.title })}
              </span>
            </div>
            <button
              onClick={() => handleNavigateToDoc(docDetail.superseded_by!.id)}
              className="font-semibold text-tlu-blue dark:text-blue-300 hover:underline shrink-0 cursor-pointer"
            >
              {t('documents.view_replacement_doc')}
            </button>
          </div>
        )}

        {/* Tab Switcher */}
        <div className="flex items-center justify-between border-b border-slate-200 px-4 dark:border-slate-800 bg-white dark:bg-[#111A35]">
          <div className="flex space-x-1 sm:space-x-2">
            <button
              onClick={() => setActiveTab('content')}
              className={`border-b-2 py-2.5 px-3 text-xs sm:text-sm font-medium transition cursor-pointer ${
                activeTab === 'content'
                  ? 'border-tlu-blue text-tlu-blue dark:border-tlu-cyan dark:text-tlu-cyan'
                  : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
              }`}
            >
              {t('documents.tab_full_content')}
            </button>
            <button
              onClick={() => setActiveTab('metadata')}
              className={`border-b-2 py-2.5 px-3 text-xs sm:text-sm font-medium transition cursor-pointer ${
                activeTab === 'metadata'
                  ? 'border-tlu-blue text-tlu-blue dark:border-tlu-cyan dark:text-tlu-cyan'
                  : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
              }`}
            >
              {t('documents.tab_metadata')}
            </button>
            <button
              onClick={() => setActiveTab('chunks')}
              className={`border-b-2 py-2.5 px-3 text-xs sm:text-sm font-medium transition cursor-pointer ${
                activeTab === 'chunks'
                  ? 'border-tlu-blue text-tlu-blue dark:border-tlu-cyan dark:text-tlu-cyan'
                  : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
              }`}
            >
              {t('documents.tab_chunks', { count: docDetail?.chunks?.length || 0 })}
            </button>
            {docDetail?.audit_logs && docDetail.audit_logs.length > 0 && (
              <button
                onClick={() => setActiveTab('audit')}
                className={`border-b-2 py-2.5 px-3 text-xs sm:text-sm font-medium transition cursor-pointer ${
                  activeTab === 'audit'
                    ? 'border-tlu-blue text-tlu-blue dark:border-tlu-cyan dark:text-tlu-cyan'
                    : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
                }`}
              >
                {t('documents.tab_audit')}
              </button>
            )}
          </div>

          {activeTab === 'content' && fullContent && (
            <button
              onClick={handleCopyContent}
              className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-2.5 py-1 text-xs font-medium text-slate-600 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800 cursor-pointer transition-colors"
              title={t('common.copy')}
            >
              {isCopied ? (
                <>
                  <Check className="h-3.5 w-3.5 text-emerald-500" />
                  <span className="text-emerald-600 dark:text-emerald-400">{t('common.copied')}</span>
                </>
              ) : (
                <>
                  <Copy className="h-3.5 w-3.5" />
                  <span>{t('common.copy')}</span>
                </>
              )}
            </button>
          )}
        </div>

        {/* Modal Body */}
        <div className="flex-1 overflow-y-auto p-4 sm:p-6">
          {isLoading ? (
            <div className="flex h-64 flex-col items-center justify-center gap-3">
              <LoadingSpinner size="lg" />
              <p className="text-xs text-slate-400">{t('common.loading')}</p>
            </div>
          ) : isError || !docDetail ? (
            <div className="flex h-64 flex-col items-center justify-center gap-3 text-slate-400 text-center">
              <AlertTriangle className="h-10 w-10 text-amber-500" />
              <div>
                <p className="text-sm font-semibold text-slate-800 dark:text-slate-200">
                  {t('documents.not_found_title')}
                </p>
                <p className="text-xs mt-1 text-slate-500">
                  {t('documents.not_found_desc', { identifier: targetIdentifier })}
                </p>
              </div>
              <button
                onClick={() => refetch()}
                className="mt-2 rounded-lg bg-tlu-blue px-3 py-1.5 text-xs font-medium text-white hover:bg-tlu-navy cursor-pointer"
              >
                {t('common.retry')}
              </button>
            </div>
          ) : (
            <>
              {/* Tab 1: Full Document Content */}
              {activeTab === 'content' && (
                <div className="space-y-4">
                  {/* Search bar inside document */}
                  <div className="relative">
                    <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                    <input
                      type="text"
                      value={searchInDoc}
                      onChange={(e) => setSearchInDoc(e.target.value)}
                      placeholder={t('documents.search_inside_placeholder')}
                      className="w-full rounded-xl border border-slate-200 bg-slate-50/70 py-2 pl-9 pr-24 text-xs sm:text-sm text-slate-800 placeholder-slate-400 focus:border-tlu-blue focus:bg-white focus:outline-hidden focus:ring-1 focus:ring-tlu-blue dark:border-slate-700 dark:bg-slate-800/70 dark:text-slate-100 dark:placeholder-slate-500"
                    />
                    {searchInDoc.trim() && (
                      <span className="absolute right-3 top-1/2 -translate-y-1/2 text-[11px] font-semibold text-slate-400 bg-slate-100 dark:bg-slate-700 px-1.5 py-0.5 rounded">
                        {t('documents.matches_count', { count: searchMatchCount })}
                      </span>
                    )}
                  </div>

                  {fullContent ? (
                    <div className="rounded-xl border border-slate-100 dark:border-slate-800/80 bg-white dark:bg-slate-900/40 p-4 sm:p-5 shadow-2xs">
                      <MarkdownRenderer content={fullContent} searchQuery={searchInDoc} />
                    </div>
                  ) : (
                    <div className="flex h-40 flex-col items-center justify-center gap-2 text-slate-400">
                      <Info className="h-8 w-8 text-slate-300 dark:text-slate-600" />
                      <p className="text-xs">{t('common.no_data')}</p>
                    </div>
                  )}
                </div>
              )}

              {/* Tab 2: Metadata */}
              {activeTab === 'metadata' && doc && (
                <div className="space-y-4">
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 text-xs sm:text-sm">
                    <div className="rounded-xl border border-slate-100 bg-slate-50/70 p-3.5 dark:border-slate-800 dark:bg-slate-800/40">
                      <span className="text-xs text-slate-400 font-medium">{t('documents.col_code')}</span>
                      <p className="mt-1 font-mono font-bold text-slate-900 dark:text-slate-100">
                        {doc.doc_code || doc.id}
                      </p>
                    </div>

                    <div className="rounded-xl border border-slate-100 bg-slate-50/70 p-3.5 dark:border-slate-800 dark:bg-slate-800/40">
                      <span className="text-xs text-slate-400 font-medium">{t('documents.col_status')}</span>
                      <div className="mt-1">{renderStatusBadge(doc.status)}</div>
                    </div>

                    <div className="rounded-xl border border-slate-100 bg-slate-50/70 p-3.5 dark:border-slate-800 dark:bg-slate-800/40">
                      <span className="text-xs text-slate-400 font-medium">{t('documents.col_publish_date')}</span>
                      <p className="mt-1 font-semibold text-slate-800 dark:text-slate-200">
                        {doc.publish_date || '-'}
                      </p>
                    </div>

                    <div className="rounded-xl border border-slate-100 bg-slate-50/70 p-3.5 dark:border-slate-800 dark:bg-slate-800/40">
                      <span className="text-xs text-slate-400 font-medium">{t('documents.col_cohort')}</span>
                      <p className="mt-1 font-semibold text-slate-800 dark:text-slate-200">
                        {doc.applicable_cohort || 'ALL'}
                      </p>
                    </div>

                    <div className="rounded-xl border border-slate-100 bg-slate-50/70 p-3.5 dark:border-slate-800 dark:bg-slate-800/40">
                      <span className="text-xs text-slate-400 font-medium">{t('documents.col_domain')}</span>
                      <p className="mt-1 font-semibold text-slate-800 dark:text-slate-200">
                        {doc.domain}
                      </p>
                    </div>

                    <div className="rounded-xl border border-slate-100 bg-slate-50/70 p-3.5 dark:border-slate-800 dark:bg-slate-800/40">
                      <span className="text-xs text-slate-400 font-medium">{t('documents.col_priority')}</span>
                      <p className="mt-1 font-semibold text-slate-800 dark:text-slate-200">
                        {doc.priority_level}
                      </p>
                    </div>
                  </div>

                  {/* Superseded links */}
                  {docDetail.superseded_by && (
                    <div className="rounded-xl border border-amber-200 bg-amber-50/70 p-4 dark:border-amber-900/50 dark:bg-amber-950/30">
                      <span className="text-xs font-bold text-amber-800 dark:text-amber-200 flex items-center gap-1.5">
                        <AlertTriangle className="w-4 h-4 text-amber-600 dark:text-amber-400" />
                        {t('documents.superseded_by_title')}
                      </span>
                      <p className="mt-1.5 text-xs text-amber-700 dark:text-amber-300">
                        {docDetail.superseded_by.title} ({docDetail.superseded_by.doc_code || docDetail.superseded_by.id})
                      </p>
                      <button
                        onClick={() => handleNavigateToDoc(docDetail.superseded_by!.id)}
                        className="mt-2 text-xs font-semibold text-tlu-blue dark:text-blue-300 hover:underline cursor-pointer"
                      >
                        {t('documents.view_replacement_doc')}
                      </button>
                    </div>
                  )}

                  {docDetail.supersedes && docDetail.supersedes.length > 0 && (
                    <div className="rounded-xl border border-blue-200 bg-blue-50/70 p-4 dark:border-blue-900/50 dark:bg-blue-950/30">
                      <span className="text-xs font-bold text-tlu-navy dark:text-blue-300 flex items-center gap-1.5">
                        <History className="w-4 h-4 text-tlu-blue" />
                        {t('documents.supersedes_count', { count: docDetail.supersedes.length })}
                      </span>
                      <ul className="mt-2 space-y-1.5 text-xs text-blue-900 dark:text-blue-200">
                        {docDetail.supersedes.map((prev) => (
                          <li key={prev.id} className="flex items-center justify-between">
                            <span>
                              • {prev.title} ({prev.doc_code || prev.id})
                            </span>
                            <button
                              onClick={() => handleNavigateToDoc(prev.id)}
                              className="font-medium text-tlu-blue dark:text-blue-300 hover:underline cursor-pointer ml-2"
                            >
                              {t('common.view')}
                            </button>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              )}

              {/* Tab 3: Chunks */}
              {activeTab === 'chunks' && (
                <div className="space-y-3">
                  {!docDetail.chunks || docDetail.chunks.length === 0 ? (
                    <p className="text-xs text-slate-400 text-center py-8">{t('common.no_data')}</p>
                  ) : (
                    docDetail.chunks.map((chunk) => (
                      <div
                        key={chunk.id}
                        className="rounded-xl border border-slate-200 p-3.5 dark:border-slate-800 dark:bg-slate-800/40 shadow-2xs"
                      >
                        <div className="flex items-center justify-between text-xs text-slate-500 pb-2 border-b border-slate-100 dark:border-slate-800">
                          <span className="font-semibold text-slate-700 dark:text-slate-300 flex items-center gap-1">
                            <Layers className="w-3.5 h-3.5 text-tlu-blue" />
                            {t('documents.chunk_index', { index: chunk.chunk_index })}
                          </span>
                          <span className="text-[11px] font-mono text-slate-400">
                            {t('documents.chunk_tokens', { chars: chunk.content.length })}
                          </span>
                        </div>
                        <p className="mt-2.5 text-xs leading-relaxed text-slate-700 dark:text-slate-300 whitespace-pre-wrap font-sans">
                          {chunk.content}
                        </p>
                      </div>
                    ))
                  )}
                </div>
              )}

              {/* Tab 4: Audit Logs */}
              {activeTab === 'audit' && (
                <div className="space-y-3">
                  {!docDetail.audit_logs || docDetail.audit_logs.length === 0 ? (
                    <p className="text-xs text-slate-400 text-center py-8">{t('common.no_data')}</p>
                  ) : (
                    docDetail.audit_logs.map((log) => (
                      <div
                        key={log.id}
                        className="rounded-xl border border-slate-200 p-3.5 text-xs dark:border-slate-800 dark:bg-slate-800/40"
                      >
                        <div className="flex items-center justify-between font-semibold text-slate-800 dark:text-slate-200">
                          <span>{log.rule_type}</span>
                          <span className="text-[11px] text-slate-400 font-normal">{log.created_at}</span>
                        </div>
                        <p className="mt-1 text-slate-600 dark:text-slate-400">{log.reason}</p>
                      </div>
                    ))
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
};
