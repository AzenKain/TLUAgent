import React, { useState, useEffect, useRef, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Sparkles,
  Send,
  Paperclip,
  Image as ImageIcon,
  X,
  Plus,
  Trash2,
  Layers,
  ChevronRight,
  ChevronDown,
  ShieldAlert,
  Bot,
  User as UserIcon,
  FileText,
  RotateCcw,
  History,
  Eye,
  Zap,
} from 'lucide-react';
import { tluclawAdminService } from '@/services/tluclawAdminService';
import { TLUClawPreviewCard } from '@/components/admin/tluclaw/TLUClawPreviewCard';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { ConfirmModal } from '@/components/common/ConfirmModal';
import { RefreshButton } from '@/components/common/RefreshButton';
import type {
  TLUClawMessage,
  TLUClawAttachment,
  TLUClawPreviewCardData,
} from '@/types/tluclaw';

async function handleConfirmAction(snapshotId: string, nonce: string, selectedKeys?: string[]): Promise<void> {
  await tluclawAdminService.confirmAction({
    snapshot_id: snapshotId,
    nonce,
    selected_keys: selectedKeys,
  });
}

async function handleCancelAction(snapshotId: string, nonce: string): Promise<void> {
  await tluclawAdminService.cancelAction({
    snapshot_id: snapshotId,
    nonce,
  });
}

export const AdminTLUClawPage: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null);
  const [optimisticMessages, setOptimisticMessages] = useState<TLUClawMessage[]>([]);
  const [inputPrompt, setInputPrompt] = useState<string>('');
  const [attachedImages, setAttachedImages] = useState<string[]>([]);
  const [attachedFiles, setAttachedFiles] = useState<TLUClawAttachment[]>([]);
  const [loading, setLoading] = useState<boolean>(false);

  const [activeTab, setActiveTab] = useState<'chat' | 'timeline'>('chat');
  const [timelineStatus, setTimelineStatus] = useState<string>('');
  const [timelinePage, setTimelinePage] = useState<number>(1);
  const timelineLimit = 20;
  const [rollingBackBatchId, setRollingBackBatchId] = useState<string | null>(null);
  const [confirmRollbackBatchId, setConfirmRollbackBatchId] = useState<string | null>(null);
  const [sessionToDelete, setSessionToDelete] = useState<string | null>(null);
  const [isDeletingSession, setIsDeletingSession] = useState<boolean>(false);
  const [expandedDiffBatches, setExpandedDiffBatches] = useState<Set<string>>(new Set());

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);

  const { data: snapshots = [], isLoading: loadingSnapshots } = useQuery({
    queryKey: ['tluclawSnapshots', timelineStatus, timelinePage],
    queryFn: () =>
      tluclawAdminService.listSnapshots(
        timelineStatus,
        timelineLimit,
        (timelinePage - 1) * timelineLimit
      ),
    enabled: activeTab === 'timeline',
  });

  const handleRollbackBatch = async (batchId: string) => {
    setRollingBackBatchId(batchId);
    try {
      await tluclawAdminService.rollbackBatch(batchId);
      queryClient.invalidateQueries({ queryKey: ['tluclawSnapshots'] });
      queryClient.invalidateQueries({ queryKey: ['tluclawSessions'] });
      if (activeSessionId) {
        queryClient.invalidateQueries({ queryKey: ['tluclawSession', activeSessionId] });
      }
    } finally {
      setRollingBackBatchId(null);
      setConfirmRollbackBatchId(null);
    }
  };

  const toggleDiffExpand = (batchId: string) => {
    setExpandedDiffBatches((prev) => {
      const next = new Set(prev);
      if (next.has(batchId)) {
        next.delete(batchId);
      } else {
        next.add(batchId);
      }
      return next;
    });
  };

  const { data: sessions = [], isLoading: loadingSessions } = useQuery({
    queryKey: ['tluclawSessions'],
    queryFn: () => tluclawAdminService.listSessions(),
  });

  const activeSessionId = selectedSessionId || (sessions.length > 0 ? sessions[0].id : null);

  const { data: sessionDetail } = useQuery({
    queryKey: ['tluclawSession', activeSessionId],
    queryFn: () => (activeSessionId ? tluclawAdminService.getSessionDetail(activeSessionId) : null),
    enabled: !!activeSessionId,
  });

  const displayedMessages = useMemo(() => {
    const serverMsgs: TLUClawMessage[] = (sessionDetail?.messages || []).map((m) => {
      let card: TLUClawPreviewCardData | undefined;
      if (sessionDetail?.snapshots && sessionDetail.snapshots.length > 0) {
        const matched = sessionDetail.snapshots.find(
          (s) => s.snapshot_id === m.id || s.batch_id === m.id
        );
        if (matched) card = matched;
      }
      return {
        id: m.id,
        session_id: m.session_id,
        role: m.role as 'user' | 'assistant',
        content: m.content,
        reasoning_content: m.reasoning_content,
        preview_card: card,
        created_at: m.created_at,
      };
    });
    return [...serverMsgs, ...optimisticMessages];
  }, [sessionDetail, optimisticMessages]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  });

  const handleCreateNewSession = async () => {
    try {
      const newSess = await tluclawAdminService.createSession(t('tluclaw.session.default_title'));
      queryClient.setQueryData(['tluclawSessions'], (prev: typeof sessions) => [newSess, ...(prev || [])]);
      setSelectedSessionId(newSess.id);
      setOptimisticMessages([]);
    } catch {
      // ignored
    }
  };

  const confirmDeleteSession = async () => {
    if (!sessionToDelete) return;
    try {
      setIsDeletingSession(true);
      await tluclawAdminService.deleteSession(sessionToDelete);
      queryClient.setQueryData(['tluclawSessions'], (prev: typeof sessions) =>
        (prev || []).filter((s) => s.id !== sessionToDelete)
      );
      if (activeSessionId === sessionToDelete) {
        setSelectedSessionId(null);
        setOptimisticMessages([]);
      }
      setSessionToDelete(null);
    } catch {
      // ignored
    } finally {
      setIsDeletingSession(false);
    }
  };

  const handleImageUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files) return;

    Array.from(files).forEach((file) => {
      const reader = new FileReader();
      reader.addEventListener('load', (event) => {
        if (event.target?.result) {
          setAttachedImages((prev) => [...prev, event.target!.result as string]);
        }
      });
      reader.readAsDataURL(file);
    });
    e.target.value = '';
  };

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files) return;

    Array.from(files).forEach((file) => {
      const reader = new FileReader();
      reader.addEventListener('load', (event) => {
        if (event.target?.result) {
          const content = event.target.result as string;
          setAttachedFiles((prev) => [
            ...prev,
            {
              file_name: file.name,
              file_type: file.type || 'text/plain',
              data: content,
            },
          ]);
        }
      });
      reader.readAsText(file);
    });
    e.target.value = '';
  };

  const handleSendMessage = async (customPrompt?: string) => {
    const textToSend = (customPrompt || inputPrompt).trim();
    if (!textToSend && attachedImages.length === 0 && attachedFiles.length === 0) return;
    if (loading) return;

    const tempUserMsg: TLUClawMessage = {
      id: `temp_user_${Date.now()}`,
      session_id: activeSessionId || '',
      role: 'user',
      content: textToSend,
      created_at: new Date().toISOString(),
    };

    setOptimisticMessages((prev) => [...prev, tempUserMsg]);
    setInputPrompt('');
    const currentImages = [...attachedImages];
    const currentFiles = [...attachedFiles];
    setAttachedImages([]);
    setAttachedFiles([]);
    setLoading(true);

    try {
      const resp = await tluclawAdminService.sendMessage({
        session_id: activeSessionId || undefined,
        message: textToSend,
        images: currentImages.length > 0 ? currentImages : undefined,
        attachments: currentFiles.length > 0 ? currentFiles : undefined,
      });

      if (!activeSessionId && resp.session_id) {
        setSelectedSessionId(resp.session_id);
      }

      await queryClient.invalidateQueries({ queryKey: ['tluclawSessions'] });
      await queryClient.invalidateQueries({ queryKey: ['tluclawSession', resp.session_id] });
      setOptimisticMessages([]);
    } catch {
      const errorMsg: TLUClawMessage = {
        id: `err_${Date.now()}`,
        session_id: activeSessionId || '',
        role: 'assistant',
        content: t('tluclaw.chat.error_response'),
        created_at: new Date().toISOString(),
      };
      setOptimisticMessages((prev) => [...prev, errorMsg]);
    } finally {
      setLoading(false);
    }
  };

  const handleQuickPrompt = (promptKey: string) => {
    const text = t(promptKey);
    setInputPrompt(text);
  };

  return (
    <div className="flex h-full w-full flex-col md:flex-row bg-background text-foreground overflow-hidden">
      {/* Hidden file inputs */}
      <input
        type="file"
        ref={imageInputRef}
        onChange={handleImageUpload}
        accept="image/*"
        multiple
        className="hidden"
      />
      <input
        type="file"
        ref={fileInputRef}
        onChange={handleFileUpload}
        accept=".pdf,.docx,.doc,.xlsx,.xls,.csv,.txt,.md"
        multiple
        className="hidden"
      />

      {/* Left Sidebar: Sessions History */}
      <div className="w-full md:w-80 border-r border-border bg-card/60 flex flex-col shrink-0 h-full overflow-hidden">
        <div className="p-4 border-b border-border flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Bot className="h-5 w-5 text-primary" />
            <h2 className="font-semibold text-sm tracking-wide">
              {t('tluclaw.sidebar.sessions_title')}
            </h2>
          </div>
          <button
            type="button"
            onClick={handleCreateNewSession}
            className="inline-flex items-center gap-1 rounded-lg bg-primary/10 px-2.5 py-1.5 text-xs font-semibold text-primary hover:bg-primary/20 transition-colors"
          >
            <Plus className="h-4 w-4" />
            {t('tluclaw.sidebar.btn_new_session')}
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-2 space-y-1">
          {loadingSessions ? (
            <div className="flex h-32 items-center justify-center">
              <LoadingSpinner className="h-5 w-5 text-muted-foreground" />
            </div>
          ) : sessions.length === 0 ? (
            <div className="p-4 text-center text-xs text-muted-foreground">
              {t('tluclaw.sidebar.no_sessions')}
            </div>
          ) : (
            sessions.map((sess) => {
              const isActive = sess.id === activeSessionId;
              return (
                <div
                  key={sess.id}
                  onClick={() => {
                    setSelectedSessionId(sess.id);
                    setOptimisticMessages([]);
                  }}
                  className={`group flex items-center justify-between rounded-lg px-3 py-2.5 text-xs cursor-pointer transition-colors ${
                    isActive
                      ? 'bg-primary/15 text-primary font-semibold'
                      : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
                  }`}
                >
                  <div className="truncate flex-1 pr-2">
                    <p className="truncate text-xs">{sess.title}</p>
                    <span className="text-[10px] text-muted-foreground/80 font-normal">
                      {new Date(sess.updated_at).toLocaleDateString()}
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      setSessionToDelete(sess.id);
                    }}
                    title={t('tluclaw.sidebar.delete_session')}
                    aria-label={t('tluclaw.sidebar.delete_session')}
                    className="opacity-0 group-hover:opacity-100 p-1 text-muted-foreground hover:text-destructive rounded transition-all cursor-pointer"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              );
            })
          )}
        </div>
      </div>

      {/* Main Workspace */}
      <div className="flex-1 flex flex-col h-full bg-background relative overflow-hidden min-w-0">
        {/* Workspace Top Header */}
        <div className="border-b border-border bg-card/40 px-5 py-3 flex flex-wrap items-center justify-between gap-3 shrink-0">
          <div>
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-lg bg-white p-1 border border-slate-200/80 shadow-xs flex items-center justify-center shrink-0">
                <img src="/logo.png" alt="TLU Emblem" className="w-full h-full object-contain" />
              </div>
              <h1 className="text-base font-bold text-foreground">
                {t('tluclaw.header.agent_title')}
              </h1>
              <span className="rounded-full bg-emerald-500/10 border border-emerald-500/30 px-2 py-0.5 text-[11px] font-semibold text-emerald-600 dark:text-emerald-400">
                {t('tluclaw.header.active_status')}
              </span>
            </div>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t('tluclaw.header.agent_description')}
            </p>
          </div>

          <div className="flex items-center gap-2">
            <div className="flex items-center rounded-lg border border-border bg-muted/40 p-0.5">
              <button
                type="button"
                onClick={() => setActiveTab('chat')}
                className={`flex items-center gap-1.5 px-3 py-1 text-xs font-semibold rounded-md transition-all cursor-pointer ${
                  activeTab === 'chat'
                    ? 'bg-tlu-navy text-white dark:bg-tlu-blue shadow-xs'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                <Bot className="h-3.5 w-3.5" />
                <span>{t('tluclaw.tabs.chat')}</span>
              </button>
              <button
                type="button"
                onClick={() => setActiveTab('timeline')}
                className={`flex items-center gap-1.5 px-3 py-1 text-xs font-semibold rounded-md transition-all cursor-pointer ${
                  activeTab === 'timeline'
                    ? 'bg-tlu-navy text-white dark:bg-tlu-blue shadow-xs'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                <History className="h-3.5 w-3.5" />
                <span>{t('tluclaw.tabs.timeline')}</span>
              </button>
            </div>

            <RefreshButton
              onClick={() => {
                queryClient.invalidateQueries({ queryKey: ['tluclawSessions'] });
                if (activeSessionId) {
                  queryClient.invalidateQueries({ queryKey: ['tluclawSession', activeSessionId] });
                }
                if (activeTab === 'timeline') {
                  queryClient.invalidateQueries({ queryKey: ['tluclawSnapshots'] });
                }
              }}
              isLoading={loadingSnapshots}
            />
          </div>
        </div>
        {activeTab === 'chat' ? (
          <>
            {/* Message Stream */}
            <div className="flex-1 overflow-y-auto p-4 md:p-6 space-y-6">
          {displayedMessages.length === 0 ? (
            <div className="mx-auto max-w-2xl py-10 text-center">
              <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/10 text-primary shadow-inner">
                <Sparkles className="h-7 w-7" />
              </div>
              <h3 className="text-lg font-bold text-foreground">
                {t('tluclaw.welcome.title')}
              </h3>
              <p className="mt-2 text-xs text-muted-foreground leading-relaxed max-w-md mx-auto">
                {t('tluclaw.welcome.description')}
              </p>

              {/* Quick Prompt Cards */}
              <div className="mt-8 grid grid-cols-1 sm:grid-cols-2 gap-3 text-left">
                {[
                  {
                    key: 'tluclaw.prompts.cohort_btn',
                    desc: 'tluclaw.prompts.cohort_desc',
                    prompt: 'tluclaw.prompts.cohort_text',
                  },
                  {
                    key: 'tluclaw.prompts.curriculum_btn',
                    desc: 'tluclaw.prompts.curriculum_desc',
                    prompt: 'tluclaw.prompts.curriculum_text',
                  },
                  {
                    key: 'tluclaw.prompts.supersede_btn',
                    desc: 'tluclaw.prompts.supersede_desc',
                    prompt: 'tluclaw.prompts.supersede_text',
                  },
                  {
                    key: 'tluclaw.prompts.rollback_btn',
                    desc: 'tluclaw.prompts.rollback_desc',
                    prompt: 'tluclaw.prompts.rollback_text',
                  },
                ].map((item, idx) => (
                  <button
                    key={idx}
                    type="button"
                    onClick={() => handleQuickPrompt(item.prompt)}
                    className="flex flex-col rounded-xl border border-border/80 bg-card/60 p-3.5 hover:border-primary/50 hover:bg-muted/40 transition-all text-left shadow-xs"
                  >
                    <span className="text-xs font-semibold text-foreground flex items-center justify-between">
                      {t(item.key)}
                      <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                    </span>
                    <span className="mt-1 text-[11px] text-muted-foreground">
                      {t(item.desc)}
                    </span>
                  </button>
                ))}
              </div>
            </div>
          ) : (
            displayedMessages.map((msg) => (
              <div
                key={msg.id}
                className={`flex gap-3 max-w-4xl ${
                  msg.role === 'user' ? 'ml-auto flex-row-reverse' : 'mr-auto'
                }`}
              >
                <div
                  className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg shadow-2xs ${
                    msg.role === 'user'
                      ? 'bg-tlu-navy text-white'
                      : 'bg-white border border-slate-200/80 p-1'
                  }`}
                >
                  {msg.role === 'user' ? (
                    <UserIcon className="h-4 w-4" />
                  ) : (
                    <img src="/logo.png" alt="TLUClaw Assistant" className="w-full h-full object-contain" />
                  )}
                </div>

                <div className="flex flex-col gap-1.5 max-w-[85%]">
                  {msg.role !== 'user' && msg.did_compact && (
                    <div className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-amber-500/10 border border-amber-500/20 text-xs text-amber-700 dark:text-amber-300 w-fit">
                      <Zap className="w-3.5 h-3.5 text-amber-500 shrink-0" />
                      <span className="font-medium">{t('tluclaw.chat.context_compacted_notice')}</span>
                    </div>
                  )}
                  <div
                    className={`rounded-2xl px-4 py-3 text-sm leading-relaxed shadow-xs ${
                      msg.role === 'user'
                        ? 'bg-tlu-blue text-white rounded-tr-none'
                        : 'bg-card border border-border text-foreground rounded-tl-none'
                    }`}
                  >
                    <p className="whitespace-pre-wrap">{msg.content}</p>

                    {/* Staged Preview Card */}
                    {msg.preview_card && (
                      <TLUClawPreviewCard
                        card={msg.preview_card}
                        onConfirm={handleConfirmAction}
                        onCancel={handleCancelAction}
                        onRefinePrompt={(txt) => setInputPrompt(txt)}
                      />
                    )}
                  </div>
                  <span
                    className={`text-[10px] text-muted-foreground px-1 ${
                      msg.role === 'user' ? 'text-right' : 'text-left'
                    }`}
                  >
                    {new Date(msg.created_at).toLocaleTimeString([], {
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </span>
                </div>
              </div>
            ))
          )}

          {/* Assistant Loading Stepper */}
          {loading && (
            <div className="flex gap-3 max-w-2xl mr-auto animate-in fade-in slide-in-from-bottom-2 duration-300">
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-white border border-tlu-blue/30 dark:border-tlu-cyan/30 p-1 shadow-xs ring-2 ring-tlu-blue/10 dark:ring-tlu-cyan/10">
                <img src="/logo.png" alt="TLUClaw Loading" className="w-full h-full object-contain animate-pulse" />
              </div>
              <div className="rounded-2xl rounded-tl-none border border-border bg-card p-4 shadow-xs w-full space-y-3">
                <div className="flex items-center justify-between gap-3">
                  <div className="flex items-center gap-2 text-xs font-semibold text-tlu-blue dark:text-tlu-cyan">
                    <div className="relative flex items-center justify-center">
                      <Sparkles className="h-4 w-4 animate-pulse shrink-0" />
                      <span className="absolute -inset-1 rounded-full bg-tlu-blue/20 dark:bg-tlu-cyan/20 blur-xs animate-ping opacity-75" />
                    </div>
                    <span>{t('tluclaw.stepper.processing_title')}</span>
                  </div>
                  <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-tlu-blue/10 dark:bg-tlu-cyan/10 border border-tlu-blue/15 dark:border-tlu-cyan/20">
                    <span className="h-1.5 w-1.5 rounded-full bg-tlu-blue dark:bg-tlu-cyan animate-bounce [animation-delay:-0.32s]" />
                    <span className="h-1.5 w-1.5 rounded-full bg-tlu-blue dark:bg-tlu-cyan animate-bounce [animation-delay:-0.16s]" />
                    <span className="h-1.5 w-1.5 rounded-full bg-tlu-blue dark:bg-tlu-cyan animate-bounce" />
                  </div>
                </div>

                <div className="relative h-1 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800">
                  <div className="absolute inset-y-0 w-2/5 rounded-full bg-linear-to-r from-transparent via-tlu-blue dark:via-tlu-cyan to-transparent animate-pulse" />
                </div>

                <div className="flex flex-wrap items-center gap-2 text-[11px]">
                  <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md bg-tlu-blue/15 text-tlu-blue dark:bg-tlu-cyan/15 dark:text-tlu-cyan font-medium border border-tlu-blue/20 dark:border-tlu-cyan/20 shadow-2xs">
                    <Layers className="h-3 w-3 animate-pulse" />
                    {t('tluclaw.stepper.step_ocr')}
                  </span>
                  <span className="text-muted-foreground/40">&rarr;</span>
                  <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-muted-foreground bg-muted/40 border border-transparent">
                    <ShieldAlert className="h-3 w-3" />
                    {t('tluclaw.stepper.step_diff')}
                  </span>
                  <span className="text-muted-foreground/40">&rarr;</span>
                  <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-muted-foreground bg-muted/40 border border-transparent">
                    <Eye className="h-3 w-3" />
                    {t('tluclaw.stepper.step_preview')}
                  </span>
                </div>
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* Input Bar */}
        <div className="border-t border-border bg-card/60 p-4 shrink-0">
          <div className="mx-auto max-w-4xl">
            {/* Attachment preview chips */}
            {(attachedImages.length > 0 || attachedFiles.length > 0) && (
              <div className="mb-2.5 flex flex-wrap gap-2">
                {attachedImages.map((img, idx) => (
                  <div
                    key={idx}
                    className="relative group flex items-center gap-1 rounded-lg border border-border bg-background p-1.5 shadow-xs"
                  >
                    <img
                      src={img}
                      alt="attachment preview"
                      className="h-10 w-10 object-cover rounded"
                    />
                    <button
                      type="button"
                      onClick={() =>
                        setAttachedImages((prev) => prev.filter((_, i) => i !== idx))
                      }
                      className="p-1 rounded-full text-muted-foreground hover:text-destructive hover:bg-muted"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </div>
                ))}

                {attachedFiles.map((file, idx) => (
                  <div
                    key={idx}
                    className="flex items-center gap-1.5 rounded-lg border border-border bg-background px-2.5 py-1.5 text-xs shadow-xs"
                  >
                    <FileText className="h-4 w-4 text-primary" />
                    <span className="font-mono text-xs max-w-48 truncate">
                      {file.file_name}
                    </span>
                    <button
                      type="button"
                      onClick={() =>
                        setAttachedFiles((prev) => prev.filter((_, i) => i !== idx))
                      }
                      className="p-0.5 rounded text-muted-foreground hover:text-destructive"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </div>
                ))}
              </div>
            )}

            {/* Input Row */}
            <div className="flex items-end gap-2 rounded-2xl border border-border bg-background p-2 shadow-xs focus-within:border-primary focus-within:ring-1 focus-within:ring-primary transition-all">
              <div className="flex items-center gap-1 pb-1">
                <button
                  type="button"
                  onClick={() => imageInputRef.current?.click()}
                  className="p-2 rounded-xl text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
                  title={t('tluclaw.input.attach_image_tooltip')}
                >
                  <ImageIcon className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  className="p-2 rounded-xl text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
                  title={t('tluclaw.input.attach_file_tooltip')}
                >
                  <Paperclip className="h-4 w-4" />
                </button>
              </div>

              <textarea
                value={inputPrompt}
                onChange={(e) => setInputPrompt(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    handleSendMessage();
                  }
                }}
                rows={1}
                placeholder={t('tluclaw.input.placeholder')}
                className="flex-1 max-h-32 resize-none bg-transparent py-1.5 text-sm text-foreground focus:outline-none placeholder:text-muted-foreground"
              />

              <button
                type="button"
                onClick={() => handleSendMessage()}
                disabled={
                  loading ||
                  (!inputPrompt.trim() &&
                    attachedImages.length === 0 &&
                    attachedFiles.length === 0)
                }
                className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-tlu-navy hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-[#1541B8] text-white shadow-xs focus:outline-none disabled:opacity-40 transition-colors cursor-pointer"
              >
                {loading ? (
                  <LoadingSpinner className="h-4 w-4" />
                ) : (
                  <Send className="h-4 w-4" />
                )}
              </button>
            </div>
            <div className="mt-1.5 flex items-center justify-between px-2 text-[10px] text-muted-foreground">
              <span>{t('tluclaw.input.hint')}</span>
              <span>TLUClaw v1.0 &bull; Two-Phase Protocol</span>
            </div>
          </div>
        </div>
      </>
    ) : (
      <div className="flex-1 overflow-y-auto p-4 md:p-6 space-y-4">
        <div className="mx-auto max-w-4xl space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 border-b border-border pb-4">
            <div>
              <h3 className="text-base font-bold text-foreground">
                {t('tluclaw.timeline.title')}
              </h3>
              <p className="text-xs text-muted-foreground mt-0.5">
                {t('tluclaw.timeline.desc')}
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-1.5 bg-muted/30 p-1 rounded-lg border border-border">
              {[
                { val: '', labelKey: 'tluclaw.timeline.filter_all' },
                { val: 'COMMITTED', labelKey: 'tluclaw.timeline.filter_committed' },
                { val: 'ROLLED_BACK', labelKey: 'tluclaw.timeline.filter_rolled_back' },
                { val: 'STAGED', labelKey: 'tluclaw.timeline.filter_staged' },
                { val: 'CANCELLED', labelKey: 'tluclaw.timeline.filter_cancelled' },
              ].map((f) => (
                <button
                  key={f.val}
                  type="button"
                  onClick={() => {
                    setTimelineStatus(f.val);
                    setTimelinePage(1);
                  }}
                  className={`px-2.5 py-1 text-xs rounded-md font-medium transition-all ${
                    timelineStatus === f.val
                      ? 'bg-primary text-primary-foreground shadow-xs'
                      : 'text-muted-foreground hover:text-foreground'
                  }`}
                >
                  {t(f.labelKey)}
                </button>
              ))}
            </div>
          </div>

          {loadingSnapshots ? (
            <div className="py-20 text-center">
              <LoadingSpinner className="mx-auto h-8 w-8 text-primary" />
            </div>
          ) : snapshots.length === 0 ? (
            <div className="py-20 text-center rounded-xl border border-dashed border-border bg-card/40">
              <History className="mx-auto h-10 w-10 text-muted-foreground/40 mb-2" />
              <p className="text-sm font-medium text-foreground">
                {t('tluclaw.timeline.empty')}
              </p>
            </div>
          ) : (
            <>
              <div className="space-y-3">
              {snapshots.map((snap) => {
                const isExpanded = expandedDiffBatches.has(snap.batch_id);
                const isRollingBack = rollingBackBatchId === snap.batch_id;

                return (
                  <div
                    key={snap.id}
                    className="rounded-xl border border-border bg-card p-4 shadow-xs space-y-3 transition-all"
                  >
                    <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <span
                          className={`px-2 py-0.5 rounded-full text-[11px] font-semibold border ${
                            snap.status === 'COMMITTED'
                              ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20'
                              : snap.status === 'ROLLED_BACK'
                              ? 'bg-cyan-500/10 text-cyan-700 dark:text-cyan-400 border-cyan-500/20'
                              : snap.status === 'STAGED'
                              ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20'
                              : 'bg-muted text-muted-foreground border-border'
                          }`}
                        >
                          {snap.status}
                        </span>
                        <span className="font-mono text-xs px-2 py-0.5 rounded bg-muted/60 text-foreground border border-border">
                          {snap.action_type}
                        </span>
                        <span className="font-mono text-xs text-muted-foreground">
                          #{snap.batch_id.slice(0, 8)}
                        </span>
                      </div>

                      <div className="flex items-center gap-2 text-xs text-muted-foreground">
                        <span>{new Date(snap.created_at).toLocaleString()}</span>
                      </div>
                    </div>

                    <p className="text-sm text-foreground leading-relaxed">
                      {snap.summary}
                    </p>

                    {snap.blast_radius && (
                      <div className="flex flex-wrap items-center gap-2 pt-1 text-xs text-muted-foreground">
                        {snap.blast_radius.affected_cohorts && snap.blast_radius.affected_cohorts.length > 0 && (
                          <div className="flex items-center gap-1">
                            <span className="font-medium text-foreground">{t('tluclaw.preview.cohort')}:</span>
                            {snap.blast_radius.affected_cohorts.map((c) => (
                              <span key={c} className="rounded bg-primary/10 px-1.5 py-0.5 text-[11px] font-semibold text-primary">
                                {c}
                              </span>
                            ))}
                          </div>
                        )}

                        {snap.blast_radius.affected_majors && snap.blast_radius.affected_majors.length > 0 && (
                          <div className="flex items-center gap-1">
                            <span className="font-medium text-foreground">{t('tluclaw.preview.major')}:</span>
                            {snap.blast_radius.affected_majors.map((m) => (
                              <span key={m} className="rounded bg-muted px-1.5 py-0.5 text-[11px] font-medium text-foreground">
                                {m}
                              </span>
                            ))}
                          </div>
                        )}

                        {snap.blast_radius.risk_level && (
                          <span className="ml-auto rounded-full bg-muted/80 px-2 py-0.5 text-[10px] uppercase font-bold text-muted-foreground">
                            {snap.blast_radius.risk_level}
                          </span>
                        )}
                      </div>
                    )}

                    <div className="flex items-center justify-between pt-2 border-t border-border/60">
                      {snap.diff_items && snap.diff_items.length > 0 ? (
                        <button
                          type="button"
                          onClick={() => toggleDiffExpand(snap.batch_id)}
                          className="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
                        >
                          {isExpanded ? (
                            <>
                              <ChevronDown className="h-3.5 w-3.5" />
                              <span>{t('tluclaw.timeline.hide_diff')}</span>
                            </>
                          ) : (
                            <>
                              <ChevronRight className="h-3.5 w-3.5" />
                              <span>{t('tluclaw.timeline.view_diff')} ({snap.diff_items.length})</span>
                            </>
                          )}
                        </button>
                      ) : (
                        <div />
                      )}

                      {snap.status === 'COMMITTED' && (
                        <button
                          type="button"
                          onClick={() => setConfirmRollbackBatchId(snap.batch_id)}
                          disabled={isRollingBack}
                          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold bg-destructive/10 text-destructive border border-destructive/20 hover:bg-destructive/20 transition-colors disabled:opacity-50"
                        >
                          {isRollingBack ? (
                            <LoadingSpinner className="h-3.5 w-3.5" />
                          ) : (
                            <RotateCcw className="h-3.5 w-3.5" />
                          )}
                          <span>{t('tluclaw.timeline.btn_rollback')}</span>
                        </button>
                      )}
                    </div>

                    {isExpanded && snap.diff_items && snap.diff_items.length > 0 && (
                      <div className="mt-2 overflow-x-auto rounded-lg border border-border bg-muted/20">
                        <table className="w-full text-left text-xs">
                          <thead className="bg-muted/50 border-b border-border text-[11px] text-muted-foreground">
                            <tr>
                              <th className="py-2 px-3">{t('tluclaw.diff.col_action')}</th>
                              <th className="py-2 px-3">{t('tluclaw.diff.col_key')}</th>
                              <th className="py-2 px-3">{t('tluclaw.diff.col_name')}</th>
                              <th className="py-2 px-3">{t('tluclaw.diff.col_change')}</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-border font-mono text-[11px]">
                            {snap.diff_items.map((item, dIdx) => (
                              <tr key={dIdx} className="hover:bg-muted/30">
                                <td className="py-2 px-3">
                                  <span
                                    className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${
                                      item.action === 'CREATE' || item.action === 'UPSERT'
                                        ? 'bg-emerald-500/10 text-emerald-600'
                                        : item.action === 'DELETE'
                                        ? 'bg-rose-500/10 text-rose-600'
                                        : 'bg-amber-500/10 text-amber-600'
                                    }`}
                                  >
                                    {item.action}
                                  </span>
                                </td>
                                <td className="py-2 px-3 font-semibold text-foreground">
                                  {item.key}
                                </td>
                                <td className="py-2 px-3 text-muted-foreground">
                                  {item.name || '-'}
                                </td>
                                <td className="py-2 px-3">
                                  {item.old_value && (
                                    <span className="text-rose-500 line-through mr-1">
                                      {item.old_value}
                                    </span>
                                  )}
                                  {item.new_value && (
                                    <span className="text-emerald-600 font-semibold">
                                      {item.new_value}
                                    </span>
                                  )}
                                  {!item.old_value && !item.new_value && '-'}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>

            {snapshots.length > 0 && (
              <div className="flex items-center justify-between pt-4 border-t border-border">
                <button
                  type="button"
                  disabled={timelinePage <= 1 || loadingSnapshots}
                  onClick={() => setTimelinePage((p) => Math.max(1, p - 1))}
                  className="px-3 py-1.5 text-xs rounded-lg border border-border bg-card hover:bg-muted text-foreground disabled:opacity-40 transition-colors cursor-pointer disabled:cursor-not-allowed"
                >
                  {t('common.previous')}
                </button>
                <span className="text-xs text-muted-foreground font-mono">
                  {t('common.page_compact', {
                    current: timelinePage,
                    total: snapshots.length < timelineLimit && timelinePage === 1 ? 1 : '...',
                  })}
                </span>
                <button
                  type="button"
                  disabled={snapshots.length < timelineLimit || loadingSnapshots}
                  onClick={() => setTimelinePage((p) => p + 1)}
                  className="px-3 py-1.5 text-xs rounded-lg border border-border bg-card hover:bg-muted text-foreground disabled:opacity-40 transition-colors cursor-pointer disabled:cursor-not-allowed"
                >
                  {t('common.next')}
                </button>
              </div>
            )}
          </>
        )}
        </div>
      </div>
    )}
      </div>

      <ConfirmModal
        isOpen={Boolean(confirmRollbackBatchId)}
        onClose={() => setConfirmRollbackBatchId(null)}
        onConfirm={() => {
          if (confirmRollbackBatchId) {
            handleRollbackBatch(confirmRollbackBatchId);
          }
        }}
        title={t('tluclaw.timeline.btn_rollback')}
        message={
          confirmRollbackBatchId
            ? t('tluclaw.timeline.confirm_rollback', { batchId: confirmRollbackBatchId })
            : ''
        }
        variant="danger"
        isLoading={Boolean(rollingBackBatchId)}
      />

      <ConfirmModal
        isOpen={Boolean(sessionToDelete)}
        onClose={() => setSessionToDelete(null)}
        onConfirm={confirmDeleteSession}
        title={t('tluclaw.sidebar.delete_session_title')}
        message={t('tluclaw.sidebar.delete_session_confirm')}
        variant="danger"
        confirmText={t('common.delete')}
        cancelText={t('common.cancel')}
        isLoading={isDeletingSession}
      />
    </div>
  );
};
