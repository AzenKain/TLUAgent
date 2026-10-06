import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Search,
  BookOpen,
  RefreshCw,
  Eye,
  CheckCircle2,
  AlertTriangle,
  Layers,
  ArrowRight,
} from 'lucide-react';
import {
  ragAdminService,
  type DocumentDTO,
} from '@/services/ragAdminService';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { Badge } from '@/components/common/Badge';
import { Modal } from '@/components/common/Modal';

// DocumentManagePage provides administrative oversight of institutional regulations, chunks, and supersession links.
export const DocumentManagePage: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [page, setPage] = useState(1);
  const [pageSize] = useState(15);
  const [search, setSearch] = useState('');
  const [domainFilter, setDomainFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [cohortFilter, setCohortFilter] = useState('');

  const [inspectModalOpen, setInspectModalOpen] = useState(false);
  const [selectedDocId, setSelectedDocId] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'metadata' | 'chunks' | 'audit'>('metadata');

  const [editStatusModalOpen, setEditStatusModalOpen] = useState(false);
  const [editingDoc, setEditingDoc] = useState<DocumentDTO | null>(null);
  const [newStatus, setNewStatus] = useState('ACTIVE');
  const [supersededById, setSupersededById] = useState('');
  const [statusReason, setStatusReason] = useState('');

  const [toastMessage, setToastMessage] = useState<string | null>(null);
  const [toastType, setToastType] = useState<'success' | 'error'>('success');

  const showToast = (message: string, type: 'success' | 'error' = 'success') => {
    setToastMessage(message);
    setToastType(type);
    setTimeout(() => setToastMessage(null), 4000);
  };

  const { data: stats } = useQuery({
    queryKey: ['admin-documents', 'stats'],
    queryFn: () => ragAdminService.getStats(),
  });

  const {
    data: docResult,
    isLoading,
    isFetching,
    refetch,
  } = useQuery({
    queryKey: [
      'admin-documents',
      'list',
      { search, domainFilter, statusFilter, cohortFilter, page, pageSize },
    ],
    queryFn: () =>
      ragAdminService.listDocuments({
        search: search.trim() || undefined,
        domain: domainFilter || undefined,
        status: statusFilter || undefined,
        cohort: cohortFilter || undefined,
        page,
        limit: pageSize,
      }),
  });

  const { data: docDetail, isLoading: isDetailLoading } = useQuery({
    queryKey: ['admin-documents', 'detail', selectedDocId],
    queryFn: () => (selectedDocId ? ragAdminService.getDocumentDetail(selectedDocId) : null),
    enabled: !!selectedDocId && inspectModalOpen,
  });

  const syncMutation = useMutation({
    mutationFn: () => ragAdminService.syncIndex(),
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ['admin-documents'] });
      showToast(`${t('documents.sync_success')} (${res.data.ingested_documents})`, 'success');
    },
    onError: () => {
      showToast(t('documents.sync_error'), 'error');
    },
  });

  const updateStatusMutation = useMutation({
    mutationFn: ({
      id,
      status,
      targetSupersededById,
      reason,
    }: {
      id: string;
      status: string;
      targetSupersededById?: string;
      reason?: string;
    }) =>
      ragAdminService.updateDocumentStatus(id, {
        status,
        superseded_by_id: targetSupersededById || undefined,
        reason: reason || undefined,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-documents'] });
      setEditStatusModalOpen(false);
      setEditingDoc(null);
      showToast(t('documents.update_success'), 'success');
    },
    onError: (err: any) => {
      showToast(err?.response?.data?.message || t('common.error_occurred'), 'error');
    },
  });

  const handleOpenEditStatus = (doc: DocumentDTO) => {
    setEditingDoc(doc);
    setNewStatus(doc.status);
    setSupersededById(doc.superseded_by_id || '');
    setStatusReason('');
    setEditStatusModalOpen(true);
  };

  const handleSaveStatus = (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingDoc) return;
    updateStatusMutation.mutate({
      id: editingDoc.id,
      status: newStatus,
      targetSupersededById: supersededById.trim() || undefined,
      reason: statusReason.trim() || undefined,
    });
  };

  const totalPages = docResult ? Math.ceil(docResult.total / pageSize) : 1;

  const renderStatusBadge = (status: string) => {
    switch (status?.toUpperCase()) {
      case 'ACTIVE':
        return <Badge variant="success">{t('documents.status_active')}</Badge>;
      case 'SUPERSEDED':
        return <Badge variant="warning">{t('documents.status_superseded')}</Badge>;
      case 'DRAFT':
        return <Badge variant="info">{t('documents.status_draft')}</Badge>;
      default:
        return <Badge variant="secondary">{status}</Badge>;
    }
  };

  return (
    <div className="space-y-6">
      {toastMessage && (
        <div
          className={`fixed bottom-5 right-5 z-50 flex items-center gap-2 rounded-xl px-4 py-3 text-sm font-medium text-white shadow-xl transition-all ${
            toastType === 'success' ? 'bg-emerald-600' : 'bg-rose-600'
          }`}
        >
          {toastType === 'success' ? (
            <CheckCircle2 className="h-5 w-5" />
          ) : (
            <AlertTriangle className="h-5 w-5" />
          )}
          <span>{toastMessage}</span>
        </div>
      )}

      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            {t('documents.title')}
          </h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            {t('documents.subtitle')}
          </p>
        </div>
        <button
          onClick={() => syncMutation.mutate()}
          disabled={syncMutation.isPending}
          className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white shadow-xs hover:bg-indigo-700 disabled:opacity-50 transition"
        >
          <RefreshCw className={`h-4 w-4 ${syncMutation.isPending ? 'animate-spin' : ''}`} />
          {syncMutation.isPending ? t('documents.btn_syncing') : t('documents.btn_sync')}
        </button>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-xs dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-indigo-50 p-2.5 text-indigo-600 dark:bg-indigo-950/50 dark:text-indigo-400">
              <BookOpen className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 dark:text-slate-400">
                {t('documents.stat_total_docs')}
              </p>
              <p className="text-xl font-bold text-slate-900 dark:text-slate-100">
                {stats?.total_documents ?? 0}
              </p>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-xs dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-emerald-50 p-2.5 text-emerald-600 dark:bg-emerald-950/50 dark:text-emerald-400">
              <CheckCircle2 className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 dark:text-slate-400">
                {t('documents.stat_active_docs')}
              </p>
              <p className="text-xl font-bold text-slate-900 dark:text-slate-100">
                {stats?.active_documents ?? 0}
              </p>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-xs dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-amber-50 p-2.5 text-amber-600 dark:bg-amber-950/50 dark:text-amber-400">
              <AlertTriangle className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 dark:text-slate-400">
                {t('documents.stat_superseded_docs')}
              </p>
              <p className="text-xl font-bold text-slate-900 dark:text-slate-100">
                {stats?.superseded_documents ?? 0}
              </p>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-xs dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-sky-50 p-2.5 text-sky-600 dark:bg-sky-950/50 dark:text-sky-400">
              <Layers className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 dark:text-slate-400">
                {t('documents.stat_total_chunks')}
              </p>
              <p className="text-xl font-bold text-slate-900 dark:text-slate-100">
                {stats?.total_chunks ?? 0}
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-xs dark:border-slate-800 dark:bg-slate-900">
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5">
          <div className="relative lg:col-span-2">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(1);
              }}
              placeholder={t('documents.search_placeholder')}
              className="w-full rounded-lg border border-slate-200 bg-white py-2 pl-9 pr-3 text-sm text-slate-800 placeholder-slate-400 focus:border-indigo-500 focus:outline-hidden focus:ring-1 focus:ring-indigo-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100 dark:placeholder-slate-500"
            />
          </div>

          <div>
            <select
              value={domainFilter}
              onChange={(e) => {
                setDomainFilter(e.target.value);
                setPage(1);
              }}
              className="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300"
            >
              <option value="">{t('documents.domain_all')}</option>
              <option value="DAO_TAO">{t('documents.domain_dao_tao')}</option>
              <option value="CONG_TAC_SINH_VIEN">{t('documents.domain_ctsv')}</option>
              <option value="KHAO_THI">{t('documents.domain_khao_thi')}</option>
              <option value="HOC_PHI">{t('documents.domain_hoc_phi')}</option>
            </select>
          </div>

          <div>
            <select
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setPage(1);
              }}
              className="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300"
            >
              <option value="">{t('documents.status_all')}</option>
              <option value="ACTIVE">{t('documents.status_active')}</option>
              <option value="SUPERSEDED">{t('documents.status_superseded')}</option>
              <option value="DRAFT">{t('documents.status_draft')}</option>
              <option value="ARCHIVED">{t('documents.status_archived')}</option>
            </select>
          </div>

          <div className="flex items-center gap-2">
            <input
              type="text"
              value={cohortFilter}
              onChange={(e) => {
                setCohortFilter(e.target.value);
                setPage(1);
              }}
              placeholder={t('documents.cohort_all')}
              className="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-800 placeholder-slate-400 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
            <button
              onClick={() => refetch()}
              className="inline-flex items-center rounded-lg border border-slate-200 p-2 text-slate-600 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
            >
              <RefreshCw className={`h-4 w-4 ${isFetching ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>
      </div>

      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xs dark:border-slate-800 dark:bg-slate-900">
        {isLoading ? (
          <div className="flex h-64 items-center justify-center">
            <LoadingSpinner size="lg" />
          </div>
        ) : !docResult?.documents || docResult.documents.length === 0 ? (
          <div className="flex h-64 flex-col items-center justify-center gap-2 text-slate-400">
            <BookOpen className="h-10 w-10 text-slate-300 dark:text-slate-600" />
            <p className="text-sm">{t('documents.empty_documents')}</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-slate-600 dark:text-slate-300">
              <thead className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase text-slate-500 dark:border-slate-800 dark:bg-slate-800/50 dark:text-slate-400">
                <tr>
                  <th className="px-4 py-3.5">{t('documents.col_code')}</th>
                  <th className="px-4 py-3.5">{t('documents.col_title')}</th>
                  <th className="px-4 py-3.5">{t('documents.col_domain')}</th>
                  <th className="px-4 py-3.5">{t('documents.col_cohort')}</th>
                  <th className="px-4 py-3.5">{t('documents.col_priority')}</th>
                  <th className="px-4 py-3.5">{t('documents.col_status')}</th>
                  <th className="px-4 py-3.5">{t('documents.col_superseded_info')}</th>
                  <th className="px-4 py-3.5 text-right">{t('common.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                {docResult.documents.map((doc) => (
                  <tr key={doc.id} className="hover:bg-slate-50/70 dark:hover:bg-slate-800/40">
                    <td className="whitespace-nowrap px-4 py-3 font-mono text-xs font-semibold text-slate-700 dark:text-slate-300">
                      {doc.doc_code || doc.id}
                    </td>
                    <td className="px-4 py-3">
                      <div className="max-w-xs font-medium text-slate-900 dark:text-slate-100 truncate">
                        {doc.title}
                      </div>
                      <div className="text-[11px] text-slate-400">
                        {doc.type} • {doc.publish_date}
                      </div>
                    </td>
                    <td className="whitespace-nowrap px-4 py-3">
                      <span className="rounded-md bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-300">
                        {doc.domain}
                      </span>
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-xs">
                      {doc.applicable_cohort}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-xs font-medium text-slate-700 dark:text-slate-300">
                      {doc.priority_level}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3">
                      {renderStatusBadge(doc.status)}
                    </td>
                    <td className="px-4 py-3 text-xs">
                      {doc.superseded_by_id ? (
                        <div className="flex items-center gap-1.5 text-amber-600 dark:text-amber-400">
                          <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
                          <span className="truncate max-w-[150px]">
                            {t('documents.superseded_by', { doc: doc.superseded_by_id })}
                          </span>
                        </div>
                      ) : (
                        <span className="text-slate-400">-</span>
                      )}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-1.5">
                        <button
                          onClick={() => {
                            setSelectedDocId(doc.id);
                            setActiveTab('metadata');
                            setInspectModalOpen(true);
                          }}
                          className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-indigo-600 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-indigo-400"
                          title={t('documents.inspect_title')}
                        >
                          <Eye className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleOpenEditStatus(doc)}
                          className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-amber-600 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-amber-400"
                          title={t('documents.edit_status_title')}
                        >
                          <RefreshCw className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {docResult && totalPages > 1 && (
          <div className="flex items-center justify-between border-t border-slate-200 px-4 py-3 dark:border-slate-800 sm:px-6">
            <div className="text-xs text-slate-500 dark:text-slate-400">
              {t('common.page', {
                current: page,
                total: totalPages,
                count: docResult.total,
              })}
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="rounded-lg border border-slate-200 px-3 py-1 text-xs font-medium text-slate-600 hover:bg-slate-50 disabled:opacity-40 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
              >
                {t('common.previous')}
              </button>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="rounded-lg border border-slate-200 px-3 py-1 text-xs font-medium text-slate-600 hover:bg-slate-50 disabled:opacity-40 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
              >
                {t('common.next')}
              </button>
            </div>
          </div>
        )}
      </div>

      <Modal
        isOpen={inspectModalOpen}
        onClose={() => setInspectModalOpen(false)}
        title={t('documents.inspect_title')}
        maxWidth="xl"
      >
        {isDetailLoading || !docDetail ? (
          <div className="flex h-64 items-center justify-center">
            <LoadingSpinner size="lg" />
          </div>
        ) : (
          <div className="space-y-4">
            <div className="flex border-b border-slate-200 dark:border-slate-800">
              <button
                onClick={() => setActiveTab('metadata')}
                className={`border-b-2 px-4 py-2 text-sm font-medium transition ${
                  activeTab === 'metadata'
                    ? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
                    : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400'
                }`}
              >
                {t('documents.tab_metadata')}
              </button>
              <button
                onClick={() => setActiveTab('chunks')}
                className={`border-b-2 px-4 py-2 text-sm font-medium transition ${
                  activeTab === 'chunks'
                    ? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
                    : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400'
                }`}
              >
                {t('documents.tab_chunks', { count: docDetail.chunks.length })}
              </button>
              <button
                onClick={() => setActiveTab('audit')}
                className={`border-b-2 px-4 py-2 text-sm font-medium transition ${
                  activeTab === 'audit'
                    ? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
                    : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400'
                }`}
              >
                {t('documents.tab_audit')}
              </button>
            </div>

            {activeTab === 'metadata' && (
              <div className="grid grid-cols-1 gap-4 text-sm sm:grid-cols-2">
                <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-800/50">
                  <span className="text-xs text-slate-400">{t('documents.col_title')}</span>
                  <p className="font-semibold text-slate-900 dark:text-slate-100">
                    {docDetail.document.title}
                  </p>
                </div>
                <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-800/50">
                  <span className="text-xs text-slate-400">{t('documents.col_code')}</span>
                  <p className="font-mono text-slate-800 dark:text-slate-200">
                    {docDetail.document.doc_code || docDetail.document.id}
                  </p>
                </div>
                <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-800/50">
                  <span className="text-xs text-slate-400">{t('documents.col_status')}</span>
                  <div className="mt-1">{renderStatusBadge(docDetail.document.status)}</div>
                </div>
                <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-800/50">
                  <span className="text-xs text-slate-400">{t('documents.col_cohort')}</span>
                  <p className="font-medium text-slate-800 dark:text-slate-200">
                    {docDetail.document.applicable_cohort}
                  </p>
                </div>
                <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-800/50">
                  <span className="text-xs text-slate-400">{t('documents.col_publish_date')}</span>
                  <p className="font-medium text-slate-800 dark:text-slate-200">
                    {docDetail.document.publish_date}
                  </p>
                </div>
                <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-800/50">
                  <span className="text-xs text-slate-400">{t('documents.col_priority')}</span>
                  <p className="font-medium text-slate-800 dark:text-slate-200">
                    {docDetail.document.priority_level}
                  </p>
                </div>
                {docDetail.superseded_by && (
                  <div className="col-span-full rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-900/50 dark:bg-amber-950/30">
                    <span className="text-xs font-semibold text-amber-700 dark:text-amber-300">
                      {t('documents.superseded_by', { doc: docDetail.superseded_by.title })}
                    </span>
                    <p className="text-xs text-amber-600 dark:text-amber-400">
                      {docDetail.superseded_by.doc_code} • {docDetail.superseded_by.publish_date}
                    </p>
                  </div>
                )}
                {docDetail.supersedes && docDetail.supersedes.length > 0 && (
                  <div className="col-span-full rounded-lg border border-indigo-200 bg-indigo-50 p-3 dark:border-indigo-900/50 dark:bg-indigo-950/30">
                    <span className="text-xs font-semibold text-indigo-700 dark:text-indigo-300">
                      {t('documents.supersedes_count', { count: docDetail.supersedes.length })}
                    </span>
                    <ul className="mt-1 space-y-1 text-xs text-indigo-600 dark:text-indigo-400">
                      {docDetail.supersedes.map((prev) => (
                        <li key={prev.id}>
                          • {prev.title} ({prev.doc_code || prev.id})
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            )}

            {activeTab === 'chunks' && (
              <div className="max-h-96 space-y-3 overflow-y-auto">
                {docDetail.chunks.length === 0 ? (
                  <p className="text-sm text-slate-400">{t('common.no_data')}</p>
                ) : (
                  docDetail.chunks.map((chunk) => (
                    <div
                      key={chunk.id}
                      className="rounded-lg border border-slate-200 p-3 dark:border-slate-800 dark:bg-slate-800/40"
                    >
                      <div className="flex items-center justify-between text-xs text-slate-500">
                        <span className="font-semibold text-slate-700 dark:text-slate-300">
                          {t('documents.chunk_index', { index: chunk.chunk_index })}
                        </span>
                        <span>{t('documents.chunk_tokens', { chars: chunk.content.length })}</span>
                      </div>
                      <p className="mt-2 text-xs leading-relaxed text-slate-700 dark:text-slate-300">
                        {chunk.content}
                      </p>
                    </div>
                  ))
                )}
              </div>
            )}

            {activeTab === 'audit' && (
              <div className="max-h-96 space-y-3 overflow-y-auto">
                {docDetail.audit_logs.length === 0 ? (
                  <p className="text-sm text-slate-400">{t('common.no_data')}</p>
                ) : (
                  docDetail.audit_logs.map((log) => (
                    <div
                      key={log.id}
                      className="rounded-lg border border-slate-200 p-3 text-xs dark:border-slate-800 dark:bg-slate-800/40"
                    >
                      <div className="flex items-center justify-between text-slate-400">
                        <span className="font-medium text-slate-700 dark:text-slate-300">
                          {log.rule_type}
                        </span>
                        <span>{log.created_at}</span>
                      </div>
                      <div className="mt-1 flex items-center gap-2 font-mono">
                        <span className="text-slate-600 dark:text-slate-400">{log.active_doc_id}</span>
                        <ArrowRight className="h-3 w-3 text-amber-500" />
                        <span className="text-amber-600 dark:text-amber-400">{log.superseded_doc_id}</span>
                      </div>
                      <p className="mt-1 text-slate-600 dark:text-slate-300">{log.reason}</p>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>
        )}
      </Modal>

      <Modal
        isOpen={editStatusModalOpen}
        onClose={() => setEditStatusModalOpen(false)}
        title={t('documents.edit_status_title')}
        maxWidth="md"
      >
        <form onSubmit={handleSaveStatus} className="space-y-4">
          <p className="text-xs text-slate-500 dark:text-slate-400">
            {t('documents.edit_status_desc')}
          </p>

          <div>
            <label className="block text-xs font-medium text-slate-700 dark:text-slate-300">
              {t('documents.select_status')}
            </label>
            <select
              value={newStatus}
              onChange={(e) => setNewStatus(e.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-800 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            >
              <option value="ACTIVE">{t('documents.status_active')}</option>
              <option value="SUPERSEDED">{t('documents.status_superseded')}</option>
              <option value="DRAFT">{t('documents.status_draft')}</option>
              <option value="ARCHIVED">{t('documents.status_archived')}</option>
            </select>
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-700 dark:text-slate-300">
              {t('documents.superseded_by_id_label')}
            </label>
            <input
              type="text"
              value={supersededById}
              onChange={(e) => setSupersededById(e.target.value)}
              placeholder={t('documents.superseded_by_id_placeholder')}
              className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-800 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-700 dark:text-slate-300">
              {t('documents.reason_label')}
            </label>
            <textarea
              value={statusReason}
              onChange={(e) => setStatusReason(e.target.value)}
              rows={3}
              placeholder={t('documents.reason_placeholder')}
              className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-800 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={() => setEditStatusModalOpen(false)}
              className="rounded-lg border border-slate-200 px-4 py-2 text-xs font-medium text-slate-600 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
            >
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              disabled={updateStatusMutation.isPending}
              className="rounded-lg bg-indigo-600 px-4 py-2 text-xs font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {updateStatusMutation.isPending ? t('common.saving') : t('common.save')}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
};
export default DocumentManagePage;
