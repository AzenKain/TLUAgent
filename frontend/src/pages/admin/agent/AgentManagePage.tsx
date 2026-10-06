import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Bot,
  ShieldCheck,
  Boxes,
  Eye,
  Save,
  RotateCcw,
  Plus,
  Edit2,
  Trash2,
  Check,
  Copy,
  Search,
  AlertTriangle,
  Sparkles,
  ToggleLeft,
  ToggleRight,
  Sliders,
} from 'lucide-react';
import {
  agentService,
} from '@/services/agentService';
import type {
  AgentPromptDTO,
  AgentSkillDTO,
  CreateAgentSkillDTO,
  UpdateAgentSkillDTO,
  CompactorSettingsDTO,
} from '@/types/agent';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { Badge } from '@/components/common/Badge';
import { Modal } from '@/components/common/Modal';

export const AgentManagePage: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [activeTab, setActiveTab] = useState<'soul' | 'rules' | 'skills' | 'preview' | 'compactor'>('soul');

  const [toastMessage, setToastMessage] = useState<string | null>(null);
  const [toastType, setToastType] = useState<'success' | 'error'>('success');

  const showToast = (message: string, type: 'success' | 'error' = 'success') => {
    setToastMessage(message);
    setToastType(type);
    setTimeout(() => setToastMessage(null), 3500);
  };

  const { data: compactorSettings, isLoading: compactorLoading } = useQuery<CompactorSettingsDTO>({
    queryKey: ['agent-compactor-settings'],
    queryFn: () => agentService.getCompactorSettings(),
  });

  const [compactorForm, setCompactorForm] = useState<CompactorSettingsDTO>({
    max_context_tokens: 250000,
    compact_threshold_ratio: 0.85,
    keep_recent_turns: 4,
  });
  const [compactorInitialized, setCompactorInitialized] = useState(false);

  if (compactorSettings && !compactorInitialized) {
    setCompactorForm(compactorSettings);
    setCompactorInitialized(true);
  }

  const updateCompactorMutation = useMutation({
    mutationFn: (data: CompactorSettingsDTO) => agentService.updateCompactorSettings(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agent-compactor-settings'] });
      showToast(t('agent_manage.compactor_save_success'), 'success');
    },
    onError: (err: Error) => {
      showToast(err.message, 'error');
    },
  });

  const { data: prompts, isLoading: promptsLoading } = useQuery<AgentPromptDTO[]>({
    queryKey: ['agent-prompts'],
    queryFn: () => agentService.getPrompts(),
  });

  const { data: skills, isLoading: skillsLoading } = useQuery<AgentSkillDTO[]>({
    queryKey: ['agent-skills'],
    queryFn: () => agentService.getSkills(),
  });

  const soulPrompt = prompts?.find((p) => p.id === 'soul');
  const rulePrompt = prompts?.find((p) => p.id === 'rule');

  const [soulContent, setSoulContent] = useState<string>('');
  const [soulTitle, setSoulTitle] = useState<string>('');
  const [soulInitialized, setSoulInitialized] = useState(false);

  if (soulPrompt && !soulInitialized) {
    setSoulContent(soulPrompt.content);
    setSoulTitle(soulPrompt.title);
    setSoulInitialized(true);
  }

  const [ruleContent, setRuleContent] = useState<string>('');
  const [ruleTitle, setRuleTitle] = useState<string>('');
  const [ruleInitialized, setRuleInitialized] = useState(false);

  if (rulePrompt && !ruleInitialized) {
    setRuleContent(rulePrompt.content);
    setRuleTitle(rulePrompt.title);
    setRuleInitialized(true);
  }

  const updatePromptMutation = useMutation({
    mutationFn: ({ id, title, content }: { id: string; title: string; content: string }) =>
      agentService.updatePrompt(id, { title, content }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agent-prompts'] });
      showToast(t('agent_manage.save_success'), 'success');
    },
    onError: (err: Error) => {
      showToast(err.message, 'error');
    },
  });

  const resetPromptMutation = useMutation({
    mutationFn: (id: string) => agentService.resetPrompt(id),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['agent-prompts'] });
      if (data.id === 'soul') {
        setSoulContent(data.content);
        setSoulTitle(data.title);
      } else if (data.id === 'rule') {
        setRuleContent(data.content);
        setRuleTitle(data.title);
      }
      showToast(t('agent_manage.reset_success'), 'success');
      setResetConfirmModal(null);
    },
    onError: (err: Error) => {
      showToast(err.message, 'error');
    },
  });

  const [skillSearch, setSkillSearch] = useState('');
  const [skillModalOpen, setSkillModalOpen] = useState(false);
  const [editingSkill, setEditingSkill] = useState<AgentSkillDTO | null>(null);
  const [skillForm, setSkillForm] = useState({
    id: '',
    name: '',
    description: '',
    content: '',
    tool_definition: '',
    priority: 0,
    is_enabled: true,
  });

  const [deleteConfirmModal, setDeleteConfirmModal] = useState<string | null>(null);
  const [resetConfirmModal, setResetConfirmModal] = useState<{ type: 'prompt' | 'skill'; id: string } | null>(null);

  const createSkillMutation = useMutation({
    mutationFn: (data: CreateAgentSkillDTO) => agentService.createSkill(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agent-skills'] });
      showToast(t('agent_manage.create_skill_success'), 'success');
      setSkillModalOpen(false);
    },
    onError: (err: Error) => {
      showToast(err.message, 'error');
    },
  });

  const updateSkillMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateAgentSkillDTO }) =>
      agentService.updateSkill(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agent-skills'] });
      showToast(t('agent_manage.update_skill_success'), 'success');
      setSkillModalOpen(false);
    },
    onError: (err: Error) => {
      showToast(err.message, 'error');
    },
  });

  const toggleSkillMutation = useMutation({
    mutationFn: ({ id, is_enabled }: { id: string; is_enabled: boolean }) =>
      agentService.toggleSkill(id, is_enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agent-skills'] });
      showToast(t('agent_manage.toggle_skill_success'), 'success');
    },
    onError: (err: Error) => {
      showToast(err.message, 'error');
    },
  });

  const deleteSkillMutation = useMutation({
    mutationFn: (id: string) => agentService.deleteSkill(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agent-skills'] });
      showToast(t('agent_manage.delete_skill_success'), 'success');
      setDeleteConfirmModal(null);
    },
    onError: (err: Error) => {
      showToast(err.message, 'error');
    },
  });

  const resetSkillMutation = useMutation({
    mutationFn: (id: string) => agentService.resetSkill(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agent-skills'] });
      showToast(t('agent_manage.reset_success'), 'success');
      setResetConfirmModal(null);
    },
    onError: (err: Error) => {
      showToast(err.message, 'error');
    },
  });

  const [previewCohort, setPreviewCohort] = useState('K36');
  const [previewSampleRAG, setPreviewSampleRAG] = useState(true);
  const [copiedPrompt, setCopiedPrompt] = useState(false);

  const { data: previewPromptText, isLoading: previewLoading, refetch: refetchPreview } = useQuery<string>({
    queryKey: ['agent-preview', previewCohort, previewSampleRAG],
    queryFn: () =>
      agentService.previewPrompt({
        student_cohort: previewCohort,
        include_sample_rag: previewSampleRAG,
      }),
    enabled: activeTab === 'preview',
  });

  const handleOpenAddSkill = () => {
    setEditingSkill(null);
    setSkillForm({
      id: '',
      name: '',
      description: '',
      content: '',
      tool_definition: '',
      priority: 50,
      is_enabled: true,
    });
    setSkillModalOpen(true);
  };

  const handleOpenEditSkill = (skill: AgentSkillDTO) => {
    setEditingSkill(skill);
    setSkillForm({
      id: skill.id,
      name: skill.name,
      description: skill.description,
      content: skill.content,
      tool_definition: skill.tool_definition || '',
      priority: skill.priority,
      is_enabled: skill.is_enabled,
    });
    setSkillModalOpen(true);
  };

  const handleSaveSkill = (e: React.FormEvent) => {
    e.preventDefault();
    if (!skillForm.name.trim() || !skillForm.content.trim()) {
      showToast(t('agent_manage.error_empty_content'), 'error');
      return;
    }

    if (editingSkill) {
      updateSkillMutation.mutate({
        id: editingSkill.id,
        data: {
          name: skillForm.name,
          description: skillForm.description,
          content: skillForm.content,
          tool_definition: skillForm.tool_definition,
          priority: skillForm.priority,
        },
      });
    } else {
      if (!skillForm.id.trim()) {
        showToast(t('agent_manage.error_empty_title'), 'error');
        return;
      }
      createSkillMutation.mutate({
        id: skillForm.id,
        name: skillForm.name,
        description: skillForm.description,
        content: skillForm.content,
        tool_definition: skillForm.tool_definition,
        priority: skillForm.priority,
        is_enabled: skillForm.is_enabled,
      });
    }
  };

  const handleCopyPrompt = () => {
    if (previewPromptText) {
      navigator.clipboard.writeText(previewPromptText);
      setCopiedPrompt(true);
      showToast(t('agent_manage.prompt_copied'), 'success');
      setTimeout(() => setCopiedPrompt(false), 2000);
    }
  };

  const filteredSkills = skills?.filter(
    (s) =>
      s.name.toLowerCase().includes(skillSearch.toLowerCase()) ||
      s.id.toLowerCase().includes(skillSearch.toLowerCase()) ||
      s.description.toLowerCase().includes(skillSearch.toLowerCase())
  );

  return (
    <div className="space-y-6">
      {/* Toast Notification */}
      {toastMessage && (
        <div
          role="status"
          aria-live="polite"
          className={`fixed bottom-5 right-5 z-50 flex items-center gap-2 rounded-xl px-4 py-3 text-sm font-medium shadow-xl transition-all ${
            toastType === 'success'
              ? 'bg-emerald-600 text-white shadow-emerald-500/20'
              : 'bg-rose-600 text-white shadow-rose-500/20'
          }`}
        >
          {toastType === 'success' ? <Check className="h-4 w-4" /> : <AlertTriangle className="h-4 w-4" />}
          <span>{toastMessage}</span>
        </div>
      )}

      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="flex items-center gap-2.5 text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            <Bot className="h-7 w-7 text-indigo-600 dark:text-indigo-400" />
            {t('agent_manage.page_title')}
          </h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            {t('agent_manage.page_subtitle')}
          </p>
        </div>
      </div>

      {/* Tab Navigation */}
      <div className="flex border-b border-slate-200 dark:border-slate-800">
        <button
          onClick={() => setActiveTab('soul')}
          className={`flex items-center gap-2 border-b-2 px-5 py-3 text-sm font-semibold transition-colors ${
            activeTab === 'soul'
              ? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
              : 'border-transparent text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'
          }`}
        >
          <Sparkles className="h-4 w-4" />
          {t('agent_manage.tab_soul')}
        </button>
        <button
          onClick={() => setActiveTab('rules')}
          className={`flex items-center gap-2 border-b-2 px-5 py-3 text-sm font-semibold transition-colors ${
            activeTab === 'rules'
              ? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
              : 'border-transparent text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'
          }`}
        >
          <ShieldCheck className="h-4 w-4" />
          {t('agent_manage.tab_rules')}
        </button>
        <button
          onClick={() => setActiveTab('skills')}
          className={`flex items-center gap-2 border-b-2 px-5 py-3 text-sm font-semibold transition-colors ${
            activeTab === 'skills'
              ? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
              : 'border-transparent text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'
          }`}
        >
          <Boxes className="h-4 w-4" />
          {t('agent_manage.tab_skills')}
          {skills && (
            <span className="ml-1 rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-800 dark:text-slate-300">
              {skills.length}
            </span>
          )}
        </button>
        <button
          onClick={() => setActiveTab('preview')}
          className={`flex items-center gap-2 border-b-2 px-5 py-3 text-sm font-semibold transition-colors ${
            activeTab === 'preview'
              ? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
              : 'border-transparent text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'
          }`}
        >
          <Eye className="h-4 w-4" />
          {t('agent_manage.tab_preview')}
        </button>
        <button
          onClick={() => setActiveTab('compactor')}
          className={`flex items-center gap-2 border-b-2 px-5 py-3 text-sm font-semibold transition-colors ${
            activeTab === 'compactor'
              ? 'border-indigo-600 text-indigo-600 dark:border-indigo-400 dark:text-indigo-400'
              : 'border-transparent text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'
          }`}
        >
          <Sliders className="h-4 w-4" />
          {t('agent_manage.tab_compactor')}
        </button>
      </div>

      {/* Tab 1: Soul */}
      {activeTab === 'soul' && (
        <div className="space-y-4">
          <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900">
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 className="text-lg font-bold text-slate-900 dark:text-slate-100">
                  {t('agent_manage.soul_title')}
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  {t('agent_manage.soul_desc')}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setResetConfirmModal({ type: 'prompt', id: 'soul' })}
                  className="flex items-center gap-1.5 rounded-xl border border-slate-300 px-3.5 py-2 text-xs font-semibold text-slate-700 transition hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                >
                  <RotateCcw className="h-3.5 w-3.5" />
                  {t('agent_manage.btn_reset_default')}
                </button>
                <button
                  type="button"
                  disabled={updatePromptMutation.isPending}
                  onClick={() =>
                    updatePromptMutation.mutate({
                      id: 'soul',
                      title: soulTitle || soulPrompt?.title || 'Soul',
                      content: soulContent,
                    })
                  }
                  className="flex items-center gap-1.5 rounded-xl bg-indigo-600 px-4 py-2 text-xs font-semibold text-white shadow-sm transition hover:bg-indigo-700 disabled:opacity-50"
                >
                  <Save className="h-3.5 w-3.5" />
                  {updatePromptMutation.isPending ? t('common.saving') : t('agent_manage.btn_save_prompt')}
                </button>
              </div>
            </div>

            {promptsLoading ? (
              <div className="flex justify-center py-12">
                <LoadingSpinner size="lg" />
              </div>
            ) : (
              <div className="mt-4 space-y-4">
                <div>
                  <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
                    {t('agent_manage.skill_name')}
                  </label>
                  <input
                    type="text"
                    value={soulTitle}
                    onChange={(e) => setSoulTitle(e.target.value)}
                    className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2 text-sm text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
                    {t('agent_manage.skill_content')}
                  </label>
                  <textarea
                    rows={18}
                    value={soulContent}
                    onChange={(e) => setSoulContent(e.target.value)}
                    className="font-mono w-full rounded-xl border border-slate-300 bg-slate-50 p-4 text-xs text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-200"
                  />
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Tab 2: Rules */}
      {activeTab === 'rules' && (
        <div className="space-y-4">
          <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900">
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 className="text-lg font-bold text-slate-900 dark:text-slate-100">
                  {t('agent_manage.rule_title')}
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  {t('agent_manage.rule_desc')}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setResetConfirmModal({ type: 'prompt', id: 'rule' })}
                  className="flex items-center gap-1.5 rounded-xl border border-slate-300 px-3.5 py-2 text-xs font-semibold text-slate-700 transition hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                >
                  <RotateCcw className="h-3.5 w-3.5" />
                  {t('agent_manage.btn_reset_default')}
                </button>
                <button
                  type="button"
                  disabled={updatePromptMutation.isPending}
                  onClick={() =>
                    updatePromptMutation.mutate({
                      id: 'rule',
                      title: ruleTitle || rulePrompt?.title || 'Rules',
                      content: ruleContent,
                    })
                  }
                  className="flex items-center gap-1.5 rounded-xl bg-indigo-600 px-4 py-2 text-xs font-semibold text-white shadow-sm transition hover:bg-indigo-700 disabled:opacity-50"
                >
                  <Save className="h-3.5 w-3.5" />
                  {updatePromptMutation.isPending ? t('common.saving') : t('agent_manage.btn_save_prompt')}
                </button>
              </div>
            </div>

            {promptsLoading ? (
              <div className="flex justify-center py-12">
                <LoadingSpinner size="lg" />
              </div>
            ) : (
              <div className="mt-4 space-y-4">
                <div>
                  <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
                    {t('agent_manage.skill_name')}
                  </label>
                  <input
                    type="text"
                    value={ruleTitle}
                    onChange={(e) => setRuleTitle(e.target.value)}
                    className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2 text-sm text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
                    {t('agent_manage.skill_content')}
                  </label>
                  <textarea
                    rows={18}
                    value={ruleContent}
                    onChange={(e) => setRuleContent(e.target.value)}
                    className="font-mono w-full rounded-xl border border-slate-300 bg-slate-50 p-4 text-xs text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-200"
                  />
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Tab 3: Skills */}
      {activeTab === 'skills' && (
        <div className="space-y-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="relative max-w-sm flex-1">
              <Search className="absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                value={skillSearch}
                onChange={(e) => setSkillSearch(e.target.value)}
                placeholder={t('agent_manage.filter_skills_placeholder')}
                className="w-full rounded-xl border border-slate-300 bg-white py-2 pl-9 pr-4 text-xs text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
              />
            </div>
            <button
              type="button"
              onClick={handleOpenAddSkill}
              className="flex items-center gap-1.5 rounded-xl bg-indigo-600 px-4 py-2 text-xs font-semibold text-white shadow-sm transition hover:bg-indigo-700"
            >
              <Plus className="h-4 w-4" />
              {t('agent_manage.btn_add_skill')}
            </button>
          </div>

          {skillsLoading ? (
            <div className="flex justify-center py-12">
              <LoadingSpinner size="lg" />
            </div>
          ) : !filteredSkills || filteredSkills.length === 0 ? (
            <div className="rounded-2xl border border-slate-200 bg-white p-8 text-center text-sm text-slate-500 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-400">
              {t('agent_manage.no_skills_found')}
            </div>
          ) : (
            <div className="grid gap-4 sm:grid-cols-1 md:grid-cols-2">
              {filteredSkills.map((skill) => (
                <div
                  key={skill.id}
                  className="flex flex-col justify-between rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-slate-300 dark:border-slate-800 dark:bg-slate-900 dark:hover:border-slate-700"
                >
                  <div className="space-y-3">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="flex items-center gap-2">
                          <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">
                            {skill.name}
                          </h3>
                          <Badge variant={skill.is_enabled ? 'success' : 'secondary'}>
                            {skill.is_enabled ? t('agent_manage.skill_enabled') : t('agent_manage.skill_disabled')}
                          </Badge>
                        </div>
                        <p className="font-mono mt-0.5 text-xs text-indigo-600 dark:text-indigo-400">
                          {skill.id}
                        </p>
                      </div>

                      <button
                        type="button"
                        onClick={() => toggleSkillMutation.mutate({ id: skill.id, is_enabled: !skill.is_enabled })}
                        className="text-slate-500 transition hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100"
                        title={skill.is_enabled ? t('agent_manage.skill_disabled') : t('agent_manage.skill_enabled')}
                      >
                        {skill.is_enabled ? (
                          <ToggleRight className="h-7 w-7 text-indigo-600 dark:text-indigo-400" />
                        ) : (
                          <ToggleLeft className="h-7 w-7 text-slate-400" />
                        )}
                      </button>
                    </div>

                    <p className="text-xs text-slate-600 line-clamp-2 dark:text-slate-300">
                      {skill.description}
                    </p>

                    <div className="flex flex-wrap items-center gap-2 pt-1 text-[11px] text-slate-500 dark:text-slate-400">
                      <span className="flex items-center gap-1 rounded-md bg-slate-100 px-2 py-0.5 dark:bg-slate-800">
                        <Sliders className="h-3 w-3" />
                        {t('agent_manage.skill_priority')}: {skill.priority}
                      </span>
                      {skill.tool_definition && (
                        <span className="rounded-md bg-indigo-50 px-2 py-0.5 text-indigo-600 dark:bg-indigo-950/50 dark:text-indigo-400">
                          Tool Call
                        </span>
                      )}
                    </div>
                  </div>

                  <div className="mt-4 flex items-center justify-end gap-2 border-t border-slate-100 pt-3 dark:border-slate-800">
                    <button
                      type="button"
                      onClick={() => setResetConfirmModal({ type: 'skill', id: skill.id })}
                      className="rounded-lg p-1.5 text-slate-500 transition hover:bg-slate-100 hover:text-slate-700 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-slate-200"
                      title={t('agent_manage.btn_reset_default')}
                    >
                      <RotateCcw className="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      onClick={() => handleOpenEditSkill(skill)}
                      className="rounded-lg p-1.5 text-indigo-600 transition hover:bg-indigo-50 dark:text-indigo-400 dark:hover:bg-indigo-950/40"
                      title={t('agent_manage.btn_edit_skill')}
                    >
                      <Edit2 className="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      onClick={() => setDeleteConfirmModal(skill.id)}
                      className="rounded-lg p-1.5 text-rose-500 transition hover:bg-rose-50 dark:hover:bg-rose-950/40"
                      title={t('agent_manage.btn_delete_skill')}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Tab 4: Preview */}
      {activeTab === 'preview' && (
        <div className="space-y-4">
          <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex flex-wrap items-center gap-4">
                <div>
                  <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
                    {t('agent_manage.preview_cohort_label')}
                  </label>
                  <input
                    type="text"
                    value={previewCohort}
                    onChange={(e) => setPreviewCohort(e.target.value)}
                    placeholder={t('agent_manage.preview_cohort_placeholder')}
                    className="w-40 rounded-xl border border-slate-300 bg-white px-3 py-1.5 text-xs text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
                  />
                </div>
                <div className="flex items-center gap-2 pt-5">
                  <input
                    type="checkbox"
                    id="sample-rag"
                    checked={previewSampleRAG}
                    onChange={(e) => setPreviewSampleRAG(e.target.checked)}
                    className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                  />
                  <label htmlFor="sample-rag" className="text-xs text-slate-700 dark:text-slate-300">
                    {t('agent_manage.preview_sample_rag_label')}
                  </label>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => refetchPreview()}
                  className="flex items-center gap-1.5 rounded-xl border border-slate-300 px-3.5 py-2 text-xs font-semibold text-slate-700 transition hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                >
                  <RotateCcw className="h-3.5 w-3.5" />
                  {t('agent_manage.btn_preview_refresh')}
                </button>
                <button
                  type="button"
                  onClick={handleCopyPrompt}
                  className="flex items-center gap-1.5 rounded-xl bg-indigo-600 px-4 py-2 text-xs font-semibold text-white shadow-sm transition hover:bg-indigo-700"
                >
                  {copiedPrompt ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                  {t('agent_manage.btn_copy_prompt')}
                </button>
              </div>
            </div>

            {previewLoading ? (
              <div className="flex justify-center py-12">
                <LoadingSpinner size="lg" />
              </div>
            ) : (
              <div className="mt-4">
                <div className="mb-2 flex items-center justify-between text-xs text-slate-500 dark:text-slate-400">
                  <span>System Prompt ({t('agent_manage.preview_char_count', { count: previewPromptText?.length || 0 })})</span>
                </div>
                <pre className="font-mono max-h-[600px] overflow-y-auto whitespace-pre-wrap rounded-xl border border-slate-200 bg-slate-50 p-4 text-xs leading-relaxed text-slate-900 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-200">
                  {previewPromptText}
                </pre>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Tab 5: Compactor Settings */}
      {activeTab === 'compactor' && (
        <div className="space-y-4">
          <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900">
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 className="text-lg font-bold text-slate-900 dark:text-slate-100">
                  {t('agent_manage.compactor_title')}
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  {t('agent_manage.compactor_desc')}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  disabled={updateCompactorMutation.isPending}
                  onClick={() => updateCompactorMutation.mutate(compactorForm)}
                  className="flex items-center gap-1.5 rounded-xl bg-indigo-600 px-4 py-2 text-xs font-semibold text-white shadow-sm transition hover:bg-indigo-700 disabled:opacity-50"
                >
                  <Save className="h-3.5 w-3.5" />
                  {updateCompactorMutation.isPending ? t('common.saving') : t('agent_manage.compactor_save_btn')}
                </button>
              </div>
            </div>

            {compactorLoading ? (
              <div className="flex justify-center py-12">
                <LoadingSpinner size="lg" />
              </div>
            ) : (
              <div className="mt-6 max-w-xl space-y-6">
                <div>
                  <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
                    {t('agent_manage.compactor_max_tokens')}
                  </label>
                  <input
                    type="number"
                    min={1000}
                    max={2000000}
                    step={1000}
                    value={compactorForm.max_context_tokens}
                    onChange={(e) =>
                      setCompactorForm({
                        ...compactorForm,
                        max_context_tokens: Math.max(1000, parseInt(e.target.value, 10) || 0),
                      })
                    }
                    className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2 text-sm text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
                  />
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    {t('agent_manage.compactor_max_tokens_help')}
                  </p>
                </div>

                <div>
                  <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
                    {t('agent_manage.compactor_threshold')} ({Math.round(compactorForm.compact_threshold_ratio * 100)}%)
                  </label>
                  <input
                    type="number"
                    min={0.1}
                    max={0.99}
                    step={0.05}
                    value={compactorForm.compact_threshold_ratio}
                    onChange={(e) =>
                      setCompactorForm({
                        ...compactorForm,
                        compact_threshold_ratio: parseFloat(e.target.value) || 0.85,
                      })
                    }
                    className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2 text-sm text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
                  />
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    {t('agent_manage.compactor_threshold_help')}
                  </p>
                </div>

                <div>
                  <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
                    {t('agent_manage.compactor_keep_turns')}
                  </label>
                  <input
                    type="number"
                    min={1}
                    max={50}
                    step={1}
                    value={compactorForm.keep_recent_turns}
                    onChange={(e) =>
                      setCompactorForm({
                        ...compactorForm,
                        keep_recent_turns: Math.max(1, parseInt(e.target.value, 10) || 1),
                      })
                    }
                    className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2 text-sm text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
                  />
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    {t('agent_manage.compactor_keep_turns_help')}
                  </p>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Modal Add / Edit Skill */}
      <Modal
        isOpen={skillModalOpen}
        onClose={() => setSkillModalOpen(false)}
        title={editingSkill ? t('agent_manage.modal_edit_skill_title') : t('agent_manage.modal_create_skill_title')}
      >
        <form onSubmit={handleSaveSkill} className="space-y-4">
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
              {t('agent_manage.skill_id')}
            </label>
            <input
              type="text"
              disabled={!!editingSkill}
              value={skillForm.id}
              onChange={(e) => setSkillForm({ ...skillForm, id: e.target.value })}
              className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2 text-xs text-slate-900 disabled:bg-slate-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100 dark:disabled:bg-slate-900"
            />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
              {t('agent_manage.skill_name')}
            </label>
            <input
              type="text"
              value={skillForm.name}
              onChange={(e) => setSkillForm({ ...skillForm, name: e.target.value })}
              className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2 text-xs text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
              {t('agent_manage.skill_desc')}
            </label>
            <textarea
              rows={2}
              value={skillForm.description}
              onChange={(e) => setSkillForm({ ...skillForm, description: e.target.value })}
              className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2 text-xs text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
              {t('agent_manage.skill_content')}
            </label>
            <textarea
              rows={8}
              value={skillForm.content}
              onChange={(e) => setSkillForm({ ...skillForm, content: e.target.value })}
              className="font-mono w-full rounded-xl border border-slate-300 bg-slate-50 px-3.5 py-2 text-xs text-slate-900 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
              {t('agent_manage.skill_tool_def')}
            </label>
            <textarea
              rows={3}
              value={skillForm.tool_definition}
              onChange={(e) => setSkillForm({ ...skillForm, tool_definition: e.target.value })}
              className="font-mono w-full rounded-xl border border-slate-300 bg-slate-50 px-3.5 py-2 text-xs text-slate-900 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
                {t('agent_manage.skill_priority')}
              </label>
              <input
                type="number"
                value={skillForm.priority}
                onChange={(e) => setSkillForm({ ...skillForm, priority: parseInt(e.target.value, 10) || 0 })}
                className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2 text-xs text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
              />
            </div>
            {!editingSkill && (
              <div className="flex items-center gap-2 pt-5">
                <input
                  type="checkbox"
                  id="skill-enabled"
                  checked={skillForm.is_enabled}
                  onChange={(e) => setSkillForm({ ...skillForm, is_enabled: e.target.checked })}
                  className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                />
                <label htmlFor="skill-enabled" className="text-xs text-slate-700 dark:text-slate-300">
                  {t('agent_manage.skill_enabled')}
                </label>
              </div>
            )}
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <button
              type="button"
              onClick={() => setSkillModalOpen(false)}
              className="rounded-xl border border-slate-300 px-4 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
            >
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              disabled={createSkillMutation.isPending || updateSkillMutation.isPending}
              className="rounded-xl bg-indigo-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-indigo-700 disabled:opacity-50"
            >
              {t('common.save')}
            </button>
          </div>
        </form>
      </Modal>

      {/* Confirmation Modal for Reset */}
      <Modal
        isOpen={!!resetConfirmModal}
        onClose={() => setResetConfirmModal(null)}
        title={t('agent_manage.confirm_reset_title')}
      >
        <div className="space-y-4">
          <p className="text-sm text-slate-600 dark:text-slate-300">
            {resetConfirmModal?.type === 'prompt'
              ? t('agent_manage.confirm_reset_prompt_desc')
              : t('agent_manage.confirm_reset_skill_desc')}
          </p>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setResetConfirmModal(null)}
              className="rounded-xl border border-slate-300 px-4 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
            >
              {t('common.cancel')}
            </button>
            <button
              type="button"
              onClick={() => {
                if (resetConfirmModal?.type === 'prompt') {
                  resetPromptMutation.mutate(resetConfirmModal.id);
                } else if (resetConfirmModal?.type === 'skill') {
                  resetSkillMutation.mutate(resetConfirmModal.id);
                }
              }}
              className="rounded-xl bg-amber-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-amber-700"
            >
              {t('agent_manage.btn_reset_default')}
            </button>
          </div>
        </div>
      </Modal>

      {/* Confirmation Modal for Delete */}
      <Modal
        isOpen={!!deleteConfirmModal}
        onClose={() => setDeleteConfirmModal(null)}
        title={t('agent_manage.confirm_delete_skill_title')}
      >
        <div className="space-y-4">
          <p className="text-sm text-slate-600 dark:text-slate-300">
            {t('agent_manage.confirm_delete_skill_desc')}
          </p>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setDeleteConfirmModal(null)}
              className="rounded-xl border border-slate-300 px-4 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
            >
              {t('common.cancel')}
            </button>
            <button
              type="button"
              onClick={() => {
                if (deleteConfirmModal) {
                  deleteSkillMutation.mutate(deleteConfirmModal);
                }
              }}
              className="rounded-xl bg-rose-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-rose-700"
            >
              {t('common.delete')}
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
};
