import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/react/shallow';
import { useLLMManageStore } from '@/stores/llmManageStore';
import { useEmbeddingManageStore } from '@/stores/embeddingManageStore';
import {
  Cpu,
  Brain,
  Plus,
  Trash2,
  Edit2,
  CheckCircle2,
  Eye,
  EyeOff,
  Star,
  Sparkles,
  Server,
  Layers,
  Search,
  GitBranch,
  ArrowUp,
  ArrowDown,
  RotateCcw,
  AlertTriangle,
  ShieldCheck,
  ShieldAlert,
  Clock,
  Shuffle,
  Play,
  Copy,
  Check,
  Terminal,
  Sliders,
  Image as ImageIcon,
  X,
  Boxes,
  Download,
  FolderOpen,
  RefreshCw,
} from 'lucide-react';
import {
  llmService,
  type LLMProviderDTO,
  type LLMModelDTO,
  type CreateLLMProviderDTO,
  type UpdateLLMProviderDTO,
  type CreateLLMModelDTO,
  type UpdateLLMModelDTO,
  type LLMChainDTO,
  type CreateLLMChainDTO,
  type UpdateLLMChainDTO,
  type TestModelRequestDTO,
} from '@/services/llmService';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { Badge } from '@/components/common/Badge';
import { Modal } from '@/components/common/Modal';
import { usePermissions } from '@/hooks/usePermissions';
import { embeddingService } from '@/services/embeddingService';
import type {
  EmbeddingProvider,
  OnnxModelStatus,
  OnnxDownloadState,
} from '@/types/embedding';

const ONNX_STATUS_VARIANTS: Record<OnnxModelStatus, 'success' | 'danger' | 'warning'> = {
  ok: 'success',
  invalid: 'danger',
  unvalidated: 'warning',
};

const DOWNLOAD_STATE_VARIANTS: Record<OnnxDownloadState, 'info' | 'success' | 'danger'> = {
  running: 'info',
  done: 'success',
  error: 'danger',
};

const formatBytes = (bytes: number): string => {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const unitIndex = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  const value = bytes / 1024 ** unitIndex;
  return `${value >= 100 ? Math.round(value) : value.toFixed(1)} ${units[unitIndex]}`;
};

const extractErrorMessage = (err: unknown, fallback: string): string =>
  (err as { response?: { data?: { message?: string } } })?.response?.data?.message || fallback;

const formatDateTime = (iso: string): string => new Date(iso).toLocaleString();

export const LLMManagePage: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { canAny, isAdmin } = usePermissions();

  const canManageLLM = canAny(['llm.manage', 'setting.manage', 'admin.access']) || isAdmin;

  const {
    activeTab,
    providerFilter,
    isProviderModalOpen,
    editingProvider,
    providerForm,
    isModelModalOpen,
    editingModel,
    modelForm,
    isProbing,
    probeResultMsg,
    actionError,
    isChainModalOpen,
    editingChain,
    chainForm,
    isNodeModalOpen,
    selectedChainForNode,
    nodeForm,
    testTargetType,
    selectedTestModelId,
    selectedTestChainId,
    testSystemPrompt,
    testPrompt,
    testTemperature,
    testMaxTokens,
    testImages,
    isAdvancedOpen,
    testResult,
    isTesting,
    testError,
    activeResultTab,
    copiedRaw,
    setActiveTab,
    setProviderFilter,
    setIsProviderModalOpen,
    setEditingProvider,
    setProviderForm,
    setIsModelModalOpen,
    setEditingModel,
    setModelForm,
    setIsProbing,
    setProbeResultMsg,
    setActionError,
    setIsChainModalOpen,
    setEditingChain,
    setChainForm,
    setIsNodeModalOpen,
    setSelectedChainForNode,
    setNodeForm,
    setTestTargetType,
    setSelectedTestModelId,
    setSelectedTestChainId,
    setTestSystemPrompt,
    setTestPrompt,
    setTestTemperature,
    setTestMaxTokens,
    setTestImages,
    setIsAdvancedOpen,
    setTestResult,
    setIsTesting,
    setTestError,
    setActiveResultTab,
    setCopiedRaw,
  } = useLLMManageStore(
    useShallow((s) => ({
      activeTab: s.activeTab,
      providerFilter: s.providerFilter,
      isProviderModalOpen: s.isProviderModalOpen,
      editingProvider: s.editingProvider,
      providerForm: s.providerForm,
      isModelModalOpen: s.isModelModalOpen,
      editingModel: s.editingModel,
      modelForm: s.modelForm,
      isProbing: s.isProbing,
      probeResultMsg: s.probeResultMsg,
      actionError: s.actionError,
      isChainModalOpen: s.isChainModalOpen,
      editingChain: s.editingChain,
      chainForm: s.chainForm,
      isNodeModalOpen: s.isNodeModalOpen,
      selectedChainForNode: s.selectedChainForNode,
      nodeForm: s.nodeForm,
      testTargetType: s.testTargetType,
      selectedTestModelId: s.selectedTestModelId,
      selectedTestChainId: s.selectedTestChainId,
      testSystemPrompt: s.testSystemPrompt,
      testPrompt: s.testPrompt,
      testTemperature: s.testTemperature,
      testMaxTokens: s.testMaxTokens,
      testImages: s.testImages,
      isAdvancedOpen: s.isAdvancedOpen,
      testResult: s.testResult,
      isTesting: s.isTesting,
      testError: s.testError,
      activeResultTab: s.activeResultTab,
      copiedRaw: s.copiedRaw,
      setActiveTab: s.setActiveTab,
      setProviderFilter: s.setProviderFilter,
      setIsProviderModalOpen: s.setIsProviderModalOpen,
      setEditingProvider: s.setEditingProvider,
      setProviderForm: s.setProviderForm,
      setIsModelModalOpen: s.setIsModelModalOpen,
      setEditingModel: s.setEditingModel,
      setModelForm: s.setModelForm,
      setIsProbing: s.setIsProbing,
      setProbeResultMsg: s.setProbeResultMsg,
      setActionError: s.setActionError,
      setIsChainModalOpen: s.setIsChainModalOpen,
      setEditingChain: s.setEditingChain,
      setChainForm: s.setChainForm,
      setIsNodeModalOpen: s.setIsNodeModalOpen,
      setSelectedChainForNode: s.setSelectedChainForNode,
      setNodeForm: s.setNodeForm,
      setTestTargetType: s.setTestTargetType,
      setSelectedTestModelId: s.setSelectedTestModelId,
      setSelectedTestChainId: s.setSelectedTestChainId,
      setTestSystemPrompt: s.setTestSystemPrompt,
      setTestPrompt: s.setTestPrompt,
      setTestTemperature: s.setTestTemperature,
      setTestMaxTokens: s.setTestMaxTokens,
      setTestImages: s.setTestImages,
      setIsAdvancedOpen: s.setIsAdvancedOpen,
      setTestResult: s.setTestResult,
      setIsTesting: s.setIsTesting,
      setTestError: s.setTestError,
      setActiveResultTab: s.setActiveResultTab,
      setCopiedRaw: s.setCopiedRaw,
    }))
  );
  const {
    isEmbeddingTabOpen,
    providerDraft,
    providerTouched,
    isDownloadModalOpen,
    deleteTarget,
    copiedModelId,
    toastMessage,
    toastTone,
    setIsEmbeddingTabOpen,
    setProviderDraft,
    resetProviderDraft,
    setIsDownloadModalOpen,
    setDeleteTarget,
    setCopiedModelId,
    setToast,
  } = useEmbeddingManageStore(
    useShallow((s) => ({
      isEmbeddingTabOpen: s.isEmbeddingTabOpen,
      providerDraft: s.providerDraft,
      providerTouched: s.providerTouched,
      isDownloadModalOpen: s.isDownloadModalOpen,
      deleteTarget: s.deleteTarget,
      copiedModelId: s.copiedModelId,
      toastMessage: s.toastMessage,
      toastTone: s.toastTone,
      setIsEmbeddingTabOpen: s.setIsEmbeddingTabOpen,
      setProviderDraft: s.setProviderDraft,
      resetProviderDraft: s.resetProviderDraft,
      setIsDownloadModalOpen: s.setIsDownloadModalOpen,
      setDeleteTarget: s.setDeleteTarget,
      setCopiedModelId: s.setCopiedModelId,
      setToast: s.setToast,
    }))
  );
  const testFileInputRef = useRef<HTMLInputElement>(null);
  const [showProviderKey, setShowProviderKey] = useState(false);

  const { data: providers = [], isLoading: isLoadingProviders } = useQuery({
    queryKey: ['llm-providers'],
    queryFn: llmService.listProviders,
  });

  const { data: models = [], isLoading: isLoadingModels } = useQuery({
    queryKey: ['llm-models', providerFilter],
    queryFn: () => llmService.listModels(providerFilter || undefined),
  });

  const { data: chains = [], isLoading: isLoadingChains } = useQuery({
    queryKey: ['llm-chains'],
    queryFn: llmService.listChains,
    refetchInterval: 5000,
  });

  const {
    data: embeddingSettings,
    isLoading: isLoadingEmbeddingSettings,
  } = useQuery({
    queryKey: ['embedding-settings'],
    queryFn: embeddingService.getSettings,
  });

  const {
    data: onnxModels = [],
    isLoading: isLoadingOnnxModels,
    refetch: refetchOnnxModels,
    isRefetching: isRescanningOnnxModels,
  } = useQuery({
    queryKey: ['embedding-onnx-models'],
    queryFn: embeddingService.listOnnxModels,
  });

  const { data: onnxCatalog } = useQuery({
    queryKey: ['embedding-onnx-catalog'],
    queryFn: embeddingService.getOnnxCatalog,
  });

  const { data: onnxDownloadStatus } = useQuery({
    queryKey: ['embedding-download-status'],
    queryFn: embeddingService.getOnnxDownloadStatus,
    refetchInterval: (query) => {
      const jobs = query.state.data?.jobs ?? [];
      return jobs.some((job) => job.state === 'running') ? 1500 : false;
    },
  });

  const saveProviderMutation = useMutation({
    mutationFn: async () => {
      if (editingProvider) {
        const updateData: UpdateLLMProviderDTO = {
          name: providerForm.name,
          provider_type: providerForm.provider_type,
          base_url: providerForm.base_url,
          api_key: providerForm.api_key || undefined,
          is_active: providerForm.is_active,
          is_default: providerForm.is_default,
          timeout_seconds: providerForm.timeout_seconds,
          max_retries: providerForm.max_retries,
          retry_initial_wait_ms: providerForm.retry_initial_wait_ms,
          retry_max_wait_ms: providerForm.retry_max_wait_ms,
          allow_private_networks: providerForm.allow_private_networks,
        };
        return llmService.updateProvider(editingProvider.id, updateData);
      }
      return llmService.createProvider(providerForm);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['llm-providers'] });
      queryClient.invalidateQueries({ queryKey: ['chat-models'] });
      setIsProviderModalOpen(false);
      setEditingProvider(null);
    },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message || t('llm.action_failed');
      setActionError(msg);
    },
  });

  const deleteProviderMutation = useMutation({
    mutationFn: llmService.deleteProvider,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['llm-providers'] });
      queryClient.invalidateQueries({ queryKey: ['llm-models'] });
      queryClient.invalidateQueries({ queryKey: ['chat-models'] });
    },
  });

  const setDefaultProviderMutation = useMutation({
    mutationFn: llmService.setDefaultProvider,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['llm-providers'] });
    },
  });

  const saveModelMutation = useMutation({
    mutationFn: async () => {
      if (editingModel) {
        const updateData: UpdateLLMModelDTO = {
          provider_id: modelForm.provider_id,
          name: modelForm.name,
          model_key: modelForm.model_key,
          model_type: modelForm.model_type,
          vision_mode: modelForm.vision_mode,
          supports_vision: modelForm.supports_vision,
          max_images: modelForm.max_images,
          thinking_enabled: modelForm.thinking_enabled,
          thinking_budget: modelForm.thinking_budget,
          effort_level: modelForm.effort_level,
          context_length: modelForm.context_length,
          order_index: modelForm.order_index,
          is_active: modelForm.is_active,
          is_default: modelForm.is_default,
        };
        return llmService.updateModel(editingModel.id, updateData);
      }
      return llmService.createModel(modelForm);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['llm-models'] });
      queryClient.invalidateQueries({ queryKey: ['chat-models'] });
      setIsModelModalOpen(false);
      setEditingModel(null);
    },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message || t('llm.action_failed');
      setActionError(msg);
    },
  });

  const deleteModelMutation = useMutation({
    mutationFn: llmService.deleteModel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['llm-models'] });
      queryClient.invalidateQueries({ queryKey: ['chat-models'] });
    },
  });

  const saveChainMutation = useMutation({
    mutationFn: async () => {
      if (editingChain) {
        const updateData: UpdateLLMChainDTO = {
          name: chainForm.name,
          description: chainForm.description,
          is_active: chainForm.is_active,
          failure_threshold: chainForm.failure_threshold,
          cooldown_seconds: chainForm.cooldown_seconds,
          retry_count: chainForm.retry_count,
        };
        return llmService.updateChain(editingChain.id, updateData);
      }
      return llmService.createChain(chainForm);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['llm-chains'] });
      setIsChainModalOpen(false);
      setEditingChain(null);
    },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message || t('llm.action_failed');
      setActionError(msg);
    },
  });

  const deleteChainMutation = useMutation({
    mutationFn: llmService.deleteChain,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['llm-chains'] });
    },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message || t('llm.delete_failed');
      setActionError(msg);
    },
  });

  const addNodeMutation = useMutation({
    mutationFn: async () => {
      if (!selectedChainForNode) return;
      return llmService.addChainNode(selectedChainForNode.id, nodeForm);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['llm-chains'] });
      setIsNodeModalOpen(false);
      setSelectedChainForNode(null);
    },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message || t('llm.add_node_failed');
      setActionError(msg);
    },
  });

  const removeNodeMutation = useMutation({
    mutationFn: ({ chainId, nodeId }: { chainId: string; nodeId: string }) =>
      llmService.removeChainNode(chainId, nodeId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['llm-chains'] });
    },
  });

  const updateNodePriorityMutation = useMutation({
    mutationFn: ({
      chainId,
      nodeId,
      priority,
      isActive,
    }: {
      chainId: string;
      nodeId: string;
      priority: number;
      isActive: boolean;
    }) =>
      llmService.updateChainNodePriority(chainId, nodeId, { priority, is_active: isActive }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['llm-chains'] });
    },
  });

  const resetBreakerMutation = useMutation({
    mutationFn: ({ chainId, nodeId }: { chainId: string; nodeId: string }) =>
      llmService.resetNodeCircuitBreaker(chainId, nodeId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['llm-chains'] });
    },
  });

  const embeddingRuntime = embeddingSettings?.runtime;
  const downloadJobs = onnxDownloadStatus?.jobs ?? [];
  const okModelCount = onnxModels.filter((m) => m.status === 'ok').length;
  const effectiveProvider: EmbeddingProvider = providerTouched
    ? providerDraft
    : (embeddingSettings?.provider ?? 'api');
  const activeOnnxModelName = onnxModels.find(
    (m) => m.model_id === embeddingSettings?.active_onnx_model_id
  )?.display_name;
  const matchedRuntimes = (onnxCatalog?.runtimes ?? []).filter(
    (rt) => embeddingRuntime && rt.goos === embeddingRuntime.os && rt.goarch === embeddingRuntime.arch
  );

  const apiEmbeddingModels = useMemo(
    () => models.filter((m) => m.model_type === 'embedding'),
    [models]
  );
  const totalEmbeddingCount = apiEmbeddingModels.length + onnxModels.length;
  const activeEmbeddingChain = useMemo(
    () => chains.find((c) => c.chain_type === 'embedding' && c.is_active),
    [chains]
  );

  const resolvePresetName = (presetId: string): string => {
    const modelPreset = onnxCatalog?.models.find((p) => p.preset_id === presetId);
    if (modelPreset) return modelPreset.display_name;
    const runtimePreset = onnxCatalog?.runtimes.find((p) => p.preset_id === presetId);
    return runtimePreset?.display_name || presetId;
  };

  const processedJobsRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    if (!embeddingSettings || providerTouched) return;
    resetProviderDraft(embeddingSettings.provider);
  }, [embeddingSettings, providerTouched, resetProviderDraft]);

  useEffect(() => {
    const jobs = onnxDownloadStatus?.jobs ?? [];
    for (const job of jobs) {
      if (job.state === 'running' || processedJobsRef.current.has(job.job_id)) continue;
      processedJobsRef.current.add(job.job_id);
      if (job.state === 'done') {
        queryClient.invalidateQueries({ queryKey: ['embedding-onnx-models'] });
        queryClient.invalidateQueries({ queryKey: ['embedding-settings'] });
        const modelPreset = onnxCatalog?.models.find((p) => p.preset_id === job.preset_id);
        const runtimePreset = onnxCatalog?.runtimes.find((p) => p.preset_id === job.preset_id);
        const presetName = modelPreset?.display_name || runtimePreset?.display_name || job.preset_id;
        setToast(t('embedding.download_done_toast', { name: presetName }), 'success');
      } else {
        setToast(
          t('embedding.download_error_toast', { error: job.error || t('embedding.status_error') }),
          'error'
        );
      }
    }
  }, [onnxDownloadStatus, onnxCatalog, queryClient, setToast, t]);

  const saveEmbeddingSettingsMutation = useMutation({
    mutationFn: () =>
      embeddingService.updateSettings({
        provider: effectiveProvider,
        active_onnx_model_id:
          effectiveProvider === 'onnx' ? (embeddingSettings?.active_onnx_model_id ?? null) : undefined,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['embedding-settings'] });
      setToast(t('embedding.settings_saved'), 'success');
    },
    onError: (err: unknown) =>
      setToast(extractErrorMessage(err, t('embedding.settings_save_failed')), 'error'),
  });

  const startDownloadMutation = useMutation({
    mutationFn: (presetId: string) => embeddingService.startOnnxDownload(presetId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['embedding-download-status'] });
      setToast(t('embedding.download_started'), 'success');
    },
    onError: (err: unknown) =>
      setToast(extractErrorMessage(err, t('embedding.download_start_failed')), 'error'),
  });

  const validateModelMutation = useMutation({
    mutationFn: (modelId: string) => embeddingService.validateOnnxModel(modelId),
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ['embedding-onnx-models'] });
      if (res.ok) {
        setToast(t('embedding.validate_success', { dim: res.dim, ms: res.latency_ms }), 'success');
      } else {
        setToast(t('embedding.validate_failed', { error: res.error || '' }), 'error');
      }
    },
    onError: (err: unknown) => setToast(extractErrorMessage(err, t('embedding.action_failed')), 'error'),
  });

  const activateModelMutation = useMutation({
    mutationFn: (modelId: string) => embeddingService.activateOnnxModel(modelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['embedding-onnx-models'] });
      queryClient.invalidateQueries({ queryKey: ['embedding-settings'] });
      setToast(t('embedding.activate_success'), 'success');
    },
    onError: (err: unknown) =>
      setToast(extractErrorMessage(err, t('embedding.activate_failed')), 'error'),
  });

  const deleteOnnxModelMutation = useMutation({
    mutationFn: (modelId: string) => embeddingService.deleteOnnxModel(modelId),
    onSuccess: () => {
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: ['embedding-onnx-models'] });
      queryClient.invalidateQueries({ queryKey: ['embedding-settings'] });
      setToast(t('embedding.delete_success'), 'success');
    },
    onError: (err: unknown) => setToast(extractErrorMessage(err, t('embedding.delete_failed')), 'error'),
  });

  const handleOpenAddChain = () => {
    setEditingChain(null);
    setChainForm({
      id: '',
      chain_type: 'chat',
      name: '',
      description: '',
      is_active: true,
      failure_threshold: 3,
      cooldown_seconds: 300,
      retry_count: 3,
    });
    setActionError(null);
    setIsChainModalOpen(true);
  };

  const handleOpenEditChain = (c: LLMChainDTO) => {
    setEditingChain(c);
    setChainForm({
      id: c.id,
      chain_type: c.chain_type,
      name: c.name,
      description: c.description,
      is_active: c.is_active,
      failure_threshold: c.failure_threshold,
      cooldown_seconds: c.cooldown_seconds,
      retry_count: c.retry_count,
    });
    setActionError(null);
    setIsChainModalOpen(true);
  };

  const handleOpenAddNode = (c: LLMChainDTO) => {
    setSelectedChainForNode(c);
    const validModels = models.filter((m) => {
      if (m.model_type !== c.chain_type) return false;
      if (c.chain_type === 'embedding' && c.nodes.length > 0) {
        return m.model_key === c.nodes[0].model_key;
      }
      return true;
    });
    setNodeForm({
      model_id: validModels[0]?.id || '',
      priority: c.nodes.length + 1,
      is_active: true,
    });
    setActionError(null);
    setIsNodeModalOpen(true);
  };

  const handleOpenAddProvider = () => {
    setEditingProvider(null);
    setProviderForm({
      id: '',
      name: '',
      provider_type: 'openrouter',
      base_url: 'https://openrouter.ai/api/v1',
      api_key: '',
      is_active: true,
      is_default: false,
      timeout_seconds: 60,
      max_retries: 3,
      retry_initial_wait_ms: 500,
      retry_max_wait_ms: 5000,
      allow_private_networks: false,
    });
    setActionError(null);
    setShowProviderKey(false);
    setIsProviderModalOpen(true);
  };

  const handleOpenEditProvider = (p: LLMProviderDTO) => {
    setEditingProvider(p);
    setProviderForm({
      id: p.id,
      name: p.name,
      provider_type: p.provider_type,
      base_url: p.base_url,
      api_key: '',
      is_active: p.is_active,
      is_default: p.is_default,
      timeout_seconds: p.timeout_seconds ?? 60,
      max_retries: p.max_retries ?? 3,
      retry_initial_wait_ms: p.retry_initial_wait_ms ?? 500,
      retry_max_wait_ms: p.retry_max_wait_ms ?? 5000,
      allow_private_networks: p.allow_private_networks ?? false,
    });
    setActionError(null);
    setShowProviderKey(false);
    setIsProviderModalOpen(true);
  };

  const handleOpenAddModel = () => {
    const defaultProvId = providers[0]?.id || '';
    setEditingModel(null);
    setModelForm({
      id: '',
      provider_id: defaultProvId,
      name: '',
      model_key: '',
      model_type: 'chat',
      vision_mode: 'auto',
      supports_vision: false,
      max_images: 5,
      context_length: 32768,
      order_index: 0,
      is_active: true,
      is_default: false,
      thinking_enabled: false,
      thinking_budget: 0,
      effort_level: 'medium',
    });
    setProbeResultMsg(null);
    setActionError(null);
    setIsModelModalOpen(true);
  };

  const handleOpenEditModel = (m: LLMModelDTO) => {
    setEditingModel(m);
    setModelForm({
      id: m.id,
      provider_id: m.provider_id,
      name: m.name,
      model_key: m.model_key,
      model_type: m.model_type,
      vision_mode: m.vision_mode,
      supports_vision: m.supports_vision,
      max_images: m.max_images,
      thinking_enabled: m.thinking_enabled ?? false,
      thinking_budget: m.thinking_budget ?? 0,
      effort_level: m.effort_level ?? 'medium',
      context_length: m.context_length,
      order_index: m.order_index,
      is_active: m.is_active,
      is_default: m.is_default,
    });
    setProbeResultMsg(null);
    setActionError(null);
    setIsModelModalOpen(true);
  };

  const handleProbeVision = async () => {
    if (!modelForm.provider_id || !modelForm.model_key) {
      setActionError(t('llm.toast_probe_failed'));
      return;
    }

    setIsProbing(true);
    setProbeResultMsg(null);
    setActionError(null);

    try {
      const res = await llmService.probeVision({
        provider_id: modelForm.provider_id,
        model_key: modelForm.model_key,
      });

      setModelForm({
        supports_vision: res.supports_vision,
        max_images: res.suggested_max_images > 0 ? res.suggested_max_images : 5,
      });

      setProbeResultMsg(
        res.supports_vision
          ? `${t('llm.toast_probe_success')} (${t('llm.probe_vision_ok', { max: res.suggested_max_images })})`
          : `${t('llm.toast_probe_success')} (${t('llm.probe_vision_no')})`
      );
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message || t('llm.toast_probe_failed');
      setActionError(msg);
    } finally {
      setIsProbing(false);
    }
  };

  const handleExecuteTest = async () => {
    if (!testPrompt.trim() || isTesting) return;
    setIsTesting(true);
    setTestError(null);
    setTestResult(null);

    const payload: TestModelRequestDTO = {
      model_id: testTargetType === 'model' ? selectedTestModelId || models[0]?.id || undefined : undefined,
      chain_id: testTargetType === 'chain' ? selectedTestChainId || chains[0]?.id || undefined : undefined,
      system_prompt: testSystemPrompt.trim() || undefined,
      prompt: testPrompt.trim(),
      images: testImages.length > 0 ? testImages : undefined,
      temperature: testTemperature,
      max_tokens: testMaxTokens,
    };

    try {
      const res = await llmService.testModel(payload);
      setTestResult(res);
      if (!res.success && res.error) {
        setTestError(res.error);
      }
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        (err as Error)?.message ||
        t('llm.test_exec_failed');
      setTestError(msg);
    } finally {
      setIsTesting(false);
    }
  };

  const handleTestFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files || files.length === 0) return;
    const file = files[0];
    const reader = new FileReader();
    reader.addEventListener('load', () => {
      const base64 = reader.result as string;
      setTestImages((prev) => [...prev, base64]);
    });
    reader.readAsDataURL(file);
    e.target.value = '';
  };

  return (
    <div className="space-y-8 max-w-7xl mx-auto">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="min-w-0">
          <div className="flex items-center gap-3">
            <div className="shrink-0 p-2 rounded-xl bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 border border-indigo-100 dark:border-indigo-900/50">
              <Cpu className="w-6 h-6" />
            </div>
            <div className="min-w-0">
              <h2 className="truncate text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100 sm:text-2xl">
                {t('llm.title')}
              </h2>
              <p className="text-sm text-slate-500 dark:text-slate-400 mt-0.5">
                {t('llm.subtitle')}
              </p>
            </div>
          </div>
        </div>

        {canManageLLM && !isEmbeddingTabOpen && (
          <div className="flex flex-col gap-2 sm:w-auto sm:flex-row sm:items-center">
            {activeTab === 'providers' && (
              <button
                onClick={handleOpenAddProvider}
                className="inline-flex min-h-[44px] w-full items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-sm shadow-md shadow-indigo-600/20 transition sm:w-auto"
              >
                <Plus className="w-4 h-4" />
                <span>{t('llm.add_provider')}</span>
              </button>
            )}
            {activeTab === 'models' && (
              <button
                onClick={handleOpenAddModel}
                className="inline-flex min-h-[44px] w-full items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-sm shadow-md shadow-indigo-600/20 transition sm:w-auto"
              >
                <Plus className="w-4 h-4" />
                <span>{t('llm.add_model')}</span>
              </button>
            )}
            {activeTab === 'chains' && (
              <button
                onClick={handleOpenAddChain}
                className="inline-flex min-h-[44px] w-full items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-sm shadow-md shadow-indigo-600/20 transition sm:w-auto"
              >
                <Plus className="w-4 h-4" />
                <span>{t('llm.add_chain')}</span>
              </button>
            )}
          </div>
        )}
      </div>

      <div className="flex gap-4 overflow-x-auto border-b border-slate-200 dark:border-slate-800 sm:gap-6">
        <button
          onClick={() => {
            setActiveTab('providers');
            setIsEmbeddingTabOpen(false);
          }}
          className={`min-h-[44px] shrink-0 whitespace-nowrap pb-3 font-semibold text-sm flex items-center gap-2 transition border-b-2 ${
            activeTab === 'providers' && !isEmbeddingTabOpen
              ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400'
              : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
          }`}
        >
          <Server className="w-4 h-4" />
          <span>{t('llm.tab_providers')}</span>
          <Badge variant="secondary" size="sm">
            {providers.length}
          </Badge>
        </button>

        <button
          onClick={() => {
            setActiveTab('models');
            setIsEmbeddingTabOpen(false);
          }}
          className={`min-h-[44px] shrink-0 whitespace-nowrap pb-3 font-semibold text-sm flex items-center gap-2 transition border-b-2 ${
            activeTab === 'models' && !isEmbeddingTabOpen
              ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400'
              : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
          }`}
        >
          <Layers className="w-4 h-4" />
          <span>{t('llm.tab_models')}</span>
          <Badge variant="secondary" size="sm">
            {models.length}
          </Badge>
        </button>

        <button
          onClick={() => {
            setActiveTab('chains');
            setIsEmbeddingTabOpen(false);
          }}
          className={`min-h-[44px] shrink-0 whitespace-nowrap pb-3 font-semibold text-sm flex items-center gap-2 transition border-b-2 ${
            activeTab === 'chains' && !isEmbeddingTabOpen
              ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400'
              : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
          }`}
        >
          <GitBranch className="w-4 h-4" />
          <span>{t('llm.tab_chains')}</span>
          <Badge variant="secondary" size="sm">
            {chains.length}
          </Badge>
        </button>

        <button
          onClick={() => {
            setActiveTab('playground');
            setIsEmbeddingTabOpen(false);
          }}
          className={`min-h-[44px] shrink-0 whitespace-nowrap pb-3 font-semibold text-sm flex items-center gap-2 transition border-b-2 ${
            activeTab === 'playground' && !isEmbeddingTabOpen
              ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400'
              : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
          }`}
        >
          <Sparkles className="w-4 h-4" />
          <span>{t('llm.tab_playground')}</span>
          <span className="px-1.5 py-0.5 rounded-full text-[10px] font-bold bg-indigo-100 dark:bg-indigo-950 text-indigo-600 dark:text-indigo-400">
            {t('llm.badge_playground')}
          </span>
        </button>

        <button
          onClick={() => setIsEmbeddingTabOpen(true)}
          className={`min-h-[44px] shrink-0 whitespace-nowrap pb-3 font-semibold text-sm flex items-center gap-2 transition border-b-2 ${
            isEmbeddingTabOpen
              ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400'
              : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
          }`}
        >
          <Boxes className="w-4 h-4" />
          <span>{t('embedding.tab_label')}</span>
          <Badge variant="secondary" size="sm">
            {totalEmbeddingCount}
          </Badge>
        </button>
      </div>

      {activeTab === 'providers' && !isEmbeddingTabOpen && (
        <div className="space-y-4">
          {isLoadingProviders ? (
            <div className="py-12 flex justify-center">
              <LoadingSpinner size="lg" />
            </div>
          ) : providers.length === 0 ? (
            <div className="p-8 text-center bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 text-slate-500">
              {t('llm.no_providers')}
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 sm:gap-5 xl:grid-cols-3">
              {providers.map((p) => (
                <div
                  key={p.id}
                  className="p-4 sm:p-5 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col justify-between"
                >
                  <div className="space-y-3">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="min-w-0 break-words font-bold text-base text-slate-900 dark:text-slate-100">
                            {p.name}
                          </span>
                          {p.is_default && (
                            <Badge variant="warning" size="sm">
                              <Star className="w-3 h-3 mr-1 fill-current" />
                              {t('llm.default_badge')}
                            </Badge>
                          )}
                        </div>
                        <span className="block truncate text-xs font-mono text-slate-400">
                          {t('llm.provider_id')}: {p.id}
                        </span>
                      </div>

                      <Badge variant={p.is_active ? 'success' : 'secondary'} size="sm">
                        {p.is_active ? t('llm.active_badge') : t('llm.inactive_badge')}
                      </Badge>
                    </div>

                    <div className="min-w-0 text-xs space-y-1.5 text-slate-600 dark:text-slate-400">
                      <div className="flex items-center justify-between gap-2">
                        <span className="font-medium text-slate-500">{t('llm.provider_type')}:</span>
                        <Badge variant="primary" size="sm">
                          {p.provider_type.toUpperCase()}
                        </Badge>
                      </div>
                      <div className="truncate">
                        <span className="font-medium text-slate-500">{t('llm.base_url')}: </span>
                        <span className="font-mono text-[11px] text-slate-700 dark:text-slate-300">
                          {p.base_url}
                        </span>
                      </div>
                      <div className="flex items-center justify-between gap-2">
                        <span className="font-medium text-slate-500">{t('llm.api_key')}:</span>
                        <span className="break-all font-mono text-xs px-2 py-0.5 rounded bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300">
                          {p.masked_api_key}
                        </span>
                      </div>
                      <div className="flex flex-wrap items-center gap-1.5 pt-1">
                        <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-mono bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400">
                          {p.timeout_seconds ?? 60}{t('llm.timeout_suffix')}
                        </span>
                        <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-mono bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400">
                          {p.max_retries ?? 3} {t('llm.retries_suffix')}
                        </span>
                        {p.allow_private_networks && (
                          <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-400">
                            {t('llm.lan_allowed_badge')}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  {canManageLLM && (
                    <div className="mt-5 pt-4 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between gap-2">
                      {!p.is_default ? (
                        <button
                          onClick={() => setDefaultProviderMutation.mutate(p.id)}
                          disabled={setDefaultProviderMutation.isPending}
                          className="flex min-h-[44px] items-center gap-1 text-xs font-semibold text-slate-600 transition hover:text-indigo-600 dark:text-slate-300 dark:hover:text-indigo-400 sm:min-h-0"
                        >
                          <Star className="w-3.5 h-3.5" />
                          <span>{t('llm.set_default')}</span>
                        </button>
                      ) : (
                        <div />
                      )}

                      <div className="flex items-center gap-1">
                        <button
                          onClick={() => handleOpenEditProvider(p)}
                          className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-100 hover:text-indigo-600 dark:hover:bg-slate-800 sm:h-9 sm:w-9"
                        >
                          <Edit2 className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => {
                            if (window.confirm(t('llm.confirm_delete_provider'))) {
                              deleteProviderMutation.mutate(p.id);
                            }
                          }}
                          disabled={deleteProviderMutation.isPending}
                          className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-100 hover:text-rose-600 dark:hover:bg-slate-800 sm:h-9 sm:w-9"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {activeTab === 'models' && !isEmbeddingTabOpen && (
        <div className="space-y-4">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-4 p-4 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800">
            <div className="flex items-center gap-3 w-full sm:w-auto">
              <Search className="w-4 h-4 text-slate-400 shrink-0" />
              <select
                value={providerFilter}
                onChange={(e) => setProviderFilter(e.target.value)}
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:w-64 sm:py-1.5 sm:text-sm"
              >
                <option value="">{t('llm.all_providers')}</option>
                {providers.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name} ({p.id})
                  </option>
                ))}
              </select>
            </div>
          </div>

          {isLoadingModels ? (
            <div className="py-12 flex justify-center">
              <LoadingSpinner size="lg" />
            </div>
          ) : models.length === 0 ? (
            <div className="p-8 text-center bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 text-slate-500">
              {t('llm.no_models')}
            </div>
          ) : (
            <>
            <div className="hidden overflow-x-auto rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 shadow-xs md:block">
              <table className="w-full min-w-[720px] text-left text-sm text-slate-600 dark:text-slate-400">
                <thead className="bg-slate-50 dark:bg-slate-800/60 border-b border-slate-200 dark:border-slate-800 text-xs uppercase font-semibold text-slate-500 dark:text-slate-400">
                  <tr>
                    <th className="px-5 py-3.5">{t('llm.model_name')}</th>
                    <th className="px-5 py-3.5">{t('llm.model_key')}</th>
                    <th className="px-5 py-3.5">{t('llm.provider_name')}</th>
                    <th className="px-5 py-3.5">{t('llm.model_type')}</th>
                    <th className="px-5 py-3.5">{t('llm.supports_vision')}</th>
                    <th className="px-5 py-3.5">{t('llm.thinking_header')}</th>
                    <th className="px-5 py-3.5">{t('common.status')}</th>
                    <th className="px-5 py-3.5 text-right">{t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                  {models.map((m) => (
                    <tr
                      key={m.id}
                      className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors"
                    >
                      <td className="px-5 py-4">
                        <div className="flex items-center gap-2">
                          <span className="font-semibold text-slate-900 dark:text-slate-100">
                            {m.name}
                          </span>
                          {m.is_default && (
                            <Badge variant="warning" size="sm">
                              {t('llm.default_badge')}
                            </Badge>
                          )}
                        </div>
                      </td>
                      <td className="px-5 py-4 font-mono text-xs text-slate-700 dark:text-slate-300">
                        {m.model_key}
                      </td>
                      <td className="px-5 py-4">
                        <span className="text-xs font-medium text-slate-600 dark:text-slate-400">
                          {m.provider_name || m.provider_id}
                        </span>
                      </td>
                      <td className="px-5 py-4">
                        <Badge
                          variant={
                            m.model_type === 'chat'
                              ? 'primary'
                              : m.model_type === 'embedding'
                              ? 'success'
                              : 'secondary'
                          }
                          size="sm"
                        >
                          {m.model_type.toUpperCase()}
                        </Badge>
                      </td>
                      <td className="px-5 py-4">
                        {m.supports_vision ? (
                          <div className="flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400 text-xs font-semibold">
                            <Eye className="w-3.5 h-3.5" />
                            <span>{t('llm.vision_max_badge', { max: m.max_images })}</span>
                          </div>
                        ) : (
                          <div className="flex items-center gap-1.5 text-slate-400 text-xs font-medium">
                            <EyeOff className="w-3.5 h-3.5" />
                            <span>{t('llm.text_only_badge')}</span>
                          </div>
                        )}
                      </td>
                      <td className="px-5 py-4">
                        {m.thinking_enabled ? (
                          <div className="flex items-center gap-1.5 text-purple-600 dark:text-purple-400 text-xs font-semibold">
                            <Brain className="w-3.5 h-3.5" />
                            <span>{m.effort_level ? m.effort_level.toUpperCase() : t('llm.thinking_active')}</span>
                          </div>
                        ) : (
                          <span className="text-slate-400 text-xs font-medium">-</span>
                        )}
                      </td>
                      <td className="px-5 py-4">
                        <Badge variant={m.is_active ? 'success' : 'secondary'} size="sm">
                          {m.is_active ? t('llm.active_badge') : t('llm.inactive_badge')}
                        </Badge>
                      </td>
                      <td className="px-5 py-4 text-right">
                        {canManageLLM ? (
                          <div className="flex items-center justify-end gap-1">
                            <button
                              onClick={() => handleOpenEditModel(m)}
                              className="p-1.5 rounded-lg text-slate-400 hover:text-indigo-600 hover:bg-slate-100 dark:hover:bg-slate-800 transition"
                            >
                              <Edit2 className="w-4 h-4" />
                            </button>
                            <button
                              onClick={() => {
                                if (window.confirm(t('llm.confirm_delete_model'))) {
                                  deleteModelMutation.mutate(m.id);
                                }
                              }}
                              disabled={deleteModelMutation.isPending}
                              className="p-1.5 rounded-lg text-slate-400 hover:text-rose-600 hover:bg-slate-100 dark:hover:bg-slate-800 transition"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          </div>
                        ) : (
                          <span className="text-xs text-slate-400">—</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="space-y-3 md:hidden">
              {models.map((m) => (
                <div
                  key={m.id}
                  className="p-4 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs"
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="min-w-0 break-words font-semibold text-slate-900 dark:text-slate-100">
                          {m.name}
                        </span>
                        {m.is_default && (
                          <Badge variant="warning" size="sm">
                            {t('llm.default_badge')}
                          </Badge>
                        )}
                      </div>
                      <span className="mt-0.5 block break-all font-mono text-xs text-slate-700 dark:text-slate-300">
                        {m.model_key}
                      </span>
                    </div>
                    <Badge
                      variant={
                        m.model_type === 'chat'
                          ? 'primary'
                          : m.model_type === 'embedding'
                          ? 'success'
                          : 'secondary'
                      }
                      size="sm"
                    >
                      {m.model_type.toUpperCase()}
                    </Badge>
                  </div>

                  <div className="mt-3 space-y-1.5 text-xs text-slate-600 dark:text-slate-400">
                    <div className="flex items-center justify-between gap-2">
                      <span className="shrink-0 font-medium text-slate-500">{t('llm.col_provider')}</span>
                      <span className="min-w-0 truncate font-medium text-slate-600 dark:text-slate-400">
                        {m.provider_name || m.provider_id}
                      </span>
                    </div>
                    <div className="flex items-center justify-between gap-2">
                      <span className="shrink-0 font-medium text-slate-500">{t('llm.supports_vision')}</span>
                      {m.supports_vision ? (
                        <span className="flex items-center gap-1.5 font-semibold text-emerald-600 dark:text-emerald-400">
                          <Eye className="w-3.5 h-3.5" />
                          <span>{t('llm.vision_max_badge', { max: m.max_images })}</span>
                        </span>
                      ) : (
                        <span className="flex items-center gap-1.5 font-medium text-slate-400">
                          <EyeOff className="w-3.5 h-3.5" />
                          <span>{t('llm.text_only_badge')}</span>
                        </span>
                      )}
                    </div>
                    <div className="flex items-center justify-between gap-2">
                      <span className="shrink-0 font-medium text-slate-500">{t('llm.thinking_header')}</span>
                      {m.thinking_enabled ? (
                        <span className="flex items-center gap-1.5 font-semibold text-purple-600 dark:text-purple-400">
                          <Brain className="w-3.5 h-3.5" />
                          <span>{m.effort_level ? m.effort_level.toUpperCase() : t('llm.thinking_active')}</span>
                        </span>
                      ) : (
                        <span className="font-medium text-slate-400">-</span>
                      )}
                    </div>
                    <div className="flex items-center justify-between gap-2">
                      <span className="shrink-0 font-medium text-slate-500">{t('common.status')}</span>
                      <Badge variant={m.is_active ? 'success' : 'secondary'} size="sm">
                        {m.is_active ? t('llm.active_badge') : t('llm.inactive_badge')}
                      </Badge>
                    </div>
                  </div>

                  {canManageLLM && (
                    <div className="mt-3 flex items-center justify-end gap-1 border-t border-slate-100 dark:border-slate-800 pt-3">
                      <button
                        onClick={() => handleOpenEditModel(m)}
                        className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-100 hover:text-indigo-600 dark:hover:bg-slate-800"
                      >
                        <Edit2 className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => {
                          if (window.confirm(t('llm.confirm_delete_model'))) {
                            deleteModelMutation.mutate(m.id);
                          }
                        }}
                        disabled={deleteModelMutation.isPending}
                        className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-100 hover:text-rose-600 dark:hover:bg-slate-800"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  )}
                </div>
              ))}
            </div>
            </>
          )}
        </div>
      )}

      {activeTab === 'chains' && !isEmbeddingTabOpen && (
        <div className="space-y-6">
          <div className="p-4 rounded-xl border border-indigo-100 dark:border-indigo-950/60 bg-indigo-50/50 dark:bg-indigo-950/20 text-xs text-indigo-900 dark:text-indigo-200 flex items-start gap-3">
            <Shuffle className="w-5 h-5 text-indigo-600 dark:text-indigo-400 shrink-0 mt-0.5" />
            <div className="space-y-1">
              <span className="font-bold text-sm block">{t('llm.circuit_breaker_title')}</span>
              <p className="text-slate-600 dark:text-slate-400">
                {t('llm.circuit_breaker_desc')}
              </p>
              <div className="flex items-center gap-1.5 font-medium text-amber-700 dark:text-amber-300 pt-1">
                <AlertTriangle className="w-4 h-4 shrink-0" />
                <span>{t('llm.embedding_homogeneity_warning')}</span>
              </div>
            </div>
          </div>

          {isLoadingChains ? (
            <div className="py-12 flex justify-center">
              <LoadingSpinner size="lg" />
            </div>
          ) : chains.length === 0 ? (
            <div className="p-8 text-center bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 text-slate-500">
              {t('llm.no_chains')}
            </div>
          ) : (
            <div className="space-y-6">
              {chains.map((chain) => (
                <div
                  key={chain.id}
                  className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 shadow-xs overflow-hidden"
                >
                  <div className="p-4 sm:p-5 border-b border-slate-100 dark:border-slate-800/80 bg-slate-50/60 dark:bg-slate-800/40 flex flex-col md:flex-row md:items-center justify-between gap-4">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="min-w-0 break-words font-bold text-base text-slate-900 dark:text-slate-100">
                          {chain.name}
                        </span>
                        <Badge
                          variant={
                            chain.chain_type === 'chat'
                              ? 'primary'
                              : chain.chain_type === 'rerank'
                              ? 'secondary'
                              : 'warning'
                          }
                          size="sm"
                        >
                          {chain.chain_type.toUpperCase()}
                        </Badge>
                        <Badge variant={chain.is_active ? 'success' : 'secondary'} size="sm">
                          {chain.is_active ? t('llm.active_badge') : t('llm.inactive_badge')}
                        </Badge>
                      </div>
                      {chain.description && (
                        <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                          {chain.description}
                        </p>
                      )}
                    </div>

                    <div className="flex flex-wrap items-center gap-3">
                      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-slate-500 dark:text-slate-400 font-mono bg-white dark:bg-slate-800 px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700">
                        <span>{t('llm.stat_threshold')}: <strong className="text-slate-800 dark:text-slate-200">{chain.failure_threshold} {t('llm.stat_fails')}</strong></span>
                        <span>•</span>
                        <span>{t('llm.stat_cooldown')}: <strong className="text-slate-800 dark:text-slate-200">{chain.cooldown_seconds}s</strong></span>
                        <span>•</span>
                        <span>{t('llm.stat_retries')}: <strong className="text-slate-800 dark:text-slate-200">{chain.retry_count}</strong></span>
                      </div>

                      {canManageLLM && (
                        <div className="flex items-center gap-1.5">
                          <button
                            onClick={() => handleOpenAddNode(chain)}
                            className="inline-flex min-h-[44px] items-center gap-1.5 px-3 py-2 rounded-lg bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 hover:bg-indigo-100 dark:hover:bg-indigo-900/50 text-xs font-semibold transition cursor-pointer sm:min-h-0 sm:py-1.5"
                          >
                            <Plus className="w-3.5 h-3.5" />
                            <span>{t('llm.add_node')}</span>
                          </button>
                          <button
                            onClick={() => handleOpenEditChain(chain)}
                            className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-500 transition hover:text-slate-800 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer sm:h-9 sm:w-9"
                            title={t('llm.edit_chain')}
                          >
                            <Edit2 className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => {
                              if (confirm(t('llm.confirm_delete_chain'))) {
                                deleteChainMutation.mutate(chain.id);
                              }
                            }}
                            className="flex h-11 w-11 items-center justify-center rounded-lg text-rose-500 transition hover:text-rose-700 hover:bg-rose-50 dark:hover:bg-rose-950/40 cursor-pointer sm:h-9 sm:w-9"
                            title={t('llm.delete_chain')}
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      )}
                    </div>
                  </div>

                  {chain.chain_type === 'embedding' && chain.nodes.length > 0 && (
                    <div className="px-4 sm:px-5 py-2.5 bg-amber-50/50 dark:bg-amber-950/20 border-b border-amber-100 dark:border-amber-900/30 flex flex-col gap-1.5 sm:flex-row sm:items-center sm:justify-between text-xs text-amber-800 dark:text-amber-300">
                      <div className="flex min-w-0 flex-wrap items-center gap-2">
                        <ShieldAlert className="w-4 h-4 text-amber-600 shrink-0" />
                        <span>{t('llm.fixed_model_architecture')}</span>
                        <code className="break-all font-mono bg-white dark:bg-slate-900 px-2 py-0.5 rounded border border-amber-200 dark:border-amber-800 font-bold">
                          {chain.nodes[0].model_key}
                        </code>
                      </div>
                      <span className="text-[11px] text-amber-700/80 dark:text-amber-400/80">{t('llm.all_nodes_match_key')}</span>
                    </div>
                  )}

                  <div className="divide-y divide-slate-100 dark:divide-slate-800">
                    {chain.nodes.length === 0 ? (
                      <div className="p-6 text-center text-xs text-slate-400">
                        {t('llm.no_nodes')}
                      </div>
                    ) : (
                      chain.nodes.map((node, nodeIdx) => (
                        <div
                          key={node.id}
                          className="px-4 sm:px-5 py-3.5 flex flex-col md:flex-row md:items-center justify-between gap-3 hover:bg-slate-50/50 dark:hover:bg-slate-800/20 transition-colors"
                        >
                          <div className="flex items-center gap-2 sm:gap-4">
                            <div className="flex items-center gap-1">
                              <span className="w-6 h-6 rounded-full bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 text-xs font-bold flex items-center justify-center border border-slate-200 dark:border-slate-700">
                                #{node.priority}
                              </span>
                              {canManageLLM && (
                                <div className="flex flex-col gap-0.5">
                                  <button
                                    disabled={nodeIdx === 0 || updateNodePriorityMutation.isPending}
                                    onClick={() => {
                                      const prevNode = chain.nodes[nodeIdx - 1];
                                      if (prevNode) {
                                        updateNodePriorityMutation.mutate({
                                          chainId: chain.id,
                                          nodeId: node.id,
                                          priority: prevNode.priority,
                                          isActive: node.is_active,
                                        });
                                        updateNodePriorityMutation.mutate({
                                          chainId: chain.id,
                                          nodeId: prevNode.id,
                                          priority: node.priority,
                                          isActive: prevNode.is_active,
                                        });
                                      }
                                    }}
                                    className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-100 hover:text-slate-700 disabled:pointer-events-none disabled:opacity-30 cursor-pointer dark:hover:bg-slate-800 dark:hover:text-slate-200 sm:h-6 sm:w-6 sm:p-0.5"
                                  >
                                    <ArrowUp className="h-4 w-4 sm:h-3 sm:w-3" />
                                  </button>
                                  <button
                                    disabled={nodeIdx === chain.nodes.length - 1 || updateNodePriorityMutation.isPending}
                                    onClick={() => {
                                      const nextNode = chain.nodes[nodeIdx + 1];
                                      if (nextNode) {
                                        updateNodePriorityMutation.mutate({
                                          chainId: chain.id,
                                          nodeId: node.id,
                                          priority: nextNode.priority,
                                          isActive: node.is_active,
                                        });
                                        updateNodePriorityMutation.mutate({
                                          chainId: chain.id,
                                          nodeId: nextNode.id,
                                          priority: node.priority,
                                          isActive: nextNode.is_active,
                                        });
                                      }
                                    }}
                                    className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-100 hover:text-slate-700 disabled:pointer-events-none disabled:opacity-30 cursor-pointer dark:hover:bg-slate-800 dark:hover:text-slate-200 sm:h-6 sm:w-6 sm:p-0.5"
                                  >
                                    <ArrowDown className="h-4 w-4 sm:h-3 sm:w-3" />
                                  </button>
                                </div>
                              )}
                            </div>

                            <div className="min-w-0">
                              <div className="flex flex-wrap items-center gap-2">
                                <span className="font-semibold text-sm text-slate-900 dark:text-slate-100">
                                  {node.model_name}
                                </span>
                                <span className="break-all font-mono text-xs text-slate-500 dark:text-slate-400">
                                  ({node.model_key})
                                </span>
                              </div>
                              <div className="flex flex-wrap items-center gap-x-2 text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                                <span>{t('llm.col_provider')}: <strong className="text-slate-700 dark:text-slate-300">{node.provider_name}</strong></span>
                                <span>•</span>
                                <span className="capitalize">{node.provider_type}</span>
                              </div>
                            </div>
                          </div>

                          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
                            <div className="flex items-center gap-2">
                              {node.health_state === 'open' ? (
                                <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-rose-100 dark:bg-rose-950/60 text-rose-700 dark:text-rose-300 text-xs font-semibold border border-rose-200 dark:border-rose-900">
                                  <Clock className="w-3.5 h-3.5 animate-spin" />
                                  <span>{t('llm.breaker_cooldown')} ({node.remaining_cooldown_seconds}s)</span>
                                </div>
                              ) : node.health_state === 'half_open' ? (
                                <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-amber-100 dark:bg-amber-950/60 text-amber-700 dark:text-amber-300 text-xs font-semibold border border-amber-200 dark:border-amber-900">
                                  <AlertTriangle className="w-3.5 h-3.5" />
                                  <span>{t('llm.breaker_probing')}</span>
                                </div>
                              ) : (
                                <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-emerald-100 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300 text-xs font-semibold border border-emerald-200 dark:border-emerald-900">
                                  <ShieldCheck className="w-3.5 h-3.5" />
                                  <span>{t('llm.breaker_healthy')}</span>
                                </div>
                              )}

                              {node.consecutive_failures > 0 && (
                                <span className="text-xs text-rose-600 dark:text-rose-400 font-medium">
                                  ({node.consecutive_failures} {t('llm.stat_fails')})
                                </span>
                              )}
                            </div>

                            {canManageLLM && (
                              <div className="flex items-center gap-2">
                                <button
                                  onClick={() =>
                                    resetBreakerMutation.mutate({
                                      chainId: chain.id,
                                      nodeId: node.id,
                                    })
                                  }
                                  title={t('llm.reset_breaker')}
                                  className="inline-flex min-h-[44px] items-center justify-center gap-1 rounded-lg px-3 py-2 text-xs font-medium text-slate-600 transition hover:bg-slate-100 cursor-pointer dark:text-slate-300 dark:hover:bg-slate-800 sm:min-h-0 sm:px-2.5 sm:py-1"
                                >
                                  <RotateCcw className="w-3.5 h-3.5" />
                                  <span className="hidden sm:inline">{t('llm.reset_breaker')}</span>
                                </button>
                                <button
                                  onClick={() => {
                                    if (confirm(t('llm.confirm_delete_node'))) {
                                      removeNodeMutation.mutate({
                                        chainId: chain.id,
                                        nodeId: node.id,
                                      });
                                    }
                                  }}
                                  title={t('llm.remove_node')}
                                  className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-400 transition hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 cursor-pointer sm:h-9 sm:w-9"
                                >
                                  <Trash2 className="w-4 h-4" />
                                </button>
                              </div>
                            )}
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {activeTab === 'playground' && !isEmbeddingTabOpen && (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
          <div className="lg:col-span-5 space-y-5">
            <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-4 sm:p-5 shadow-xs space-y-4">
              <div className="flex flex-wrap items-center justify-between gap-3 pb-3 border-b border-slate-100 dark:border-slate-800">
                <div className="flex items-center gap-2">
                  <div className="p-2 rounded-xl bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400">
                    <Sliders className="w-4 h-4" />
                  </div>
                  <div>
                    <h3 className="text-sm font-bold text-slate-900 dark:text-slate-100">
                      {t('llm.playground_config_title')}
                    </h3>
                    <p className="text-[11px] text-slate-400">
                      {t('llm.playground_config_sub')}
                    </p>
                  </div>
                </div>

                <div className="flex rounded-lg bg-slate-100 dark:bg-slate-800 p-0.5">
                  <button
                    type="button"
                    onClick={() => setTestTargetType('model')}
                    className={`min-h-[44px] px-3 py-2 text-xs font-semibold rounded-md transition cursor-pointer sm:min-h-0 sm:px-2.5 sm:py-1 ${
                      testTargetType === 'model'
                        ? 'bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 shadow-xs'
                        : 'text-slate-500 hover:text-slate-900 dark:hover:text-slate-100'
                    }`}
                  >
                    {t('llm.target_model')}
                  </button>
                  <button
                    type="button"
                    onClick={() => setTestTargetType('chain')}
                    className={`min-h-[44px] px-3 py-2 text-xs font-semibold rounded-md transition cursor-pointer sm:min-h-0 sm:px-2.5 sm:py-1 ${
                      testTargetType === 'chain'
                        ? 'bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 shadow-xs'
                        : 'text-slate-500 hover:text-slate-900 dark:hover:text-slate-100'
                    }`}
                  >
                    {t('llm.target_chain')}
                  </button>
                </div>
              </div>

              {testTargetType === 'model' ? (
                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                    {t('llm.select_test_model')}
                  </label>
                  <select
                    value={selectedTestModelId || (models[0]?.id || '')}
                    onChange={(e) => setSelectedTestModelId(e.target.value)}
                    className="w-full px-3 py-2.5 text-base rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-indigo-500 focus:outline-hidden sm:py-2 sm:text-sm"
                  >
                    {models.map((m) => (
                      <option key={m.id} value={m.id}>
                        {m.name} ({m.model_key}) - {m.provider_name || m.provider_id} [{m.model_type}]
                      </option>
                    ))}
                  </select>
                </div>
              ) : (
                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                    {t('llm.select_test_chain')}
                  </label>
                  <select
                    value={selectedTestChainId || (chains[0]?.id || '')}
                    onChange={(e) => setSelectedTestChainId(e.target.value)}
                    className="w-full px-3 py-2.5 text-base rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-indigo-500 focus:outline-hidden sm:py-2 sm:text-sm"
                  >
                    {chains.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name} ({c.chain_type}) - {t('llm.chain_option_meta', { count: c.nodes.length, threshold: c.failure_threshold })}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                  {t('llm.quick_sample_prompts')}
                </label>
                <div className="flex flex-wrap gap-1.5">
                  <button
                    type="button"
                    onClick={() =>
                      setTestPrompt(t('llm.prompt_sample_greeting'))
                    }
                    className="inline-flex min-h-[44px] items-center rounded-lg px-3 py-2 text-xs bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-indigo-50 dark:hover:bg-indigo-950/50 hover:text-indigo-600 dark:hover:text-indigo-400 transition cursor-pointer sm:min-h-0 sm:px-2.5 sm:py-1"
                  >
                    {t('llm.prompt_tag_greeting')}
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      setTestPrompt(t('llm.prompt_sample_credits'))
                    }
                    className="inline-flex min-h-[44px] items-center rounded-lg px-3 py-2 text-xs bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-indigo-50 dark:hover:bg-indigo-950/50 hover:text-indigo-600 dark:hover:text-indigo-400 transition cursor-pointer sm:min-h-0 sm:px-2.5 sm:py-1"
                  >
                    {t('llm.prompt_tag_credits')}
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      setTestPrompt(t('llm.prompt_sample_thesis'))
                    }
                    className="inline-flex min-h-[44px] items-center rounded-lg px-3 py-2 text-xs bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-indigo-50 dark:hover:bg-indigo-950/50 hover:text-indigo-600 dark:hover:text-indigo-400 transition cursor-pointer sm:min-h-0 sm:px-2.5 sm:py-1"
                  >
                    {t('llm.prompt_tag_thesis')}
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      setTestPrompt(t('llm.prompt_sample_cpa'))
                    }
                    className="inline-flex min-h-[44px] items-center rounded-lg px-3 py-2 text-xs bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-indigo-50 dark:hover:bg-indigo-950/50 hover:text-indigo-600 dark:hover:text-indigo-400 transition cursor-pointer sm:min-h-0 sm:px-2.5 sm:py-1"
                  >
                    {t('llm.prompt_tag_cpa')}
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      setTestPrompt(t('llm.prompt_sample_reasoning'))
                    }
                    className="inline-flex min-h-[44px] items-center rounded-lg px-3 py-2 text-xs bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-indigo-50 dark:hover:bg-indigo-950/50 hover:text-indigo-600 dark:hover:text-indigo-400 transition cursor-pointer sm:min-h-0 sm:px-2.5 sm:py-1"
                  >
                    {t('llm.prompt_tag_reasoning')}
                  </button>
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                  {t('llm.user_prompt_label')}
                </label>
                <textarea
                  rows={4}
                  value={testPrompt}
                  onChange={(e) => setTestPrompt(e.target.value)}
                  placeholder={t('llm.user_prompt_placeholder')}
                  className="w-full px-3 py-2.5 text-base rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-indigo-500 focus:outline-hidden leading-relaxed resize-y sm:py-2 sm:text-sm"
                />
              </div>

              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="text-xs font-semibold text-slate-700 dark:text-slate-300 flex items-center gap-1.5">
                    <ImageIcon className="w-3.5 h-3.5 text-slate-400" />
                    <span>{t('llm.attached_images_label')}</span>
                  </label>
                  <input
                    ref={testFileInputRef}
                    type="file"
                    accept="image/*"
                    className="hidden"
                    onChange={handleTestFileChange}
                  />
                  <button
                    type="button"
                    onClick={() => testFileInputRef.current?.click()}
                    className="inline-flex min-h-[44px] items-center text-xs text-indigo-600 dark:text-indigo-400 hover:underline font-medium cursor-pointer sm:min-h-0"
                  >
                    {t('llm.btn_select_image')}
                  </button>
                </div>

                {testImages.length > 0 && (
                  <div className="flex gap-2 p-2 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 overflow-x-auto">
                    {testImages.map((img, i) => (
                      <div
                        key={i}
                        className="relative group shrink-0 h-20 w-20 rounded-lg overflow-hidden border border-slate-300 dark:border-slate-700 shadow-xs sm:h-16 sm:w-16"
                      >
                        <img src={img} alt={t('llm.test_image_alt')} className="w-full h-full object-cover" />
                        <button
                          type="button"
                          onClick={() => setTestImages((prev) => prev.filter((_, idx) => idx !== i))}
                          className="absolute right-0 top-0 flex h-11 w-11 items-center justify-center bg-black/70 hover:bg-black text-white rounded-full cursor-pointer sm:h-7 sm:w-7 sm:p-1"
                        >
                          <X className="h-4 w-4 sm:h-3 sm:w-3" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="pt-2 border-t border-slate-100 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setIsAdvancedOpen(!isAdvancedOpen)}
                  className="w-full flex items-center justify-between text-xs font-semibold text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100 cursor-pointer"
                >
                  <span>{t('llm.advanced_params_label')}</span>
                  <span>{isAdvancedOpen ? '▲' : '▼'}</span>
                </button>

                {isAdvancedOpen && (
                  <div className="mt-3 space-y-3 p-3 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 animate-in fade-in">
                    <div>
                      <label className="block text-[11px] font-semibold text-slate-700 dark:text-slate-300 mb-1">
                        {t('llm.system_prompt')}
                      </label>
                      <textarea
                        rows={2}
                        value={testSystemPrompt}
                        onChange={(e) => setTestSystemPrompt(e.target.value)}
                        className="w-full px-3 py-2 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-hidden focus:ring-1 focus:ring-indigo-500 sm:px-2.5 sm:py-1.5 sm:text-xs"
                      />
                    </div>

                    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                      <div>
                        <div className="flex justify-between text-[11px] font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          <span>{t('llm.temperature')}</span>
                          <span className="font-mono">{testTemperature}</span>
                        </div>
                        <input
                          type="range"
                          min={0}
                          max={1}
                          step={0.1}
                          value={testTemperature}
                          onChange={(e) => setTestTemperature(parseFloat(e.target.value))}
                          className="w-full"
                        />
                      </div>

                      <div>
                        <div className="flex justify-between text-[11px] font-semibold text-slate-700 dark:text-slate-300 mb-1">
                          <span>{t('llm.max_tokens')}</span>
                          <span className="font-mono">{testMaxTokens}</span>
                        </div>
                        <input
                          type="number"
                          min={64}
                          max={8192}
                          step={128}
                          value={testMaxTokens}
                          onChange={(e) => setTestMaxTokens(parseInt(e.target.value, 10) || 1024)}
                          className="w-full px-3 py-2 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 sm:px-2 sm:py-1 sm:text-xs"
                        />
                      </div>
                    </div>
                  </div>
                )}
              </div>

              <button
                type="button"
                onClick={handleExecuteTest}
                disabled={isTesting || !testPrompt.trim()}
                className="w-full py-3 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-sm shadow-md shadow-indigo-600/20 transition flex items-center justify-center gap-2 disabled:opacity-50 cursor-pointer"
              >
                {isTesting ? (
                  <>
                    <LoadingSpinner size="sm" />
                    <span>{t('llm.executing_test')}</span>
                  </>
                ) : (
                  <>
                    <Play className="w-4 h-4 fill-white" />
                    <span>{t('llm.run_test_btn')}</span>
                  </>
                )}
              </button>
            </div>
          </div>

          <div className="lg:col-span-7 space-y-4">
            {testError && (
              <div className="p-4 rounded-2xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900/60 text-rose-800 dark:text-rose-200 flex items-start gap-3">
                <AlertTriangle className="w-5 h-5 text-rose-600 dark:text-rose-400 shrink-0 mt-0.5" />
                <div>
                  <h4 className="font-bold text-sm">{t('llm.test_failed_title')}</h4>
                  <p className="text-xs mt-1 leading-relaxed">{testError}</p>
                </div>
              </div>
            )}

            {!testResult && !isTesting && !testError && (
              <div className="p-8 sm:p-12 text-center bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col items-center justify-center min-h-[280px] sm:min-h-[460px]">
                <div className="w-16 h-16 rounded-2xl bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400 flex items-center justify-center mb-4 shadow-sm">
                  <Terminal className="w-8 h-8" />
                </div>
                <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">
                  {t('llm.playground_empty_title')}
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400 max-w-md mt-1.5 leading-relaxed">
                  {t('llm.playground_empty_desc')}
                </p>
              </div>
            )}

            {isTesting && (
              <div className="p-8 sm:p-12 text-center bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col items-center justify-center min-h-[280px] sm:min-h-[460px]">
                <LoadingSpinner size="lg" />
                <h4 className="text-sm font-bold text-slate-900 dark:text-slate-100 mt-4">
                  {t('llm.connecting_provider')}
                </h4>
                <p className="text-xs text-slate-400 mt-1">
                  {t('llm.processing_prompt')}
                </p>
              </div>
            )}

            {testResult && !isTesting && (
              <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
                <div className="p-4 bg-slate-50/70 dark:bg-slate-800/40 border-b border-slate-100 dark:border-slate-800 flex flex-wrap items-center justify-between gap-3">
                  <div className="flex flex-wrap items-center gap-2">
                    <span
                      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold ${
                        testResult.success
                          ? 'bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800'
                          : 'bg-rose-50 dark:bg-rose-950/60 text-rose-700 dark:text-rose-300 border border-rose-200 dark:border-rose-800'
                      }`}
                    >
                      <span className={`w-2 h-2 rounded-full ${testResult.success ? 'bg-emerald-500' : 'bg-rose-500'}`} />
                      <span>{testResult.success ? t('llm.result_status_ok') : t('llm.result_status_error')}</span>
                    </span>

                    <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold bg-amber-50 dark:bg-amber-950/60 text-amber-700 dark:text-amber-300 border border-amber-200 dark:border-amber-800">
                      <span>⚡ {testResult.latency_ms} ms</span>
                    </span>

                    {testResult.total_tokens ? (
                      <span className="break-words text-xs font-mono text-slate-500 dark:text-slate-400">
                        {t('llm.tokens_breakdown', { total: testResult.total_tokens, prompt: testResult.prompt_tokens, completion: testResult.completion_tokens })}
                      </span>
                    ) : null}
                  </div>

                  <div className="flex rounded-lg bg-slate-200/80 dark:bg-slate-800 p-0.5 text-xs font-semibold">
                    <button
                      type="button"
                      onClick={() => setActiveResultTab('output')}
                      className={`min-h-[44px] px-3 py-2 rounded-md transition cursor-pointer sm:min-h-0 sm:py-1 ${
                        activeResultTab === 'output'
                          ? 'bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 shadow-xs'
                          : 'text-slate-600 dark:text-slate-400'
                      }`}
                    >
                      {t('llm.tab_formatted')}
                    </button>
                    <button
                      type="button"
                      onClick={() => setActiveResultTab('raw')}
                      className={`min-h-[44px] px-3 py-2 rounded-md transition cursor-pointer sm:min-h-0 sm:py-1 ${
                        activeResultTab === 'raw'
                          ? 'bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 shadow-xs'
                          : 'text-slate-600 dark:text-slate-400'
                      }`}
                    >
                      {t('llm.tab_raw_json')}
                    </button>
                  </div>
                </div>

                <div className="px-4 py-2.5 bg-indigo-50/40 dark:bg-indigo-950/20 border-b border-indigo-100/50 dark:border-indigo-900/30 flex flex-wrap items-center justify-between gap-2 text-xs">
                  <div className="flex min-w-0 flex-wrap items-center gap-2">
                    <span className="font-semibold text-indigo-900 dark:text-indigo-200">
                      {t('llm.model_reply_label')}
                    </span>
                    <span className="break-all font-mono text-indigo-700 dark:text-indigo-400 font-bold">
                      {testResult.node_used || testResult.model_name || testResult.model_key}
                    </span>
                    {testResult.provider_name && (
                      <span className="break-all text-slate-400 font-mono">
                        {t('llm.via_provider', { provider: testResult.provider_name })}
                      </span>
                    )}
                  </div>
                  {testResult.finish_reason && (
                    <span className="px-1.5 py-0.5 rounded text-[10px] bg-slate-200 dark:bg-slate-800 text-slate-600 dark:text-slate-400 font-mono">
                      {t('llm.finish_reason_label', { reason: testResult.finish_reason })}
                    </span>
                  )}
                </div>

                <div className="p-5">
                  {activeResultTab === 'output' ? (
                    <div className="space-y-4">
                      {testResult.reasoning_content && (
                        <div className="p-3.5 rounded-xl bg-violet-50/60 dark:bg-violet-950/30 border border-violet-100 dark:border-violet-900/40 space-y-1.5">
                          <div className="text-xs font-bold text-violet-800 dark:text-violet-300 flex items-center gap-1.5">
                            <Sparkles className="w-3.5 h-3.5" />
                            <span>{t('llm.reasoning_label')}</span>
                          </div>
                          <p className="text-xs text-violet-700/90 dark:text-violet-300/90 whitespace-pre-wrap leading-relaxed font-mono">
                            {testResult.reasoning_content}
                          </p>
                        </div>
                      )}

                      <div className="text-sm leading-relaxed text-slate-800 dark:text-slate-100 whitespace-pre-wrap font-sans">
                        {testResult.content}
                      </div>
                    </div>
                  ) : (
                    <div className="relative">
                      <button
                        type="button"
                        onClick={() => {
                          navigator.clipboard.writeText(JSON.stringify(testResult, null, 2));
                          setCopiedRaw(true);
                          setTimeout(() => setCopiedRaw(false), 2000);
                        }}
                        className="absolute top-2 right-2 inline-flex min-h-[44px] items-center gap-1 rounded-lg px-3 py-2 text-xs bg-slate-700 hover:bg-slate-600 text-white font-medium transition shadow-xs z-10 cursor-pointer sm:min-h-0 sm:px-2.5 sm:py-1"
                      >
                        {copiedRaw ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                        <span>{copiedRaw ? t('llm.copied_json') : t('llm.copy_json')}</span>
                      </button>
                      <pre className="p-4 rounded-xl bg-slate-900 text-slate-100 text-xs font-mono overflow-x-auto max-h-[300px] sm:max-h-[460px] leading-relaxed">
                        {JSON.stringify(testResult, null, 2)}
                      </pre>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {isEmbeddingTabOpen && (
        <div className="space-y-6">
          {toastMessage && (
            <div className="fixed inset-x-4 bottom-24 z-50 flex items-center gap-2 rounded-xl bg-slate-900 text-white dark:bg-white dark:text-slate-900 px-4 py-3 shadow-lg text-xs font-medium border border-slate-700 dark:border-slate-200 animate-in fade-in slide-in-from-bottom-2 duration-200 sm:inset-x-auto sm:right-5 sm:max-w-sm lg:bottom-5">
              {toastTone === 'success' ? (
                <CheckCircle2 className="w-4 h-4 text-emerald-400 dark:text-emerald-600 shrink-0" />
              ) : (
                <AlertTriangle className="w-4 h-4 text-rose-400 dark:text-rose-600 shrink-0" />
              )}
              <span className="break-words">{toastMessage}</span>
            </div>
          )}

          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2 lg:gap-5">
            <div className="p-4 sm:p-5 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs space-y-4">
              <div className="flex items-center gap-3">
                <div className="shrink-0 p-2 rounded-xl bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 border border-indigo-100 dark:border-indigo-900/50">
                  <Server className="w-5 h-5" />
                </div>
                <div className="min-w-0">
                  <h3 className="text-sm font-bold text-slate-900 dark:text-slate-100">
                    {t('embedding.provider_title')}
                  </h3>
                  <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                    {t('embedding.provider_desc')}
                  </p>
                </div>
              </div>

              {isLoadingEmbeddingSettings ? (
                <div className="py-6 flex justify-center">
                  <LoadingSpinner size="md" />
                </div>
              ) : !embeddingSettings ? (
                <div className="p-3 rounded-lg bg-rose-50 dark:bg-rose-950/40 text-rose-600 dark:text-rose-400 text-xs">
                  {t('embedding.load_failed')}
                </div>
              ) : canManageLLM ? (
                <>
                  <div className="flex w-full rounded-lg bg-slate-100 dark:bg-slate-800 p-0.5">
                    <button
                      type="button"
                      onClick={() => setProviderDraft('api')}
                      className={`min-h-[44px] flex-1 px-3 py-2 text-xs font-semibold rounded-md transition cursor-pointer sm:flex-none sm:px-4 ${
                        effectiveProvider === 'api'
                          ? 'bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 shadow-xs'
                          : 'text-slate-500 hover:text-slate-900 dark:hover:text-slate-100'
                      }`}
                    >
                      {t('embedding.provider_api_label')}
                    </button>
                    <button
                      type="button"
                      onClick={() => setProviderDraft('onnx')}
                      className={`min-h-[44px] flex-1 px-3 py-2 text-xs font-semibold rounded-md transition cursor-pointer sm:flex-none sm:px-4 ${
                        effectiveProvider === 'onnx'
                          ? 'bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 shadow-xs'
                          : 'text-slate-500 hover:text-slate-900 dark:hover:text-slate-100'
                      }`}
                    >
                      {t('embedding.provider_onnx_label')}
                    </button>
                  </div>

                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    {effectiveProvider === 'api'
                      ? t('embedding.provider_api_hint')
                      : t('embedding.provider_onnx_hint')}
                  </p>

                  {effectiveProvider === 'api' && (
                    <div className="p-3.5 rounded-lg bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700 space-y-3">
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                          {t('embedding.api_models_title')} ({apiEmbeddingModels.length})
                        </span>
                        <button
                          type="button"
                          onClick={() => {
                            setActiveTab('models');
                            setIsEmbeddingTabOpen(false);
                          }}
                          className="text-[11px] text-indigo-600 dark:text-indigo-400 hover:underline font-medium cursor-pointer"
                        >
                          {t('embedding.view_models_tab')}
                        </button>
                      </div>

                      {apiEmbeddingModels.length === 0 ? (
                        <p className="text-xs text-amber-600 dark:text-amber-400">
                          {t('embedding.api_models_empty')}
                        </p>
                      ) : (
                        <div className="space-y-1.5">
                          {apiEmbeddingModels.map((m) => {
                            const providerName =
                              providers.find((p) => p.id === m.provider_id)?.name || m.provider_id;
                            return (
                              <div
                                key={m.id}
                                className="flex items-center justify-between text-xs py-1.5 px-2.5 rounded bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800"
                              >
                                <div className="min-w-0">
                                  <span className="font-semibold text-slate-900 dark:text-slate-100">
                                    {m.name}
                                  </span>
                                  <span className="ml-2 font-mono text-[11px] text-slate-400">
                                    {m.model_key}
                                  </span>
                                </div>
                                <Badge variant="secondary" size="sm">
                                  {providerName}
                                </Badge>
                              </div>
                            );
                          })}
                        </div>
                      )}

                      <div className="pt-2 border-t border-slate-200 dark:border-slate-700/60">
                        <div className="flex items-center justify-between gap-2 mb-1">
                          <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                            {t('embedding.api_chain_status')}
                          </span>
                          <button
                            type="button"
                            onClick={() => {
                              setActiveTab('chains');
                              setIsEmbeddingTabOpen(false);
                            }}
                            className="text-[11px] text-indigo-600 dark:text-indigo-400 hover:underline font-medium cursor-pointer"
                          >
                            {t('embedding.configure_chains_btn')}
                          </button>
                        </div>
                        {activeEmbeddingChain && activeEmbeddingChain.nodes.length > 0 ? (
                          <p className="text-xs text-emerald-600 dark:text-emerald-400">
                            {t('embedding.api_chain_configured', {
                              name: activeEmbeddingChain.name,
                              count: activeEmbeddingChain.nodes.length,
                            })}
                          </p>
                        ) : (
                          <p className="text-xs text-amber-600 dark:text-amber-400">
                            {t('embedding.api_chain_missing')}
                          </p>
                        )}
                      </div>
                    </div>
                  )}

                  {effectiveProvider === 'onnx' && (
                    <div className="p-3 rounded-lg bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700 space-y-1.5">
                      <div className="flex items-center justify-between gap-2 text-xs">
                        <span className="shrink-0 font-medium text-slate-500">
                          {t('embedding.active_model_label')}
                        </span>
                        <span className="min-w-0 truncate font-mono text-slate-700 dark:text-slate-300">
                          {activeOnnxModelName || t('embedding.no_active_model')}
                        </span>
                      </div>
                      {okModelCount === 0 && (
                        <div className="flex items-start gap-1.5 text-amber-700 dark:text-amber-300">
                          <AlertTriangle className="w-3.5 h-3.5 shrink-0 mt-0.5" />
                          <span>{t('embedding.onnx_needs_model_hint')}</span>
                        </div>
                      )}
                    </div>
                  )}

                  <button
                    type="button"
                    onClick={() => saveEmbeddingSettingsMutation.mutate()}
                    disabled={
                      effectiveProvider === embeddingSettings.provider ||
                      saveEmbeddingSettingsMutation.isPending
                    }
                    className="inline-flex min-h-[44px] w-full items-center justify-center gap-2 rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-md shadow-indigo-600/20 transition hover:bg-indigo-700 disabled:opacity-50 disabled:pointer-events-none cursor-pointer sm:w-auto sm:py-2"
                  >
                    {saveEmbeddingSettingsMutation.isPending && <LoadingSpinner size="sm" />}
                    <span>{t('common.save')}</span>
                  </button>
                </>
              ) : (
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs font-medium text-slate-500">
                    {t('embedding.active_provider_label')}
                  </span>
                  <Badge variant="primary" size="sm">
                    {effectiveProvider === 'onnx'
                      ? t('embedding.provider_onnx_label')
                      : t('embedding.provider_api_label')}
                  </Badge>
                </div>
              )}
            </div>

            <div className="p-4 sm:p-5 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs space-y-4">
              <div className="flex items-center gap-3">
                <div className="shrink-0 p-2 rounded-xl bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 border border-indigo-100 dark:border-indigo-900/50">
                  <Terminal className="w-5 h-5" />
                </div>
                <div className="min-w-0">
                  <h3 className="text-sm font-bold text-slate-900 dark:text-slate-100">
                    {t('embedding.runtime_title')}
                  </h3>
                  <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                    {t('embedding.runtime_desc')}
                  </p>
                </div>
              </div>

              {isLoadingEmbeddingSettings ? (
                <div className="py-6 flex justify-center">
                  <LoadingSpinner size="md" />
                </div>
              ) : !embeddingRuntime ? (
                <div className="p-3 rounded-lg bg-rose-50 dark:bg-rose-950/40 text-rose-600 dark:text-rose-400 text-xs">
                  {t('embedding.load_failed')}
                </div>
              ) : (
                <>
                  <div className="space-y-2 text-xs text-slate-600 dark:text-slate-400">
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-medium text-slate-500">{t('embedding.runtime_platform')}</span>
                      <code className="px-2 py-0.5 rounded bg-slate-100 dark:bg-slate-800 font-mono text-slate-700 dark:text-slate-300">
                        {embeddingRuntime.os}/{embeddingRuntime.arch}
                      </code>
                    </div>
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-medium text-slate-500">{t('common.status')}</span>
                      <Badge variant={embeddingRuntime.lib_present ? 'success' : 'danger'} size="sm">
                        {embeddingRuntime.lib_present
                          ? t('embedding.runtime_lib_present')
                          : t('embedding.runtime_lib_missing')}
                      </Badge>
                    </div>
                    {embeddingRuntime.lib_path && (
                      <div className="flex flex-col gap-1">
                        <span className="font-medium text-slate-500">
                          {t('embedding.runtime_expected_path')}
                        </span>
                        <code className="break-all px-2 py-1 rounded bg-slate-100 dark:bg-slate-800 font-mono text-[11px] text-slate-700 dark:text-slate-300">
                          {embeddingRuntime.lib_path}
                        </code>
                      </div>
                    )}
                  </div>

                  <div className="pt-4 border-t border-slate-100 dark:border-slate-800 space-y-3">
                    <div className="flex items-center gap-2">
                      <Download className="w-4 h-4 text-indigo-600 dark:text-indigo-400 shrink-0" />
                      <h4 className="text-xs font-bold text-slate-900 dark:text-slate-100">
                        {t('embedding.runtime_download_title')}
                      </h4>
                    </div>
                    <p className="text-xs text-slate-500 dark:text-slate-400">
                      {t('embedding.runtime_download_desc')}
                    </p>

                    {matchedRuntimes.length === 0 ? (
                      <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300 text-xs">
                        <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
                        <span>{t('embedding.runtime_unsupported')}</span>
                      </div>
                    ) : (
                      <div className="space-y-2">
                        {matchedRuntimes.map((rt) => {
                          const runtimeJob = downloadJobs.find((j) => j.preset_id === rt.preset_id);
                          const isRuntimeRunning = runtimeJob?.state === 'running';
                          const isInstalled = !!embeddingRuntime.lib_present;
                          const totalBytes = runtimeJob?.total_bytes || 0;
                          const percent =
                            totalBytes > 0 && runtimeJob
                              ? Math.min(100, Math.round((runtimeJob.downloaded_bytes / totalBytes) * 100))
                              : 0;

                          return (
                            <div
                              key={rt.preset_id}
                              className="p-3 rounded-lg border border-slate-200 dark:border-slate-800 space-y-2"
                            >
                              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                                <div className="min-w-0">
                                  <div className="flex items-center gap-2">
                                    <span className="text-xs font-semibold text-slate-900 dark:text-slate-100 break-words">
                                      {rt.display_name}
                                    </span>
                                    {isInstalled && (
                                      <Badge variant="success" size="sm">
                                        {t('embedding.runtime_installed')}
                                      </Badge>
                                    )}
                                  </div>
                                  <span className="block truncate text-[11px] font-mono text-slate-400">
                                    {rt.archive}
                                  </span>
                                </div>
                                <div className="flex items-center gap-2 shrink-0">
                                  <span className="text-[11px] text-slate-500 dark:text-slate-400">
                                    {formatBytes(rt.approx_size_bytes || rt.size_bytes)}
                                  </span>
                                  {canManageLLM && (
                                    <button
                                      type="button"
                                      onClick={() => startDownloadMutation.mutate(rt.preset_id)}
                                      disabled={startDownloadMutation.isPending || isRuntimeRunning}
                                      className={`inline-flex min-h-[44px] items-center gap-1.5 px-3 py-2 rounded-lg text-xs font-semibold transition cursor-pointer disabled:opacity-50 sm:min-h-0 sm:py-1.5 ${
                                        isInstalled
                                          ? 'bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300'
                                          : 'bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 hover:bg-indigo-100 dark:hover:bg-indigo-900/50'
                                      }`}
                                    >
                                      {isRuntimeRunning ? (
                                        <>
                                          <LoadingSpinner size="sm" />
                                          <span>{t('embedding.downloading')}</span>
                                        </>
                                      ) : isInstalled ? (
                                        <>
                                          <RefreshCw className="w-3.5 h-3.5" />
                                          <span>{t('embedding.runtime_reinstall')}</span>
                                        </>
                                      ) : (
                                        <>
                                          <Download className="w-3.5 h-3.5" />
                                          <span>{t('embedding.download_action')}</span>
                                        </>
                                      )}
                                    </button>
                                  )}
                                </div>
                              </div>

                              {isRuntimeRunning && runtimeJob && (
                                <div className="space-y-1 pt-1">
                                  <div className="h-1.5 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800">
                                    <div
                                      className={`h-full rounded-full bg-indigo-600 transition-all duration-300 ${
                                        totalBytes > 0 ? '' : 'animate-pulse w-2/5'
                                      }`}
                                      style={totalBytes > 0 ? { width: `${percent}%` } : undefined}
                                    />
                                  </div>
                                  <span className="block text-[11px] text-slate-500 dark:text-slate-400 font-mono">
                                    {totalBytes > 0
                                      ? t('embedding.download_progress', {
                                          downloaded: formatBytes(runtimeJob.downloaded_bytes),
                                          total: formatBytes(totalBytes),
                                          percent,
                                        })
                                      : t('embedding.download_progress_unknown', {
                                          downloaded: formatBytes(runtimeJob.downloaded_bytes),
                                        })}
                                  </span>
                                </div>
                              )}

                              {runtimeJob?.state === 'error' && runtimeJob.error && (
                                <p className="text-[11px] text-rose-600 dark:text-rose-400 pt-1">
                                  {runtimeJob.error}
                                </p>
                              )}
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </div>
                </>
              )}
            </div>
          </div>

          <div className="space-y-4">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div className="min-w-0">
                <h3 className="text-sm font-bold text-slate-900 dark:text-slate-100">
                  {t('embedding.models_title')}
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  {t('embedding.models_desc')}
                </p>
              </div>
              {canManageLLM && (
                <button
                  type="button"
                  onClick={() => setIsDownloadModalOpen(true)}
                  className="inline-flex min-h-[44px] w-full items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-sm shadow-md shadow-indigo-600/20 transition sm:w-auto sm:py-2"
                >
                  <Download className="w-4 h-4" />
                  <span>{t('embedding.download_open_btn')}</span>
                </button>
              )}
            </div>

            {isLoadingOnnxModels ? (
              <div className="py-12 flex justify-center">
                <LoadingSpinner size="lg" />
              </div>
            ) : onnxModels.length === 0 ? (
              <div className="p-8 text-center bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 text-slate-500 text-sm">
                {t('embedding.models_empty')}
              </div>
            ) : (
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
                {onnxModels.map((m) => (
                  <div
                    key={m.model_id}
                    className={`p-4 rounded-xl border flex flex-col gap-3 ${
                      m.is_active
                        ? 'border-indigo-300 dark:border-indigo-800 ring-1 ring-indigo-500/30 bg-indigo-50/40 dark:bg-indigo-950/20'
                        : 'bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-xs'
                    }`}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="min-w-0 break-words font-bold text-base text-slate-900 dark:text-slate-100">
                            {m.display_name}
                          </span>
                          {m.is_active && (
                            <Badge variant="primary" size="sm">
                              {t('embedding.badge_active')}
                            </Badge>
                          )}
                        </div>
                        <span className="block truncate text-xs font-mono text-slate-400" title={m.model_id}>
                          {m.model_id}
                        </span>
                      </div>
                      <Badge variant={ONNX_STATUS_VARIANTS[m.status]} size="sm">
                        {t(`embedding.status_${m.status}`)}
                      </Badge>
                    </div>

                    <div className="min-w-0 text-xs space-y-1.5 text-slate-600 dark:text-slate-400">
                      <div className="flex items-center justify-between gap-2">
                        <span className="font-medium text-slate-500">
                          {t('embedding.model_dims', { dim: m.dim })}
                        </span>
                        <span className="font-mono">{formatBytes(m.size_bytes)}</span>
                      </div>
                      {m.sha256 && (
                        <div className="flex items-center gap-1.5 min-w-0">
                          <span className="shrink-0 font-medium text-slate-500">
                            {t('embedding.sha_label')}
                          </span>
                          <span
                            className="min-w-0 truncate font-mono text-[11px] text-slate-700 dark:text-slate-300"
                            title={m.sha256}
                          >
                            {m.sha256}
                          </span>
                          {canManageLLM && (
                            <button
                              type="button"
                              onClick={() => {
                                navigator.clipboard.writeText(m.sha256 || '');
                                setCopiedModelId(m.model_id);
                              }}
                              title={copiedModelId === m.model_id ? t('embedding.copied') : t('embedding.copy_sha')}
                              className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-100 hover:text-indigo-600 dark:hover:bg-slate-800 sm:h-7 sm:w-7"
                            >
                              {copiedModelId === m.model_id ? (
                                <Check className="w-3.5 h-3.5 text-emerald-500" />
                              ) : (
                                <Copy className="w-3.5 h-3.5" />
                              )}
                            </button>
                          )}
                        </div>
                      )}
                      {m.validated_at && (
                        <div className="text-[11px] text-slate-500 dark:text-slate-400">
                          {t('embedding.validated_at', { time: formatDateTime(m.validated_at) })}
                        </div>
                      )}
                      {m.status === 'invalid' && m.error && (
                        <div className="p-2 rounded-lg bg-rose-50 dark:bg-rose-950/40 text-rose-600 dark:text-rose-400 break-words">
                          <span className="font-semibold">{t('embedding.model_error_label')}: </span>
                          {m.error}
                        </div>
                      )}
                    </div>

                    {canManageLLM && (
                      <div className="mt-auto pt-3 border-t border-slate-100 dark:border-slate-800 flex flex-wrap items-center gap-2">
                        <button
                          type="button"
                          onClick={() => validateModelMutation.mutate(m.model_id)}
                          disabled={
                            validateModelMutation.isPending &&
                            validateModelMutation.variables === m.model_id
                          }
                          className="inline-flex min-h-[44px] flex-1 sm:flex-none items-center justify-center gap-1.5 px-3 py-2 rounded-lg bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 hover:bg-indigo-100 dark:hover:bg-indigo-900/50 text-xs font-semibold transition cursor-pointer disabled:opacity-50 sm:min-h-0 sm:py-1.5"
                        >
                          {validateModelMutation.isPending && validateModelMutation.variables === m.model_id ? (
                            <LoadingSpinner size="sm" />
                          ) : (
                            <ShieldCheck className="w-3.5 h-3.5" />
                          )}
                          <span>
                            {validateModelMutation.isPending && validateModelMutation.variables === m.model_id
                              ? t('embedding.validating')
                              : t('embedding.validate_btn')}
                          </span>
                        </button>

                        {m.status === 'ok' && !m.is_active && (
                          <button
                            type="button"
                            onClick={() => activateModelMutation.mutate(m.model_id)}
                            disabled={activateModelMutation.isPending}
                            className="inline-flex min-h-[44px] flex-1 sm:flex-none items-center justify-center gap-1.5 px-3 py-2 rounded-lg bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-100 dark:hover:bg-emerald-900/50 text-xs font-semibold transition cursor-pointer disabled:opacity-50 sm:min-h-0 sm:py-1.5"
                          >
                            <CheckCircle2 className="w-3.5 h-3.5" />
                            <span>{t('embedding.activate_btn')}</span>
                          </button>
                        )}

                        <button
                          type="button"
                          onClick={() => setDeleteTarget(m)}
                          disabled={m.is_active}
                          title={m.is_active ? t('embedding.delete_active_hint') : t('common.delete')}
                          className="inline-flex min-h-[44px] items-center justify-center gap-1.5 px-3 py-2 rounded-lg text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/40 text-xs font-semibold transition cursor-pointer disabled:opacity-40 disabled:pointer-events-none sm:min-h-0 sm:py-1.5 sm:ml-auto"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                          <span>{t('common.delete')}</span>
                        </button>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}

            <div className="p-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/60 dark:bg-slate-800/40 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div className="min-w-0 space-y-2 text-xs text-slate-600 dark:text-slate-400">
                <div className="flex items-center gap-2 font-bold text-slate-700 dark:text-slate-200">
                  <FolderOpen className="w-4 h-4 shrink-0" />
                  <span>{t('embedding.manual_title')}</span>
                </div>
                <p className="leading-relaxed">{t('embedding.manual_desc')}</p>
                <code className="inline-block break-all px-2 py-1 rounded bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 font-mono text-[11px] text-slate-700 dark:text-slate-300">
                  data/onnx/&lt;model_id&gt;/
                </code>
                <div className="flex flex-wrap items-center gap-1.5">
                  <span className="font-medium text-slate-500">{t('embedding.manual_files_label')}</span>
                  <code className="px-1.5 py-0.5 rounded bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 font-mono text-[11px] text-slate-700 dark:text-slate-300">
                    model.onnx
                  </code>
                  <code className="px-1.5 py-0.5 rounded bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 font-mono text-[11px] text-slate-700 dark:text-slate-300">
                    vocab.txt
                  </code>
                  <code className="px-1.5 py-0.5 rounded bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 font-mono text-[11px] text-slate-700 dark:text-slate-300">
                    manifest.json
                  </code>
                </div>
              </div>
              {canManageLLM && (
                <button
                  type="button"
                  onClick={() => {
                    void refetchOnnxModels().then((res) => {
                      if (res.isSuccess) {
                        setToast(t('embedding.rescan_done'), 'success');
                      } else {
                        setToast(t('embedding.rescan_failed'), 'error');
                      }
                    });
                  }}
                  disabled={isRescanningOnnxModels}
                  className="inline-flex min-h-[44px] w-full sm:w-auto shrink-0 items-center justify-center gap-1.5 px-3 py-2 rounded-lg bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-300 hover:text-indigo-600 dark:hover:text-indigo-400 hover:border-indigo-300 text-xs font-semibold transition cursor-pointer disabled:opacity-50 sm:min-h-0 sm:py-1.5"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${isRescanningOnnxModels ? 'animate-spin' : ''}`} />
                  <span>{t('embedding.rescan_btn')}</span>
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Provider Modal */}
      <Modal
        isOpen={isProviderModalOpen}
        onClose={() => setIsProviderModalOpen(false)}
        title={editingProvider ? t('common.edit') : t('llm.add_provider')}
        maxWidth="lg"
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            saveProviderMutation.mutate();
          }}
          className="space-y-4"
        >
          {actionError && (
            <div className="p-3 rounded-lg bg-rose-50 dark:bg-rose-950/40 text-rose-600 dark:text-rose-400 text-xs">
              {actionError}
            </div>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.provider_id')}
              </label>
              <input
                type="text"
                value={providerForm.id}
                disabled={!!editingProvider}
                onChange={(e) => setProviderForm({ ...providerForm, id: e.target.value })}
                required
                placeholder={t('llm.provider_id_placeholder')}
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.provider_name')}
              </label>
              <input
                type="text"
                value={providerForm.name}
                onChange={(e) => setProviderForm({ ...providerForm, name: e.target.value })}
                required
                placeholder={t('llm.provider_name_placeholder')}
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.provider_type')}
              </label>
              <select
                value={providerForm.provider_type}
                onChange={(e) =>
                  setProviderForm({
                    ...providerForm,
                    provider_type: e.target.value as CreateLLMProviderDTO['provider_type'],
                  })
                }
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
              >
                <option value="openrouter">{t('llm.ptype_openrouter')}</option>
                <option value="openai">{t('llm.ptype_openai')}</option>
                <option value="gemini">{t('llm.ptype_gemini')}</option>
                <option value="custom">{t('llm.ptype_custom')}</option>
              </select>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.base_url')}
              </label>
              <input
                type="text"
                value={providerForm.base_url}
                onChange={(e) => setProviderForm({ ...providerForm, base_url: e.target.value })}
                required
                placeholder={t('llm.base_url_placeholder')}
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-xs"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
              {t('llm.api_key')}
            </label>
            <div className="relative">
              <input
                type={showProviderKey ? 'text' : 'password'}
                value={providerForm.api_key}
                onChange={(e) => setProviderForm({ ...providerForm, api_key: e.target.value })}
                required={!editingProvider}
                autoComplete="new-password"
                placeholder={
                  editingProvider
                    ? t('llm.api_key_edit_placeholder')
                    : t('llm.api_key_placeholder')
                }
                className="w-full px-3 py-2.5 pr-12 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
              />
              <button
                type="button"
                onClick={() => setShowProviderKey((v) => !v)}
                title={showProviderKey ? t('llm.api_key_hide') : t('llm.api_key_show')}
                className="absolute right-1 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-slate-800 dark:hover:text-slate-200"
              >
                {showProviderKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>

          <div className="border border-slate-200 dark:border-slate-800 rounded-lg p-3 space-y-3 bg-slate-50/50 dark:bg-slate-900/50">
            <h4 className="text-xs font-bold text-slate-700 dark:text-slate-200">
              {t('llm.retry_and_timeout_settings')}
            </h4>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
                  {t('llm.timeout_seconds')}
                </label>
                <input
                  type="number"
                  min={1}
                  max={600}
                  value={providerForm.timeout_seconds ?? 60}
                  onChange={(e) =>
                    setProviderForm({
                      ...providerForm,
                      timeout_seconds: parseInt(e.target.value) || 60,
                    })
                  }
                  className="w-full px-3 py-2 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 sm:py-1.5 sm:text-xs"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
                  {t('llm.max_retries')}
                </label>
                <input
                  type="number"
                  min={0}
                  max={10}
                  value={providerForm.max_retries ?? 3}
                  onChange={(e) =>
                    setProviderForm({
                      ...providerForm,
                      max_retries: parseInt(e.target.value) || 0,
                    })
                  }
                  className="w-full px-3 py-2 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 sm:py-1.5 sm:text-xs"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
                  {t('llm.retry_initial_wait_ms')}
                </label>
                <input
                  type="number"
                  min={50}
                  max={10000}
                  value={providerForm.retry_initial_wait_ms ?? 500}
                  onChange={(e) =>
                    setProviderForm({
                      ...providerForm,
                      retry_initial_wait_ms: parseInt(e.target.value) || 500,
                    })
                  }
                  className="w-full px-3 py-2 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 sm:py-1.5 sm:text-xs"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
                  {t('llm.retry_max_wait_ms')}
                </label>
                <input
                  type="number"
                  min={100}
                  max={60000}
                  value={providerForm.retry_max_wait_ms ?? 5000}
                  onChange={(e) =>
                    setProviderForm({
                      ...providerForm,
                      retry_max_wait_ms: parseInt(e.target.value) || 5000,
                    })
                  }
                  className="w-full px-3 py-2 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 sm:py-1.5 sm:text-xs"
                />
              </div>
            </div>

            <label className="flex items-center gap-2 cursor-pointer pt-1">
              <input
                type="checkbox"
                checked={providerForm.allow_private_networks ?? false}
                onChange={(e) =>
                  setProviderForm({
                    ...providerForm,
                    allow_private_networks: e.target.checked,
                  })
                }
                className="h-5 w-5 rounded text-indigo-600 focus:ring-indigo-500"
              />
              <span className="text-xs text-slate-600 dark:text-slate-300">
                {t('llm.allow_private_networks')}
              </span>
            </label>
          </div>

          <div className="flex items-center gap-6 pt-2">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={providerForm.is_active}
                onChange={(e) => setProviderForm({ ...providerForm, is_active: e.target.checked })}
                className="h-5 w-5 rounded text-indigo-600 focus:ring-indigo-500"
              />
              <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                {t('llm.is_active')}
              </span>
            </label>

            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={providerForm.is_default}
                onChange={(e) => setProviderForm({ ...providerForm, is_default: e.target.checked })}
                className="h-5 w-5 rounded text-indigo-600 focus:ring-indigo-500"
              />
              <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                {t('llm.is_default')}
              </span>
            </label>
          </div>

          <div className="flex flex-col-reverse gap-2 border-t border-slate-100 pt-4 dark:border-slate-800 sm:flex-row sm:justify-end sm:gap-3">
            <button
              type="button"
              onClick={() => setIsProviderModalOpen(false)}
              className="inline-flex min-h-[44px] w-full items-center justify-center rounded-lg px-4 py-2.5 text-sm font-medium text-slate-600 transition hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800 sm:w-auto sm:py-2"
            >
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              disabled={saveProviderMutation.isPending}
              className="inline-flex min-h-[44px] w-full items-center justify-center gap-2 rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-md shadow-indigo-600/20 transition hover:bg-indigo-700 sm:w-auto sm:py-2"
            >
              {saveProviderMutation.isPending && <LoadingSpinner size="sm" />}
              <span>{t('common.save')}</span>
            </button>
          </div>
        </form>
      </Modal>

      {/* Model Modal */}
      <Modal
        isOpen={isModelModalOpen}
        onClose={() => setIsModelModalOpen(false)}
        title={editingModel ? t('common.edit') : t('llm.add_model')}
        maxWidth="lg"
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            saveModelMutation.mutate();
          }}
          className="space-y-4"
        >
          {actionError && (
            <div className="p-3 rounded-lg bg-rose-50 dark:bg-rose-950/40 text-rose-600 dark:text-rose-400 text-xs">
              {actionError}
            </div>
          )}

          {probeResultMsg && (
            <div className="p-3 rounded-lg bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 text-xs flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 shrink-0" />
              <span>{probeResultMsg}</span>
            </div>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.provider_name')}
              </label>
              <select
                value={modelForm.provider_id}
                onChange={(e) => setModelForm({ ...modelForm, provider_id: e.target.value })}
                required
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
              >
                {providers.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name} ({p.id})
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.model_id')}
              </label>
              <input
                type="text"
                value={modelForm.id}
                disabled={!!editingModel}
                onChange={(e) => setModelForm({ ...modelForm, id: e.target.value })}
                required
                placeholder={t('llm.model_id_placeholder')}
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.model_name')}
              </label>
              <input
                type="text"
                value={modelForm.name}
                onChange={(e) => setModelForm({ ...modelForm, name: e.target.value })}
                required
                placeholder={t('llm.model_name_placeholder')}
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.model_type')}
              </label>
              <select
                value={modelForm.model_type}
                onChange={(e) =>
                  setModelForm({
                    ...modelForm,
                    model_type: e.target.value as CreateLLMModelDTO['model_type'],
                  })
                }
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
              >
                <option value="chat">{t('llm.type_chat')}</option>
                <option value="embedding">{t('llm.type_embedding')}</option>
                <option value="rerank">{t('llm.type_rerank')}</option>
              </select>
            </div>
          </div>

          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300">
                {t('llm.model_key')}
              </label>
              <button
                type="button"
                onClick={handleProbeVision}
                disabled={isProbing || !modelForm.model_key}
                className="flex min-h-[44px] items-center gap-1 px-2 text-xs font-semibold text-indigo-600 transition hover:text-indigo-700 disabled:opacity-50 dark:text-indigo-400 dark:hover:text-indigo-300 sm:min-h-0 sm:px-0"
              >
                {isProbing ? <LoadingSpinner size="sm" /> : <Sparkles className="w-3.5 h-3.5" />}
                <span>{isProbing ? t('llm.probing_vision') : t('llm.probe_vision_btn')}</span>
              </button>
            </div>
            <input
              type="text"
              value={modelForm.model_key}
              onChange={(e) => setModelForm({ ...modelForm, model_key: e.target.value })}
              required
              placeholder={t('llm.model_key_placeholder')}
              className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-xs"
            />
          </div>

          {/* Vision Configuration Section */}
          <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700 space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <span className="text-xs font-bold text-slate-800 dark:text-slate-200">
                  {t('llm.supports_vision')}
                </span>
                <p className="text-[11px] text-slate-500 dark:text-slate-400">
                  {t('llm.supports_vision_desc')}
                </p>
              </div>

              <label className="relative -m-2 inline-flex items-center cursor-pointer p-2">
                <input
                  type="checkbox"
                  checked={modelForm.supports_vision}
                  onChange={(e) =>
                    setModelForm({ ...modelForm, supports_vision: e.target.checked })
                  }
                  className="sr-only peer"
                />
                <div className="relative shrink-0 w-11 h-6 bg-slate-300 peer-focus:outline-none rounded-full peer dark:bg-slate-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-slate-600 peer-checked:bg-indigo-600"></div>
              </label>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                  {t('llm.vision_mode')}
                </label>
                <select
                  value={modelForm.vision_mode}
                  onChange={(e) =>
                    setModelForm({
                      ...modelForm,
                      vision_mode: e.target.value as 'auto' | 'manual',
                    })
                  }
                  className="w-full px-3 py-2 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-1.5 sm:text-xs"
                >
                  <option value="auto">{t('llm.vision_auto')}</option>
                  <option value="manual">{t('llm.vision_manual')}</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                  {t('llm.max_images')}
                </label>
                <input
                  type="number"
                  min={0}
                  max={20}
                  value={modelForm.max_images}
                  onChange={(e) =>
                    setModelForm({ ...modelForm, max_images: parseInt(e.target.value, 10) || 0 })
                  }
                  className="w-full px-3 py-2 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-1.5 sm:text-xs"
                />
              </div>
            </div>
          </div>

          <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700 space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <span className="text-xs font-bold text-slate-800 dark:text-slate-200">
                  {t('llm.thinking_enabled')}
                </span>
                <p className="text-[11px] text-slate-500 dark:text-slate-400">
                  {t('llm.thinking_enabled_desc')}
                </p>
              </div>

              <label className="relative -m-2 inline-flex items-center cursor-pointer p-2">
                <input
                  type="checkbox"
                  checked={modelForm.thinking_enabled || false}
                  onChange={(e) =>
                    setModelForm({ ...modelForm, thinking_enabled: e.target.checked })
                  }
                  className="sr-only peer"
                />
                <div className="relative shrink-0 w-11 h-6 bg-slate-300 peer-focus:outline-none rounded-full peer dark:bg-slate-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-slate-600 peer-checked:bg-indigo-600"></div>
              </label>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                  {t('llm.effort_level')}
                </label>
                <select
                  value={modelForm.effort_level || 'medium'}
                  onChange={(e) =>
                    setModelForm({
                      ...modelForm,
                      effort_level: e.target.value as 'low' | 'medium' | 'high',
                    })
                  }
                  className="w-full px-3 py-2 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-1.5 sm:text-xs"
                >
                  <option value="low">{t('llm.effort_low')}</option>
                  <option value="medium">{t('llm.effort_medium')}</option>
                  <option value="high">{t('llm.effort_high')}</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                  {t('llm.thinking_budget')}
                </label>
                <input
                  type="number"
                  min={0}
                  step={512}
                  placeholder={t('llm.thinking_budget_placeholder')}
                  value={modelForm.thinking_budget || 0}
                  onChange={(e) =>
                    setModelForm({ ...modelForm, thinking_budget: parseInt(e.target.value, 10) || 0 })
                  }
                  className="w-full px-3 py-2 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-1.5 sm:text-xs"
                />
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.context_length')}
              </label>
              <input
                type="number"
                value={modelForm.context_length}
                onChange={(e) =>
                  setModelForm({
                    ...modelForm,
                    context_length: parseInt(e.target.value, 10) || 32768,
                  })
                }
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.order_index')}
              </label>
              <input
                type="number"
                value={modelForm.order_index}
                onChange={(e) =>
                  setModelForm({ ...modelForm, order_index: parseInt(e.target.value, 10) || 0 })
                }
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
              />
            </div>
          </div>

          <div className="flex items-center gap-6 pt-2">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={modelForm.is_active}
                onChange={(e) => setModelForm({ ...modelForm, is_active: e.target.checked })}
                className="h-5 w-5 rounded text-indigo-600 focus:ring-indigo-500"
              />
              <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                {t('llm.is_active')}
              </span>
            </label>

            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={modelForm.is_default}
                onChange={(e) => setModelForm({ ...modelForm, is_default: e.target.checked })}
                className="h-5 w-5 rounded text-indigo-600 focus:ring-indigo-500"
              />
              <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                {t('llm.is_default')}
              </span>
            </label>
          </div>

          <div className="flex flex-col-reverse gap-2 border-t border-slate-100 pt-4 dark:border-slate-800 sm:flex-row sm:justify-end sm:gap-3">
            <button
              type="button"
              onClick={() => setIsModelModalOpen(false)}
              className="inline-flex min-h-[44px] w-full items-center justify-center rounded-lg px-4 py-2.5 text-sm font-medium text-slate-600 transition hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800 sm:w-auto sm:py-2"
            >
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              disabled={saveModelMutation.isPending}
              className="inline-flex min-h-[44px] w-full items-center justify-center gap-2 rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-md shadow-indigo-600/20 transition hover:bg-indigo-700 cursor-pointer sm:w-auto sm:py-2"
            >
              {saveModelMutation.isPending && <LoadingSpinner size="sm" />}
              <span>{t('common.save')}</span>
            </button>
          </div>
        </form>
      </Modal>

      {/* Chain Modal */}
      <Modal
        isOpen={isChainModalOpen}
        onClose={() => setIsChainModalOpen(false)}
        title={editingChain ? t('llm.edit_chain') : t('llm.add_chain')}
        maxWidth="md"
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            saveChainMutation.mutate();
          }}
          className="space-y-4"
        >
          {actionError && (
            <div className="p-3 rounded-lg bg-rose-50 dark:bg-rose-950/40 text-rose-600 dark:text-rose-400 text-xs">
              {actionError}
            </div>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.chain_id')}
              </label>
              <input
                type="text"
                value={chainForm.id}
                disabled={!!editingChain}
                onChange={(e) => setChainForm({ ...chainForm, id: e.target.value })}
                required
                placeholder={t('llm.chain_id_placeholder')}
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.chain_type')}
              </label>
              <select
                value={chainForm.chain_type}
                disabled={!!editingChain}
                onChange={(e) =>
                  setChainForm({
                    ...chainForm,
                    chain_type: e.target.value as CreateLLMChainDTO['chain_type'],
                  })
                }
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
              >
                <option value="chat">{t('llm.chain_type_chat')}</option>
                <option value="rerank">{t('llm.chain_type_rerank')}</option>
                <option value="embedding">{t('llm.chain_type_embedding')}</option>
              </select>
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
              {t('llm.chain_name')}
            </label>
            <input
              type="text"
              value={chainForm.name}
              onChange={(e) => setChainForm({ ...chainForm, name: e.target.value })}
              required
              placeholder={t('llm.chain_name_placeholder')}
              className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
              {t('llm.description')}
            </label>
            <textarea
              rows={2}
              value={chainForm.description || ''}
              onChange={(e) => setChainForm({ ...chainForm, description: e.target.value })}
              placeholder={t('llm.chain_desc_placeholder')}
              className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
            />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.failure_threshold')}
              </label>
              <input
                type="number"
                min={1}
                max={20}
                value={chainForm.failure_threshold}
                onChange={(e) =>
                  setChainForm({ ...chainForm, failure_threshold: parseInt(e.target.value, 10) || 3 })
                }
                className="w-full px-3 py-2 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-1.5 sm:text-xs"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.cooldown_seconds')}
              </label>
              <input
                type="number"
                min={5}
                max={86400}
                value={chainForm.cooldown_seconds}
                onChange={(e) =>
                  setChainForm({ ...chainForm, cooldown_seconds: parseInt(e.target.value, 10) || 300 })
                }
                className="w-full px-3 py-2 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-1.5 sm:text-xs"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.retry_count')}
              </label>
              <input
                type="number"
                min={0}
                max={10}
                value={chainForm.retry_count}
                onChange={(e) =>
                  setChainForm({ ...chainForm, retry_count: parseInt(e.target.value, 10) || 3 })
                }
                className="w-full px-3 py-2 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-1.5 sm:text-xs"
              />
            </div>
          </div>

          <div className="pt-2">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={chainForm.is_active}
                onChange={(e) => setChainForm({ ...chainForm, is_active: e.target.checked })}
                className="h-5 w-5 rounded text-indigo-600 focus:ring-indigo-500"
              />
              <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                {t('llm.is_active')}
              </span>
            </label>
          </div>

          <div className="flex flex-col-reverse gap-2 border-t border-slate-100 pt-4 dark:border-slate-800 sm:flex-row sm:justify-end sm:gap-3">
            <button
              type="button"
              onClick={() => setIsChainModalOpen(false)}
              className="inline-flex min-h-[44px] w-full items-center justify-center rounded-lg px-4 py-2.5 text-sm font-medium text-slate-600 transition hover:bg-slate-100 cursor-pointer dark:text-slate-300 dark:hover:bg-slate-800 sm:w-auto sm:py-2"
            >
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              disabled={saveChainMutation.isPending}
              className="inline-flex min-h-[44px] w-full items-center justify-center gap-2 rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-md shadow-indigo-600/20 transition hover:bg-indigo-700 cursor-pointer sm:w-auto sm:py-2"
            >
              {saveChainMutation.isPending && <LoadingSpinner size="sm" />}
              <span>{t('common.save')}</span>
            </button>
          </div>
        </form>
      </Modal>

      {/* Add Chain Node Modal */}
      <Modal
        isOpen={isNodeModalOpen}
        onClose={() => setIsNodeModalOpen(false)}
        title={`${t('llm.add_node')} - ${selectedChainForNode?.name || ''}`}
        maxWidth="md"
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            addNodeMutation.mutate();
          }}
          className="space-y-4"
        >
          {actionError && (
            <div className="p-3 rounded-lg bg-rose-50 dark:bg-rose-950/40 text-rose-600 dark:text-rose-400 text-xs">
              {actionError}
            </div>
          )}

          {selectedChainForNode?.chain_type === 'embedding' && selectedChainForNode.nodes.length > 0 && (
            <div className="p-3 rounded-lg bg-amber-50 dark:bg-amber-950/40 text-amber-800 dark:text-amber-300 text-xs flex items-start gap-2">
              <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
              <div>
                <strong>{t('llm.embedding_constraint_title')}:</strong> {t('llm.embedding_constraint_msg')}:
                <code className="font-mono font-bold ml-1 bg-white dark:bg-slate-900 px-1 py-0.5 rounded border border-amber-200 dark:border-amber-800">
                  {selectedChainForNode.nodes[0].model_key}
                </code>
              </div>
            </div>
          )}

          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
              {t('llm.select_test_model')}
            </label>
            <select
              value={nodeForm.model_id}
              onChange={(e) => setNodeForm({ ...nodeForm, model_id: e.target.value })}
              required
              className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
            >
              <option value="" disabled>{t('llm.select_model_placeholder')}</option>
              {models
                .filter((m) => {
                  if (!selectedChainForNode) return true;
                  return m.model_type === selectedChainForNode.chain_type;
                })
                .map((m) => {
                  const isEmbeddingMismatch =
                    selectedChainForNode?.chain_type === 'embedding' &&
                    selectedChainForNode.nodes.length > 0 &&
                    m.model_key !== selectedChainForNode.nodes[0].model_key;

                  const isAlreadyInChain = selectedChainForNode?.nodes.some(
                    (n) => n.model_id === m.id
                  );

                  return (
                    <option
                      key={m.id}
                      value={m.id}
                      disabled={isEmbeddingMismatch || isAlreadyInChain}
                    >
                      {m.name} ({m.model_key}) - {m.provider_name || m.provider_id}
                      {isEmbeddingMismatch ? ` [${t('llm.incompatible_model_key')}]` : ''}
                      {isAlreadyInChain ? ` [${t('llm.already_in_chain')}]` : ''}
                    </option>
                  );
                })}
            </select>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('llm.priority')}
              </label>
              <input
                type="number"
                min={1}
                value={nodeForm.priority}
                onChange={(e) =>
                  setNodeForm({ ...nodeForm, priority: parseInt(e.target.value, 10) || 1 })
                }
                required
                className="w-full px-3 py-2.5 text-base rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:py-2 sm:text-sm"
              />
            </div>

            <div className="flex items-center pt-2 sm:pt-5">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={nodeForm.is_active}
                  onChange={(e) => setNodeForm({ ...nodeForm, is_active: e.target.checked })}
                  className="h-5 w-5 rounded text-indigo-600 focus:ring-indigo-500"
                />
                <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                  {t('llm.is_active')}
                </span>
              </label>
            </div>
          </div>

          <div className="flex flex-col-reverse gap-2 border-t border-slate-100 pt-4 dark:border-slate-800 sm:flex-row sm:justify-end sm:gap-3">
            <button
              type="button"
              onClick={() => setIsNodeModalOpen(false)}
              className="inline-flex min-h-[44px] w-full items-center justify-center rounded-lg px-4 py-2.5 text-sm font-medium text-slate-600 transition hover:bg-slate-100 cursor-pointer dark:text-slate-300 dark:hover:bg-slate-800 sm:w-auto sm:py-2"
            >
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              disabled={addNodeMutation.isPending || !nodeForm.model_id}
              className="inline-flex min-h-[44px] w-full items-center justify-center gap-2 rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-md shadow-indigo-600/20 transition hover:bg-indigo-700 cursor-pointer sm:w-auto sm:py-2"
            >
              {addNodeMutation.isPending && <LoadingSpinner size="sm" />}
              <span>{t('llm.add_node')}</span>
            </button>
          </div>
        </form>
      </Modal>

      <Modal
        isOpen={isDownloadModalOpen}
        onClose={() => setIsDownloadModalOpen(false)}
        title={t('embedding.download_modal_title')}
        maxWidth="xl"
      >
        <div className="space-y-5">
          <p className="text-xs text-slate-500 dark:text-slate-400">{t('embedding.download_modal_desc')}</p>

          {downloadJobs.length > 0 && (
            <div className="space-y-2">
              <h4 className="text-xs font-bold text-slate-700 dark:text-slate-200">
                {t('embedding.download_jobs_title')}
              </h4>
              {downloadJobs.map((job) => {
                const totalBytes = job.total_bytes || 0;
                const percent =
                  totalBytes > 0 ? Math.min(100, Math.round((job.downloaded_bytes / totalBytes) * 100)) : 0;
                return (
                  <div
                    key={job.job_id}
                    className="p-3 rounded-lg border border-slate-200 dark:border-slate-800 space-y-2"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="min-w-0 truncate text-xs font-semibold text-slate-900 dark:text-slate-100">
                        {resolvePresetName(job.preset_id)}
                      </span>
                      <Badge variant={DOWNLOAD_STATE_VARIANTS[job.state]} size="sm">
                        {t(`embedding.status_${job.state}`)}
                      </Badge>
                    </div>
                    {job.state === 'running' && (
                      <>
                        <div className="h-1.5 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800">
                          <div
                            className={`h-full rounded-full bg-indigo-600 transition-all duration-500 ${
                              totalBytes > 0 ? '' : 'animate-pulse w-2/5'
                            }`}
                            style={totalBytes > 0 ? { width: `${percent}%` } : undefined}
                          />
                        </div>
                        <span className="block text-[11px] text-slate-500 dark:text-slate-400 font-mono">
                          {totalBytes > 0
                            ? t('embedding.download_progress', {
                                downloaded: formatBytes(job.downloaded_bytes),
                                total: formatBytes(totalBytes),
                                percent,
                              })
                            : t('embedding.download_progress_unknown', {
                                downloaded: formatBytes(job.downloaded_bytes),
                              })}
                        </span>
                      </>
                    )}
                    {job.state === 'error' && job.error && (
                      <span className="block text-[11px] text-rose-600 dark:text-rose-400 break-words">
                        {job.error}
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
          )}

          <div className="space-y-2">
            {!onnxCatalog && (
              <div className="py-8 flex justify-center">
                <LoadingSpinner size="md" />
              </div>
            )}
            {onnxCatalog && onnxCatalog.models.length === 0 && (
              <div className="p-6 text-center text-xs text-slate-400">{t('embedding.models_empty')}</div>
            )}
            {(onnxCatalog?.models ?? []).map((preset) => {
              const isRunning = downloadJobs.some(
                (job) => job.preset_id === preset.preset_id && job.state === 'running'
              );
              return (
                <div
                  key={preset.preset_id}
                  className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-3 sm:p-4 rounded-xl border border-slate-200 dark:border-slate-800"
                >
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-semibold text-sm text-slate-900 dark:text-slate-100 break-words">
                        {preset.display_name}
                      </span>
                      <Badge variant="secondary" size="sm">
                        {t('embedding.model_dims', { dim: preset.dim })}
                      </Badge>
                      {preset.dim === 384 && (
                        <Badge variant="success" size="sm">
                          {t('embedding.db_match_badge')}
                        </Badge>
                      )}
                      <span className="text-[11px] text-slate-400">
                        {formatBytes(preset.approx_size_bytes)}
                      </span>
                    </div>
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1 leading-relaxed">
                      {preset.description}
                    </p>
                  </div>
                  {canManageLLM && (
                    <button
                      type="button"
                      onClick={() => startDownloadMutation.mutate(preset.preset_id)}
                      disabled={isRunning || startDownloadMutation.isPending}
                      className="inline-flex min-h-[44px] w-full sm:w-auto shrink-0 items-center justify-center gap-1.5 px-4 py-2.5 rounded-lg bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold shadow-md shadow-indigo-600/20 transition cursor-pointer disabled:opacity-50 sm:min-h-0 sm:py-2"
                    >
                      <Download className="w-3.5 h-3.5" />
                      <span>{isRunning ? t('embedding.status_running') : t('embedding.download_action')}</span>
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </Modal>

      <Modal
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        title={t('embedding.delete_confirm_title')}
        maxWidth="sm"
      >
        <div className="space-y-4">
          <p className="text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
            {t('embedding.delete_confirm_desc', {
              name: deleteTarget?.display_name || deleteTarget?.model_id || '',
            })}
          </p>
          <div className="flex flex-col-reverse gap-2 border-t border-slate-100 pt-4 dark:border-slate-800 sm:flex-row sm:justify-end sm:gap-3">
            <button
              type="button"
              onClick={() => setDeleteTarget(null)}
              className="inline-flex min-h-[44px] w-full items-center justify-center rounded-lg px-4 py-2.5 text-sm font-medium text-slate-600 transition hover:bg-slate-100 cursor-pointer dark:text-slate-300 dark:hover:bg-slate-800 sm:w-auto sm:py-2"
            >
              {t('common.cancel')}
            </button>
            <button
              type="button"
              onClick={() => deleteOnnxModelMutation.mutate(deleteTarget?.model_id || '')}
              disabled={deleteOnnxModelMutation.isPending || !deleteTarget}
              className="inline-flex min-h-[44px] w-full items-center justify-center gap-2 rounded-lg bg-rose-600 px-4 py-2.5 text-sm font-semibold text-white shadow-md shadow-rose-600/20 transition hover:bg-rose-700 disabled:opacity-50 cursor-pointer sm:w-auto sm:py-2"
            >
              {deleteOnnxModelMutation.isPending && <LoadingSpinner size="sm" />}
              <span>{t('common.delete')}</span>
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
};
