import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Search,
  MessageSquare,
  MessagesSquare,
  ThumbsUp,
  ThumbsDown,
  Trash2,
  Eye,
  RefreshCw,
  User as UserIcon,
  Bot,
  Layers,
  BookOpen,
  CheckCircle2,
  Sparkles,
} from 'lucide-react';
import {
  chatService,
  type ConversationDetailDTO,
} from '@/services/chatService';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { Badge } from '@/components/common/Badge';
import { Modal } from '@/components/common/Modal';

// Admin page for auditing and moderating student advisory conversations.
export const ChatManagePage: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [page, setPage] = useState(1);
  const [pageSize] = useState(15);
  const [search, setSearch] = useState('');
  const [feedbackFilter, setFeedbackFilter] = useState('');
  const [modelFilter, setModelFilter] = useState('');

  const [inspectModalOpen, setInspectModalOpen] = useState(false);
  const [selectedConversation, setSelectedConversation] = useState<ConversationDetailDTO | null>(null);
  const [isDetailLoading, setIsDetailLoading] = useState(false);

  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  const { data: stats, isFetching: isStatsFetching } = useQuery({
    queryKey: ['admin-chats', 'stats'],
    queryFn: () => chatService.getAdminStats(),
  });

  const { data: models = [] } = useQuery({
    queryKey: ['chat', 'models'],
    queryFn: () => chatService.getModels(),
  });

  const {
    data: convResult,
    isLoading,
    isFetching,
    refetch,
  } = useQuery({
    queryKey: ['admin-chats', 'list', { search, feedbackFilter, modelFilter, page, pageSize }],
    queryFn: () =>
      chatService.listAdminConversations({
        q: search.trim() || undefined,
        feedback: feedbackFilter || undefined,
        model_id: modelFilter || undefined,
        page,
        limit: pageSize,
      }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => chatService.deleteAdminConversation(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-chats'] });
      setDeleteModalOpen(false);
      setDeletingId(null);
      if (inspectModalOpen && selectedConversation?.conversation.id === deletingId) {
        setInspectModalOpen(false);
        setSelectedConversation(null);
      }
      setToastMessage(t('admin_chat.delete_success'));
      setTimeout(() => setToastMessage(null), 3000);
    },
    onError: () => {
      setToastMessage(t('admin_chat.delete_error'));
      setTimeout(() => setToastMessage(null), 3000);
    },
  });

  const conversations = convResult?.data || [];
  const totalRecords = convResult?.pagination?.total_records || 0;
  const totalPages = convResult?.pagination?.total_pages || 1;

  const handleOpenInspect = async (id: string) => {
    setIsDetailLoading(true);
    setInspectModalOpen(true);
    try {
      const data = await chatService.getAdminConversation(id);
      setSelectedConversation(data);
    } catch {
      setInspectModalOpen(false);
    } finally {
      setIsDetailLoading(false);
    }
  };

  const handleConfirmDelete = () => {
    if (!deletingId) return;
    deleteMutation.mutate(deletingId);
  };

  return (
    <div className="space-y-6">
      {toastMessage && (
        <div className="fixed inset-x-4 top-4 z-50 flex items-center justify-center gap-2 px-4 py-3 rounded-xl bg-slate-900 text-white shadow-xl animate-in fade-in slide-in-from-top-4 duration-200 sm:inset-x-auto sm:right-4 sm:max-w-sm sm:justify-start">
          <CheckCircle2 className="w-4 h-4 shrink-0 text-emerald-400" />
          <span className="text-xs font-medium">{toastMessage}</span>
        </div>
      )}

      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div className="min-w-0">
          <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100 flex items-center gap-2">
            <MessagesSquare className="w-6 h-6 sm:w-7 sm:h-7 shrink-0 text-indigo-600 dark:text-indigo-400" />
            <span className="min-w-0">{t('admin_chat.title')}</span>
          </h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t('admin_chat.subtitle')}</p>
        </div>
        <button
          onClick={() => {
            queryClient.invalidateQueries({ queryKey: ['admin-chats'] });
            refetch();
          }}
          disabled={isLoading || isFetching || isStatsFetching}
          className="inline-flex items-center justify-center gap-2 px-3.5 min-h-11 shrink-0 rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-sm sm:text-xs font-semibold text-slate-700 dark:text-slate-300 shadow-xs hover:bg-slate-50 dark:hover:bg-slate-700 transition disabled:opacity-50"
        >
          <RefreshCw className={`w-4 h-4 sm:w-3.5 sm:h-3.5 ${isFetching ? 'animate-spin' : ''}`} />
          <span>{t('admin_chat.btn_refresh')}</span>
        </button>
      </div>

      {stats && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs">
            <div className="flex items-center justify-between gap-2">
              <span className="min-w-0 text-xs font-semibold uppercase tracking-wider text-slate-400">
                {t('admin_chat.stat_total_chats')}
              </span>
              <div className="p-2 rounded-xl bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400 shrink-0">
                <MessageSquare className="w-4 h-4" />
              </div>
            </div>
            <div className="mt-2 text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
              {stats.total_conversations.toLocaleString()}
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs">
            <div className="flex items-center justify-between gap-2">
              <span className="min-w-0 text-xs font-semibold uppercase tracking-wider text-slate-400">
                {t('admin_chat.stat_total_messages')}
              </span>
              <div className="p-2 rounded-xl bg-sky-50 dark:bg-sky-950/60 text-sky-600 dark:text-sky-400 shrink-0">
                <Layers className="w-4 h-4" />
              </div>
            </div>
            <div className="mt-2 text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
              {stats.total_messages.toLocaleString()}
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs">
            <div className="flex items-center justify-between gap-2">
              <span className="min-w-0 text-xs font-semibold uppercase tracking-wider text-slate-400">
                {t('admin_chat.col_feedback')}
              </span>
              <div className="flex items-center gap-1 shrink-0">
                <div className="p-1.5 rounded-lg bg-emerald-50 dark:bg-emerald-950/60 text-emerald-600 dark:text-emerald-400">
                  <ThumbsUp className="w-3.5 h-3.5" />
                </div>
                <div className="p-1.5 rounded-lg bg-rose-50 dark:bg-rose-950/60 text-rose-600 dark:text-rose-400">
                  <ThumbsDown className="w-3.5 h-3.5" />
                </div>
              </div>
            </div>
            <div className="mt-2 flex items-baseline gap-3">
              <span className="text-2xl font-bold tracking-tight text-emerald-600 dark:text-emerald-400">
                +{stats.feedback_up}
              </span>
              <span className="text-sm font-medium text-slate-300 dark:text-slate-700">/</span>
              <span className="text-2xl font-bold tracking-tight text-rose-600 dark:text-rose-400">
                -{stats.feedback_down}
              </span>
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs">
            <div className="flex items-center justify-between gap-2">
              <span className="min-w-0 text-xs font-semibold uppercase tracking-wider text-slate-400">
                {t('admin_chat.stat_satisfaction_rate')}
              </span>
              <div className="p-2 rounded-xl bg-violet-50 dark:bg-violet-950/60 text-violet-600 dark:text-violet-400 shrink-0">
                <Sparkles className="w-4 h-4" />
              </div>
            </div>
            <div className="mt-2 text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
              {stats.satisfaction_rate.toFixed(1)}%
            </div>
          </div>
        </div>
      )}

      <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 p-4 bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-xs">
        <div className="relative flex-1 min-w-0">
          <Search className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none" />
          <input
            type="text"
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(1);
            }}
            placeholder={t('admin_chat.search_placeholder')}
            className="w-full pl-10 pr-4 min-h-11 py-2 bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700/80 rounded-xl text-base sm:text-xs text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500"
          />
        </div>

        <select
          value={feedbackFilter}
          onChange={(e) => {
            setFeedbackFilter(e.target.value);
            setPage(1);
          }}
          className="w-full sm:w-auto px-3 min-h-11 py-2 bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700/80 rounded-xl text-base sm:text-xs text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/20"
        >
          <option value="">{t('admin_chat.filter_all_feedback')}</option>
          <option value="up">{t('admin_chat.filter_up_only')}</option>
          <option value="down">{t('admin_chat.filter_down_only')}</option>
        </select>

        <select
          value={modelFilter}
          onChange={(e) => {
            setModelFilter(e.target.value);
            setPage(1);
          }}
          className="w-full sm:w-auto px-3 min-h-11 py-2 bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700/80 rounded-xl text-base sm:text-xs text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/20"
        >
          <option value="">{t('admin_chat.filter_all_models')}</option>
          {models.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </select>
      </div>

      <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
        <div className="hidden md:block overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-slate-200 dark:border-slate-800 bg-slate-50/70 dark:bg-slate-800/40 text-[11px] font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                <th className="px-3 lg:px-5 py-3.5">{t('admin_chat.col_student')}</th>
                <th className="px-3 lg:px-5 py-3.5">{t('admin_chat.col_title')}</th>
                <th className="px-3 lg:px-5 py-3.5">{t('admin_chat.col_model')}</th>
                <th className="px-3 lg:px-5 py-3.5 text-center">{t('admin_chat.col_messages')}</th>
                <th className="px-3 lg:px-5 py-3.5 text-center">{t('admin_chat.col_feedback')}</th>
                <th className="px-3 lg:px-5 py-3.5">{t('admin_chat.col_updated_at')}</th>
                <th className="px-3 lg:px-5 py-3.5 text-right">{t('admin_chat.col_actions')}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 text-xs">
              {isLoading ? (
                <tr>
                  <td colSpan={7} className="px-5 py-12 text-center text-slate-400">
                    <LoadingSpinner size="md" />
                  </td>
                </tr>
              ) : conversations.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-5 py-12 text-center text-slate-400">
                    {t('admin_chat.empty_list')}
                  </td>
                </tr>
              ) : (
                conversations.map((item) => (
                  <tr
                    key={item.id}
                    className="hover:bg-slate-50/80 dark:hover:bg-slate-800/50 transition cursor-pointer"
                    onClick={() => handleOpenInspect(item.id)}
                  >
                    <td className="px-3 lg:px-5 py-3.5">
                      <div className="flex items-center gap-2.5">
                        <div className="w-7 h-7 rounded-full bg-gradient-to-tr from-indigo-600 to-violet-500 text-white flex items-center justify-center text-xs font-bold uppercase shrink-0">
                          {item.user_name ? item.user_name.charAt(0) : <UserIcon className="w-3.5 h-3.5" />}
                        </div>
                        <div className="min-w-0 max-w-[11rem] xl:max-w-[14rem]">
                          <div className="font-semibold text-slate-900 dark:text-slate-100 truncate">
                            {item.user_name || t('admin_chat.anonymous')}
                          </div>
                          <div className="text-[11px] text-slate-400 truncate">
                            {item.student_code ? `${item.student_code} • ` : ''}
                            {item.user_email || ''}
                          </div>
                        </div>
                      </div>
                    </td>

                    <td className="px-3 lg:px-5 py-3.5 max-w-[12rem] lg:max-w-[16rem]">
                      <div className="font-medium text-slate-800 dark:text-slate-200 truncate">
                        {item.title}
                      </div>
                    </td>

                    <td className="px-3 lg:px-5 py-3.5">
                      <Badge variant="secondary" size="sm" className="max-w-[9rem]">
                        <span className="truncate">{item.model_id || t('admin_chat.model_default')}</span>
                      </Badge>
                    </td>

                    <td className="px-3 lg:px-5 py-3.5 text-center font-medium text-slate-700 dark:text-slate-300">
                      {item.total_messages}
                    </td>

                    <td className="px-3 lg:px-5 py-3.5 text-center">
                      <div className="inline-flex items-center gap-2">
                        {item.thumbs_up > 0 && (
                          <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-600 dark:text-emerald-400">
                            <ThumbsUp className="w-3 h-3" />
                            {item.thumbs_up}
                          </span>
                        )}
                        {item.thumbs_down > 0 && (
                          <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-rose-600 dark:text-rose-400">
                            <ThumbsDown className="w-3 h-3" />
                            {item.thumbs_down}
                          </span>
                        )}
                        {item.thumbs_up === 0 && item.thumbs_down === 0 && (
                          <span className="text-slate-300 dark:text-slate-600">-</span>
                        )}
                      </div>
                    </td>

                    <td className="px-3 lg:px-5 py-3.5 text-slate-500 dark:text-slate-400 text-[11px] whitespace-nowrap">
                      {new Date(item.updated_at).toLocaleString()}
                    </td>

                    <td className="px-3 lg:px-5 py-2.5 text-right" onClick={(e) => e.stopPropagation()}>
                      <div className="inline-flex items-center gap-1">
                        <button
                          onClick={() => handleOpenInspect(item.id)}
                          title={t('admin_chat.btn_inspect')}
                          aria-label={t('admin_chat.btn_inspect')}
                          className="inline-flex items-center justify-center w-11 h-11 rounded-lg text-slate-400 hover:text-indigo-600 hover:bg-indigo-50 dark:hover:bg-indigo-950/40 transition"
                        >
                          <Eye className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => {
                            setDeletingId(item.id);
                            setDeleteModalOpen(true);
                          }}
                          title={t('admin_chat.btn_delete')}
                          aria-label={t('admin_chat.btn_delete')}
                          className="inline-flex items-center justify-center w-11 h-11 rounded-lg text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 transition"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        <div className="md:hidden divide-y divide-slate-100 dark:divide-slate-800/60">
          {isLoading ? (
            <div className="px-4 py-12 flex justify-center text-slate-400">
              <LoadingSpinner size="md" />
            </div>
          ) : conversations.length === 0 ? (
            <div className="px-4 py-12 text-center text-sm text-slate-400">
              {t('admin_chat.empty_list')}
            </div>
          ) : (
            conversations.map((item) => (
              <div
                key={item.id}
                onClick={() => handleOpenInspect(item.id)}
                className="p-4 cursor-pointer transition hover:bg-slate-50/80 active:bg-slate-100 dark:hover:bg-slate-800/50 dark:active:bg-slate-800"
              >
                <div className="flex items-start gap-3">
                  <div className="w-9 h-9 rounded-full bg-gradient-to-tr from-indigo-600 to-violet-500 text-white flex items-center justify-center text-sm font-bold uppercase shrink-0">
                    {item.user_name ? item.user_name.charAt(0) : <UserIcon className="w-4 h-4" />}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="font-semibold text-sm text-slate-900 dark:text-slate-100 truncate">
                      {item.user_name || t('admin_chat.anonymous')}
                    </div>
                    <div className="text-[11px] text-slate-400 truncate">
                      {item.student_code ? `${item.student_code} • ` : ''}
                      {item.user_email || ''}
                    </div>
                  </div>
                  <div className="flex items-center gap-0.5 shrink-0 -my-1.5" onClick={(e) => e.stopPropagation()}>
                    <button
                      onClick={() => handleOpenInspect(item.id)}
                      aria-label={t('admin_chat.btn_inspect')}
                      className="inline-flex items-center justify-center w-11 h-11 rounded-lg text-slate-400 hover:text-indigo-600 hover:bg-indigo-50 dark:hover:bg-indigo-950/40 transition"
                    >
                      <Eye className="w-5 h-5" />
                    </button>
                    <button
                      onClick={() => {
                        setDeletingId(item.id);
                        setDeleteModalOpen(true);
                      }}
                      aria-label={t('admin_chat.btn_delete')}
                      className="inline-flex items-center justify-center w-11 h-11 rounded-lg text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 transition"
                    >
                      <Trash2 className="w-5 h-5" />
                    </button>
                  </div>
                </div>

                <div className="mt-1.5 text-sm font-medium text-slate-800 dark:text-slate-200 break-words line-clamp-2">
                  {item.title}
                </div>

                <div className="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-[11px] text-slate-500 dark:text-slate-400">
                  <Badge variant="secondary" size="sm" className="max-w-[10rem]">
                    <span className="truncate">{item.model_id || t('admin_chat.model_default')}</span>
                  </Badge>
                  <span className="inline-flex items-center gap-1 font-medium text-slate-600 dark:text-slate-300">
                    <MessageSquare className="w-3 h-3" />
                    {item.total_messages}
                  </span>
                  {item.thumbs_up > 0 && (
                    <span className="inline-flex items-center gap-1 font-semibold text-emerald-600 dark:text-emerald-400">
                      <ThumbsUp className="w-3 h-3" />
                      {item.thumbs_up}
                    </span>
                  )}
                  {item.thumbs_down > 0 && (
                    <span className="inline-flex items-center gap-1 font-semibold text-rose-600 dark:text-rose-400">
                      <ThumbsDown className="w-3 h-3" />
                      {item.thumbs_down}
                    </span>
                  )}
                  {item.thumbs_up === 0 && item.thumbs_down === 0 && (
                    <span className="text-slate-300 dark:text-slate-600">-</span>
                  )}
                  <span className="ml-auto text-slate-400">
                    {new Date(item.updated_at).toLocaleString()}
                  </span>
                </div>
              </div>
            ))
          )}
        </div>

        <div className="p-4 border-t border-slate-200 dark:border-slate-800 flex flex-col-reverse sm:flex-row items-center sm:justify-between gap-3 text-xs text-slate-500 dark:text-slate-400">
          <div className="text-center sm:text-left">
            <span className="sm:hidden">
              {t('common.page_compact', { current: page, total: totalPages })}
            </span>
            <span className="hidden sm:inline">
              {t('common.page', { current: page, total: totalPages, count: totalRecords })}
            </span>
          </div>
          <div className="flex items-center gap-2 w-full sm:w-auto">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1 || isLoading}
              className="flex-1 sm:flex-none inline-flex items-center justify-center min-h-11 px-4 sm:px-3 rounded-lg border border-slate-200 dark:border-slate-700 disabled:opacity-50 hover:bg-slate-50 dark:hover:bg-slate-800 transition"
            >
              {t('common.previous')}
            </button>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages || isLoading}
              className="flex-1 sm:flex-none inline-flex items-center justify-center min-h-11 px-4 sm:px-3 rounded-lg border border-slate-200 dark:border-slate-700 disabled:opacity-50 hover:bg-slate-50 dark:hover:bg-slate-800 transition"
            >
              {t('common.next')}
            </button>
          </div>
        </div>
      </div>

      <Modal
        isOpen={inspectModalOpen}
        onClose={() => {
          setInspectModalOpen(false);
          setSelectedConversation(null);
        }}
        title={t('admin_chat.modal_inspect_title')}
        maxWidth="xl"
      >
        {isDetailLoading || !selectedConversation ? (
          <div className="py-12 flex justify-center text-slate-400">
            <LoadingSpinner size="md" />
          </div>
        ) : (
          <div className="flex flex-col space-y-4 max-h-[50dvh] sm:max-h-[65dvh]">
            <div className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700/80 flex flex-wrap items-center justify-between gap-3 text-xs">
              <div className="flex items-center gap-3 min-w-0">
                <div className="w-8 h-8 rounded-full bg-gradient-to-tr from-indigo-600 to-violet-500 text-white flex items-center justify-center font-bold uppercase shrink-0">
                  {selectedConversation.user?.full_name ? selectedConversation.user.full_name.charAt(0) : <UserIcon className="w-4 h-4" />}
                </div>
                <div className="min-w-0">
                  <div className="font-semibold text-slate-900 dark:text-slate-100 truncate">
                    {selectedConversation.user?.full_name || t('admin_chat.anonymous')}
                  </div>
                  <div className="text-[11px] text-slate-400 truncate">
                    {selectedConversation.user?.student_code ? `${selectedConversation.user.student_code} • ` : ''}
                    {selectedConversation.user?.email || ''}
                  </div>
                </div>
              </div>

              <div className="flex items-center gap-2 shrink-0">
                <Badge variant="secondary" size="sm" className="max-w-[9rem]">
                  <span className="truncate">{selectedConversation.conversation.model_id || t('admin_chat.model_default')}</span>
                </Badge>
                <button
                  onClick={() => {
                    setDeletingId(selectedConversation.conversation.id);
                    setDeleteModalOpen(true);
                  }}
                  className="inline-flex items-center min-h-11 px-3 rounded-lg text-rose-600 dark:text-rose-400 bg-rose-50 dark:bg-rose-950/40 hover:bg-rose-100 dark:hover:bg-rose-900/60 text-xs font-semibold transition"
                >
                  {t('admin_chat.btn_delete')}
                </button>
              </div>
            </div>

            <div className="flex-1 min-h-0 overflow-y-auto space-y-4 p-2 pr-2 sm:pr-3">
              {selectedConversation.messages.length === 0 ? (
                <div className="py-8 text-center text-xs text-slate-400">
                  {t('admin_chat.no_messages')}
                </div>
              ) : (
                selectedConversation.messages.map((msg) => (
                  <div
                    key={msg.id}
                    className={`flex flex-col max-w-full ${
                      msg.sender === 'user' ? 'items-end' : 'items-start'
                    }`}
                  >
                    <div className="flex items-center gap-1.5 mb-1 px-1 max-w-full text-[10px] text-slate-400">
                      {msg.sender === 'user' ? (
                        <>
                          <span className="truncate max-w-[12rem] sm:max-w-[16rem]">
                            {selectedConversation.user?.full_name || t('admin_chat.anonymous')}
                          </span>
                          <UserIcon className="w-3 h-3 shrink-0 text-indigo-500" />
                        </>
                      ) : (
                        <>
                          <Bot className="w-3 h-3 shrink-0 text-emerald-500" />
                          <span>{t('chat.advisor_name')}</span>
                        </>
                      )}
                      <span className="shrink-0">•</span>
                      <span className="shrink-0 whitespace-nowrap">
                        {new Date(msg.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                      </span>
                    </div>

                    <div
                      className={`max-w-[85%] rounded-2xl p-3.5 text-xs leading-relaxed ${
                        msg.sender === 'user'
                          ? 'bg-indigo-600 text-white rounded-br-xs'
                          : 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-slate-100 rounded-bl-xs'
                      }`}
                    >
                      {msg.images && msg.images.length > 0 && (
                        <div className="flex flex-wrap gap-2 mb-2">
                          {msg.images.map((img, i) => (
                            <img
                              key={i}
                              src={img}
                              alt=""
                              className="w-20 h-20 object-cover rounded-lg border border-white/20 shadow-xs"
                            />
                          ))}
                        </div>
                      )}

                      <div className="whitespace-pre-wrap break-words">{msg.content}</div>

                      {msg.sources && msg.sources.length > 0 && (
                        <div className="mt-3 pt-2 border-t border-slate-200/60 dark:border-slate-700/60 text-[11px]">
                          <div className="font-semibold text-slate-500 dark:text-slate-400 flex items-center gap-1 mb-1">
                            <BookOpen className="w-3 h-3 shrink-0 text-indigo-500" />
                            <span>{t('admin_chat.sources_cited')}</span>
                          </div>
                          <ul className="list-disc pl-4 space-y-0.5 text-slate-600 dark:text-slate-300 break-words">
                            {msg.sources.map((src, sIdx) => (
                              <li key={sIdx}>{src}</li>
                            ))}
                          </ul>
                        </div>
                      )}

                      {msg.feedback && (
                        <div className="mt-2 pt-1.5 flex items-center gap-1 text-[11px]">
                          {msg.feedback === 'up' ? (
                            <span className="inline-flex items-center gap-1 text-emerald-600 dark:text-emerald-400 font-medium">
                              <ThumbsUp className="w-3 h-3" />
                              <span>{t('chat.helpful')}</span>
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 text-rose-600 dark:text-rose-400 font-medium">
                              <ThumbsDown className="w-3 h-3" />
                              <span>{t('chat.not_helpful')}</span>
                            </span>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                ))
              )}
            </div>

            <div className="pt-2 flex">
              <button
                onClick={() => {
                  setInspectModalOpen(false);
                  setSelectedConversation(null);
                }}
                className="w-full sm:w-auto inline-flex items-center justify-center min-h-11 px-4 rounded-xl bg-slate-100 dark:bg-slate-800 text-xs font-semibold text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700 transition"
              >
                {t('admin_chat.close')}
              </button>
            </div>
          </div>
        )}
      </Modal>

      <Modal
        isOpen={deleteModalOpen}
        onClose={() => {
          setDeleteModalOpen(false);
          setDeletingId(null);
        }}
        title={t('admin_chat.btn_delete')}
        maxWidth="sm"
      >
        <div className="space-y-4">
          <p className="text-sm sm:text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
            {t('admin_chat.modal_delete_confirm')}
          </p>
          <div className="flex gap-2 pt-2">
            <button
              onClick={() => {
                setDeleteModalOpen(false);
                setDeletingId(null);
              }}
              disabled={deleteMutation.isPending}
              className="flex-1 sm:flex-none inline-flex items-center justify-center min-h-11 px-3.5 rounded-xl border border-slate-200 dark:border-slate-700 text-sm sm:text-xs font-semibold text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800 transition disabled:opacity-50"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={handleConfirmDelete}
              disabled={deleteMutation.isPending}
              className="flex-1 sm:flex-none inline-flex items-center justify-center min-h-11 px-3.5 rounded-xl bg-rose-600 hover:bg-rose-700 text-white text-sm sm:text-xs font-semibold shadow-xs disabled:opacity-50 transition"
            >
              {deleteMutation.isPending ? t('common.loading') : t('common.delete')}
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
};
