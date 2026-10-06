import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import {
  Search,
  RefreshCw,
  Eye,
  CheckCircle2,
  AlertTriangle,
  Clock,
  Send,
  ShieldAlert,
  Bot,
} from 'lucide-react';
import { inquiryService, type InquiryDTO } from '@/services/inquiryService';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { Badge } from '@/components/common/Badge';
import { Modal } from '@/components/common/Modal';

export const InquiryManagePage: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();

  const [activeTab, setActiveTab] = useState<'PENDING' | 'ANSWERED' | 'EXPIRED'>('PENDING');
  const [page, setPage] = useState(1);
  const [pageSize] = useState(15);
  const [search, setSearch] = useState('');

  const [selectedInquiry, setSelectedInquiry] = useState<InquiryDTO | null>(null);
  const [reviewModalOpen, setReviewModalOpen] = useState(false);

  const [answerText, setAnswerText] = useState('');
  const [supersedeReason, setSupersedeReason] = useState('');

  const [toastMessage, setToastMessage] = useState<string | null>(null);
  const [toastType, setToastType] = useState<'success' | 'error'>('success');

  const showToast = (message: string, type: 'success' | 'error' = 'success') => {
    setToastMessage(message);
    setToastType(type);
    setTimeout(() => setToastMessage(null), 4000);
  };

  const {
    data: inquiryData,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ['advisor-inquiries', activeTab, search, page, pageSize],
    queryFn: () => inquiryService.getTeacherInquiries(activeTab, search, page, pageSize),
  });

  const handleOpenReview = React.useCallback((inquiry: InquiryDTO) => {
    setSelectedInquiry(inquiry);
    setAnswerText(inquiry.teacher_reply || '');
    setSupersedeReason(inquiry.expired_reason || inquiry.context || '');
    setReviewModalOpen(true);
  }, []);

  useEffect(() => {
    const targetId = searchParams.get('id');
    if (targetId) {
      inquiryService.getInquiry(targetId).then((data) => {
        if (data) {
          handleOpenReview(data);
        }
      });
    }
  }, [searchParams, handleOpenReview]);

  const answerMutation = useMutation({
    mutationFn: (data: { id: string; reply: string }) => inquiryService.answerInquiry(data.id, data.reply),
    onSuccess: () => {
      showToast(t('inquiries.answer_success'));
      setReviewModalOpen(false);
      queryClient.invalidateQueries({ queryKey: ['advisor-inquiries'] });
    },
    onError: () => {
      showToast(t('common.error'), 'error');
    },
  });

  const expireMutation = useMutation({
    mutationFn: (data: { id: string; superseded_by_doc_id?: string; reason: string }) =>
      inquiryService.expireInquiry(data.id, {
        superseded_by_doc_id: data.superseded_by_doc_id,
        reason: data.reason,
      }),
    onSuccess: () => {
      showToast(t('inquiries.confirm_supersede_success'));
      setReviewModalOpen(false);
      queryClient.invalidateQueries({ queryKey: ['advisor-inquiries'] });
    },
    onError: () => {
      showToast(t('common.error'), 'error');
    },
  });

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'ANSWERED':
        return <Badge variant="success">{t('inquiries.status_answered')}</Badge>;
      case 'EXPIRED':
        return <Badge variant="danger">{t('inquiries.status_expired')}</Badge>;
      case 'REJECTED':
        return <Badge variant="danger">{t('inquiries.status_rejected')}</Badge>;
      default:
        return <Badge variant="warning">{t('inquiries.status_pending')}</Badge>;
    }
  };

  const isAISupersession =
    selectedInquiry?.student_code === 'AI_AGENT' ||
    Boolean(selectedInquiry?.context?.toLowerCase().includes('superseded'));

  return (
    <div className="space-y-6">
      {toastMessage && (
        <div
          className={`fixed bottom-6 right-6 z-50 rounded-xl px-5 py-3 shadow-xl backdrop-blur-md transition-all ${
            toastType === 'success'
              ? 'bg-emerald-600/90 text-white'
              : 'bg-rose-600/90 text-white'
          }`}
        >
          {toastMessage}
        </div>
      )}

      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-100">{t('inquiries.title')}</h1>
          <p className="mt-1 text-sm text-slate-400">{t('inquiries.sub')}</p>
        </div>
        <button
          onClick={() => refetch()}
          className="flex items-center gap-2 rounded-xl border border-slate-700 bg-slate-800/80 px-4 py-2.5 text-sm font-medium text-slate-300 transition-colors hover:bg-slate-700/80 hover:text-white"
        >
          <RefreshCw className="h-4 w-4" />
          <span>{t('common.loading')}</span>
        </button>
      </div>

      <div className="flex border-b border-slate-800">
        <button
          onClick={() => {
            setActiveTab('PENDING');
            setPage(1);
          }}
          className={`flex items-center gap-2 border-b-2 px-5 py-3 text-sm font-medium transition-colors ${
            activeTab === 'PENDING'
              ? 'border-indigo-500 text-indigo-400'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <Clock className="h-4 w-4" />
          <span>{t('inquiries.tabs.pending')}</span>
        </button>
        <button
          onClick={() => {
            setActiveTab('ANSWERED');
            setPage(1);
          }}
          className={`flex items-center gap-2 border-b-2 px-5 py-3 text-sm font-medium transition-colors ${
            activeTab === 'ANSWERED'
              ? 'border-indigo-500 text-indigo-400'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <CheckCircle2 className="h-4 w-4" />
          <span>{t('inquiries.tabs.answered')}</span>
        </button>
        <button
          onClick={() => {
            setActiveTab('EXPIRED');
            setPage(1);
          }}
          className={`flex items-center gap-2 border-b-2 px-5 py-3 text-sm font-medium transition-colors ${
            activeTab === 'EXPIRED'
              ? 'border-indigo-500 text-indigo-400'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <AlertTriangle className="h-4 w-4" />
          <span>{t('inquiries.tabs.expired')}</span>
        </button>
      </div>

      <div className="flex items-center gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
          <input
            type="text"
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(1);
            }}
            placeholder={t('inquiries.filter_search_placeholder')}
            className="w-full rounded-xl border border-slate-700 bg-slate-800/80 pl-10 pr-4 py-2 text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />
        </div>
      </div>

      <div className="overflow-hidden rounded-2xl border border-slate-800 bg-slate-900/60 shadow-xl backdrop-blur-sm">
        {isLoading ? (
          <div className="flex h-64 items-center justify-center">
            <LoadingSpinner size="lg" />
          </div>
        ) : !inquiryData?.items || inquiryData.items.length === 0 ? (
          <div className="flex h-64 flex-col items-center justify-center text-slate-500">
            <Clock className="h-10 w-10 stroke-[1.5]" />
            <p className="mt-3 text-sm">
              {activeTab === 'PENDING'
                ? t('inquiries.empty_pending')
                : activeTab === 'ANSWERED'
                ? t('inquiries.empty_answered')
                : t('inquiries.empty_expired')}
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-slate-300">
              <thead className="border-b border-slate-800 bg-slate-800/40 text-xs uppercase text-slate-400">
                <tr>
                  <th className="px-6 py-4">{t('inquiries.student_name')}</th>
                  <th className="px-6 py-4">{t('inquiries.question')}</th>
                  <th className="px-6 py-4">{t('inquiries.status')}</th>
                  <th className="px-6 py-4">{t('inquiries.created_at')}</th>
                  <th className="px-6 py-4 text-right">{t('common.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60">
                {inquiryData.items.map((inq) => {
                  const isAgent = inq.student_code === 'AI_AGENT';
                  return (
                    <tr key={inq.id} className="transition-colors hover:bg-slate-800/30">
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-2">
                          {isAgent ? (
                            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-indigo-950 border border-indigo-700/60 text-indigo-400">
                              <Bot className="h-4 w-4" />
                            </div>
                          ) : (
                            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-slate-800 border border-slate-700 text-xs font-semibold text-slate-300">
                              {inq.student_name.charAt(0)}
                            </div>
                          )}
                          <div>
                            <div className="font-medium text-slate-200">{inq.student_name}</div>
                            <div className="text-xs text-slate-500 font-mono">
                              {inq.student_code}
                              {inq.student_class ? ` • ${inq.student_class}` : ''}
                            </div>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <p className="max-w-md truncate text-slate-300">{inq.question}</p>
                      </td>
                      <td className="px-6 py-4">{getStatusBadge(inq.status)}</td>
                      <td className="px-6 py-4 text-xs text-slate-400">
                        {new Date(inq.created_at).toLocaleString()}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <button
                          onClick={() => handleOpenReview(inq)}
                          className="inline-flex items-center gap-1 rounded-lg border border-slate-700 bg-slate-800/60 px-3 py-1.5 text-xs font-medium text-slate-300 transition-colors hover:bg-slate-700 hover:text-white"
                        >
                          <Eye className="h-3.5 w-3.5" />
                          <span>{t('common.edit')}</span>
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {reviewModalOpen && selectedInquiry && (
        <Modal
          isOpen={reviewModalOpen}
          onClose={() => setReviewModalOpen(false)}
          title={t('inquiries.review_modal_title')}
        >
          <div className="space-y-5">
            {isAISupersession && (
              <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4">
                <div className="flex items-start gap-3">
                  <ShieldAlert className="mt-0.5 h-5 w-5 shrink-0 text-amber-400" />
                  <div>
                    <h4 className="text-sm font-semibold text-amber-300">
                      {t('inquiries.superseded_alert_title')}
                    </h4>
                    <p className="mt-1 text-xs text-amber-200/80">
                      {t('inquiries.superseded_alert_desc')}
                    </p>
                  </div>
                </div>
              </div>
            )}

            <div>
              <label className="text-xs font-semibold uppercase text-slate-400">
                {t('inquiries.student_name')}
              </label>
              <div className="mt-1 flex items-center gap-2">
                <span className="font-medium text-slate-200">{selectedInquiry.student_name}</span>
                <span className="font-mono text-xs text-indigo-400">({selectedInquiry.student_code})</span>
                {selectedInquiry.student_class && (
                  <span className="font-mono text-xs text-slate-400">• {selectedInquiry.student_class}</span>
                )}
              </div>
            </div>

            <div>
              <label className="text-xs font-semibold uppercase text-slate-400">
                {t('inquiries.question')}
              </label>
              <div className="mt-1 rounded-xl bg-slate-950 p-3.5 text-sm text-slate-200 border border-slate-800">
                {selectedInquiry.question}
              </div>
            </div>

            {selectedInquiry.context && (
              <div>
                <label className="text-xs font-semibold uppercase text-slate-400">
                  {t('inquiries.context')}
                </label>
                <div className="mt-1 rounded-xl bg-slate-950 p-3.5 text-xs text-slate-300 border border-slate-800">
                  {selectedInquiry.context}
                </div>
              </div>
            )}

            {selectedInquiry.status === 'PENDING' && (
              <div className="space-y-4 pt-2 border-t border-slate-800">
                {isAISupersession ? (
                  <div>
                    <label className="text-xs font-semibold uppercase text-slate-400">
                      {t('inquiries.expired_reason')}
                    </label>
                    <textarea
                      rows={3}
                      value={supersedeReason}
                      onChange={(e) => setSupersedeReason(e.target.value)}
                      className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 p-3 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                    />
                    <div className="mt-4 flex justify-end gap-3">
                      <button
                        onClick={() => setReviewModalOpen(false)}
                        className="rounded-xl border border-slate-700 px-4 py-2 text-sm font-medium text-slate-300 hover:bg-slate-800"
                      >
                        {t('common.cancel')}
                      </button>
                      <button
                        onClick={() =>
                          expireMutation.mutate({
                            id: selectedInquiry.id,
                            superseded_by_doc_id: selectedInquiry.superseded_by_doc_id,
                            reason: supersedeReason,
                          })
                        }
                        className="flex items-center gap-2 rounded-xl bg-amber-600 px-4 py-2 text-sm font-medium text-white shadow-lg shadow-amber-600/20 hover:bg-amber-500"
                      >
                        <AlertTriangle className="h-4 w-4" />
                        <span>{t('inquiries.confirm_supersede_btn')}</span>
                      </button>
                    </div>
                  </div>
                ) : (
                  <div>
                    <label className="text-xs font-semibold uppercase text-slate-400">
                      {t('inquiries.teacher_reply')}
                    </label>
                    <textarea
                      rows={4}
                      value={answerText}
                      onChange={(e) => setAnswerText(e.target.value)}
                      placeholder={t('inquiries.answer_placeholder')}
                      className="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 p-3 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                    />
                    <div className="mt-4 flex justify-end gap-3">
                      <button
                        onClick={() => setReviewModalOpen(false)}
                        className="rounded-xl border border-slate-700 px-4 py-2 text-sm font-medium text-slate-300 hover:bg-slate-800"
                      >
                        {t('common.cancel')}
                      </button>
                      <button
                        onClick={() =>
                          answerMutation.mutate({
                            id: selectedInquiry.id,
                            reply: answerText,
                          })
                        }
                        className="flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-lg shadow-indigo-600/20 hover:bg-indigo-500"
                      >
                        <Send className="h-4 w-4" />
                        <span>{t('inquiries.answer_btn')}</span>
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )}

            {selectedInquiry.status === 'ANSWERED' && (
              <div className="rounded-xl border border-emerald-500/20 bg-emerald-950/20 p-4">
                <div className="flex items-center gap-2 text-emerald-400 font-semibold text-xs uppercase">
                  <CheckCircle2 className="h-4 w-4" />
                  <span>
                    {t('inquiries.teacher_name')}: {selectedInquiry.teacher_name}
                  </span>
                </div>
                <p className="mt-2 text-sm text-slate-200">{selectedInquiry.teacher_reply}</p>
                {selectedInquiry.answered_at && (
                  <span className="mt-2 block text-xs text-slate-500">
                    {t('inquiries.answered_at')}: {new Date(selectedInquiry.answered_at).toLocaleString()}
                  </span>
                )}
              </div>
            )}

            {selectedInquiry.status === 'EXPIRED' && (
              <div className="rounded-xl border border-slate-800 bg-slate-950 p-4 text-xs text-slate-400">
                <p className="font-semibold text-rose-400">{t('inquiries.status_expired')}</p>
                <p className="mt-1 text-slate-300">{selectedInquiry.expired_reason}</p>
              </div>
            )}
          </div>
        </Modal>
      )}
    </div>
  );
};
