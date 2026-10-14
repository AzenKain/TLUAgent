import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Search,
  BookOpen,
  Edit3,
  Eye,
  CheckCircle2,
  AlertTriangle,
  Layers,
  FileEdit,
} from 'lucide-react';
import {
  ragAdminService,
  type DocumentDTO,
} from '@/services/ragAdminService';
import { curriculumAdminService } from '@/services/curriculumAdminService';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { Badge } from '@/components/common/Badge';
import { Modal } from '@/components/common/Modal';
import { DocumentViewerModal } from '@/components/common/DocumentViewerModal';
import { PageHeader } from '@/components/common/PageHeader';
import { RefreshButton } from '@/components/common/RefreshButton';
import { StatCard } from '@/components/common/StatCard';

const formatApplicableCohortBadge = (rawCohort?: string): string => {
  if (!rawCohort) return 'ALL';
  const trimmed = rawCohort.trim();
  if (!trimmed || trimmed.toUpperCase() === 'ALL') return 'ALL';
  const parts = trimmed
    .split(',')
    .map((p) => p.trim().toUpperCase())
    .filter(Boolean);
  if (parts.length <= 1) return parts[0] || 'ALL';
  const nums = parts
    .map((p) => parseInt(p.replace(/[^0-9]/g, ''), 10))
    .filter((n) => !Number.isNaN(n))
    .toSorted((a, b) => a - b);
  if (nums.length >= 2) {
    const isContiguous = nums.every((n, i) => i === 0 || n === nums[i - 1] + 1);
    if (isContiguous && nums.length >= 3) {
      return `K${nums[0]} - K${nums[nums.length - 1]}`;
    }
  }
  return parts.join(', ');
};

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

  const { data: cohortsData } = useQuery({
    queryKey: ['curriculumCohorts'],
    queryFn: () => curriculumAdminService.getCohorts(),
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

  const { data: docDetail } = useQuery({
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
      case 'PARTIALLY_AMENDED':
      case 'PARTIALLY_SUPERSEDED':
        return <Badge variant="warning">{t('documents.status_partially_amended')}</Badge>;
      case 'SUPERSEDED':
        return <Badge variant="danger">{t('documents.status_superseded')}</Badge>;
      case 'DRAFT':
        return <Badge variant="info">{t('documents.status_draft')}</Badge>;
      default:
        return <Badge variant="secondary">{status}</Badge>;
    }
  };

  return (
    <div className="space-y-6 max-w-7xl mx-auto pb-10">
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

      <PageHeader
        icon={<BookOpen className="h-6 w-6" />}
        title={t('documents.title')}
        subtitle={t('documents.subtitle')}
        actions={
          <div className="flex items-center gap-2">
            <RefreshButton
              onClick={() => {
                queryClient.invalidateQueries({ queryKey: ['admin-documents'] });
                refetch();
              }}
              isLoading={isLoading || isFetching}
            />
            <button
              type="button"
              onClick={() => syncMutation.mutate()}
              disabled={syncMutation.isPending}
              className="inline-flex min-h-10 items-center justify-center gap-2 rounded-xl bg-tlu-blue px-3.5 py-2 text-xs font-semibold text-white shadow-xs transition hover:bg-tlu-navy-light disabled:opacity-50"
            >
              {syncMutation.isPending && <LoadingSpinner size="sm" />}
              <span>{syncMutation.isPending ? t('documents.btn_syncing') : t('documents.btn_sync')}</span>
            </button>
          </div>
        }
      />

      {/* Stats Summary Cards */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5 sm:gap-4">
        <StatCard
          icon={<BookOpen className="h-5 w-5" />}
          label={t('documents.stat_total_docs')}
          value={stats?.total_documents ?? 0}
          variant="blue"
        />
        <StatCard
          icon={<CheckCircle2 className="h-5 w-5" />}
          label={t('documents.stat_active_docs')}
          value={stats?.active_documents ?? 0}
          variant="emerald"
        />
        <StatCard
          icon={<FileEdit className="h-5 w-5" />}
          label={t('documents.stat_partially_amended_docs')}
          value={stats?.partially_amended_documents ?? 0}
          variant="amber"
        />
        <StatCard
          icon={<AlertTriangle className="h-5 w-5" />}
          label={t('documents.stat_superseded_docs')}
          value={stats?.superseded_documents ?? 0}
          variant="rose"
        />
        <StatCard
          icon={<Layers className="h-5 w-5" />}
          label={t('documents.stat_total_chunks')}
          value={stats?.total_chunks ?? 0}
          variant="sky"
        />
      </div>

      <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-xs dark:border-blue-900/30 dark:bg-[#111A35]">
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
              className="w-full rounded-lg border border-slate-200 bg-white py-2 pl-9 pr-3 text-sm text-slate-800 placeholder-slate-400 focus:border-tlu-blue focus:outline-hidden focus:ring-1 focus:ring-tlu-blue dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100 dark:placeholder-slate-500"
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
              <option value="QUY_CHE">{t('documents.domain_quy_che')}</option>
              <option value="HOC_VU">{t('documents.domain_hoc_vu')}</option>
              <option value="HOC_PHI">{t('documents.domain_hoc_phi')}</option>
              <option value="TUYEN_SINH">{t('documents.domain_tuyen_sinh')}</option>
              <option value="CHUAN_DAU_RA">{t('documents.domain_chuan_dau_ra')}</option>
              <option value="CO_VAN_HOC_TAP">{t('documents.domain_cvht')}</option>
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
              <option value="PARTIALLY_AMENDED">{t('documents.status_partially_amended')}</option>
              <option value="SUPERSEDED">{t('documents.status_superseded')}</option>
              <option value="DRAFT">{t('documents.status_draft')}</option>
              <option value="ARCHIVED">{t('documents.status_archived')}</option>
            </select>
          </div>

          <div>
            <select
              value={cohortFilter}
              onChange={(e) => {
                setCohortFilter(e.target.value);
                setPage(1);
              }}
              className="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300"
            >
              <option value="">{t('documents.cohort_all')}</option>
              {cohortsData?.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xs dark:border-blue-900/30 dark:bg-[#111A35]">
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
                  <tr
                    key={doc.id}
                    onClick={() => {
                      setSelectedDocId(doc.id);
                      setInspectModalOpen(true);
                    }}
                    className="hover:bg-blue-50/50 dark:hover:bg-slate-800/60 cursor-pointer transition-colors group"
                  >
                    <td className="whitespace-nowrap px-4 py-3 font-mono text-xs font-semibold text-slate-700 dark:text-slate-300">
                      {doc.doc_code || doc.id}
                    </td>
                    <td className="px-4 py-3">
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          setSelectedDocId(doc.id);
                          setInspectModalOpen(true);
                        }}
                        className="max-w-xs font-semibold text-tlu-navy group-hover:text-tlu-blue dark:text-blue-300 dark:group-hover:text-blue-200 truncate cursor-pointer text-left block"
                      >
                        {doc.title}
                      </button>
                      <div className="text-[11px] text-slate-400">
                        {doc.type} • {doc.publish_date}
                      </div>
                    </td>
                    <td className="whitespace-nowrap px-4 py-3">
                      <span className="rounded-md bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-300">
                        {doc.domain}
                      </span>
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-xs font-medium">
                      <span
                        className="inline-flex items-center rounded-md border border-slate-200 bg-slate-50 px-2 py-0.5 text-xs font-semibold text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300"
                        title={doc.applicable_cohort}
                      >
                        {formatApplicableCohortBadge(doc.applicable_cohort)}
                      </span>
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
                          <span className="truncate max-w-37.5">
                            {doc.status === 'PARTIALLY_AMENDED' || doc.status === 'PARTIALLY_SUPERSEDED'
                              ? t('documents.partially_amended_by', { doc: doc.superseded_by_id })
                              : t('documents.superseded_by', { doc: doc.superseded_by_id })}
                          </span>
                        </div>
                      ) : (
                        <span className="text-slate-400">-</span>
                      )}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-1.5">
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            setSelectedDocId(doc.id);
                            setInspectModalOpen(true);
                          }}
                          className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg bg-blue-50 text-tlu-blue hover:bg-tlu-blue hover:text-white dark:bg-blue-950/60 dark:text-blue-300 dark:hover:bg-tlu-blue dark:hover:text-white transition-all text-xs font-semibold cursor-pointer shadow-2xs"
                          title={t('documents.inspect_title')}
                        >
                          <Eye className="h-3.5 w-3.5" />
                          <span>{t('documents.view_detail')}</span>
                        </button>
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleOpenEditStatus(doc);
                          }}
                          className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-amber-600 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-amber-400 cursor-pointer"
                          title={t('documents.edit_status_title')}
                        >
                          <Edit3 className="h-4 w-4" />
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

      <DocumentViewerModal
        isOpen={inspectModalOpen}
        onClose={() => setInspectModalOpen(false)}
        documentIdentifier={selectedDocId}
        initialDocDetail={docDetail}
      />

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
              <option value="PARTIALLY_AMENDED">{t('documents.status_partially_amended')}</option>
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
              className="rounded-lg bg-tlu-navy px-4 py-2 text-xs font-medium text-white hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-blue-600 disabled:opacity-50"
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
