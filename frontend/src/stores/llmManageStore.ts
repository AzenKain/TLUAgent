import { create } from 'zustand';
import type {
  LLMProviderDTO,
  CreateLLMProviderDTO,
  LLMModelDTO,
  CreateLLMModelDTO,
  LLMChainDTO,
  CreateLLMChainDTO,
  AddChainNodeDTO,
  TestModelResponseDTO,
} from '@/services/llmService';
import i18n from '@/i18n';

export type LLMTab = 'providers' | 'models' | 'chains' | 'playground';

const initialProviderForm: CreateLLMProviderDTO = {
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
};

const initialModelForm: CreateLLMModelDTO = {
  id: '',
  provider_id: '',
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
};

const initialChainForm: CreateLLMChainDTO = {
  id: '',
  chain_type: 'chat',
  name: '',
  description: '',
  is_active: true,
  failure_threshold: 3,
  cooldown_seconds: 300,
  retry_count: 3,
};

const initialNodeForm: AddChainNodeDTO = {
  model_id: '',
  priority: 1,
  is_active: true,
};

// LLMManageState defines state and actions for the LLM management and playground screens.
export interface LLMManageState {
  activeTab: LLMTab;
  providerFilter: string;

  isProviderModalOpen: boolean;
  editingProvider: LLMProviderDTO | null;
  providerForm: CreateLLMProviderDTO;

  isModelModalOpen: boolean;
  editingModel: LLMModelDTO | null;
  modelForm: CreateLLMModelDTO;

  isProbing: boolean;
  probeResultMsg: string | null;
  actionError: string | null;

  isChainModalOpen: boolean;
  editingChain: LLMChainDTO | null;
  chainForm: CreateLLMChainDTO;

  isNodeModalOpen: boolean;
  selectedChainForNode: LLMChainDTO | null;
  nodeForm: AddChainNodeDTO;

  testTargetType: 'model' | 'chain';
  selectedTestModelId: string;
  selectedTestChainId: string;
  testSystemPrompt: string;
  testPrompt: string;
  testTemperature: number;
  testMaxTokens: number;
  testImages: string[];
  isAdvancedOpen: boolean;
  testResult: TestModelResponseDTO | null;
  isTesting: boolean;
  testError: string | null;
  activeResultTab: 'output' | 'raw';
  copiedRaw: boolean;

  setActiveTab: (tab: LLMTab) => void;
  setProviderFilter: (filter: string) => void;
  setIsProviderModalOpen: (open: boolean) => void;
  setEditingProvider: (provider: LLMProviderDTO | null) => void;
  setProviderForm: (form: CreateLLMProviderDTO | Partial<CreateLLMProviderDTO> | ((prev: CreateLLMProviderDTO) => CreateLLMProviderDTO)) => void;
  setIsModelModalOpen: (open: boolean) => void;
  setEditingModel: (model: LLMModelDTO | null) => void;
  setModelForm: (form: CreateLLMModelDTO | Partial<CreateLLMModelDTO> | ((prev: CreateLLMModelDTO) => CreateLLMModelDTO)) => void;
  setIsProbing: (probing: boolean) => void;
  setProbeResultMsg: (msg: string | null) => void;
  setActionError: (err: string | null) => void;
  setIsChainModalOpen: (open: boolean) => void;
  setEditingChain: (chain: LLMChainDTO | null) => void;
  setChainForm: (form: CreateLLMChainDTO | Partial<CreateLLMChainDTO> | ((prev: CreateLLMChainDTO) => CreateLLMChainDTO)) => void;
  setIsNodeModalOpen: (open: boolean) => void;
  setSelectedChainForNode: (chain: LLMChainDTO | null) => void;
  setNodeForm: (form: AddChainNodeDTO | Partial<AddChainNodeDTO> | ((prev: AddChainNodeDTO) => AddChainNodeDTO)) => void;

  setTestTargetType: (target: 'model' | 'chain') => void;
  setSelectedTestModelId: (id: string) => void;
  setSelectedTestChainId: (id: string) => void;
  setTestSystemPrompt: (prompt: string) => void;
  setTestPrompt: (prompt: string) => void;
  setTestTemperature: (temp: number) => void;
  setTestMaxTokens: (tokens: number) => void;
  setTestImages: (images: string[] | ((prev: string[]) => string[])) => void;
  setIsAdvancedOpen: (open: boolean) => void;
  setTestResult: (result: TestModelResponseDTO | null) => void;
  setIsTesting: (testing: boolean) => void;
  setTestError: (err: string | null) => void;
  setActiveResultTab: (tab: 'output' | 'raw') => void;
  setCopiedRaw: (copied: boolean) => void;
}

// useLLMManageStore provides centralized state management for LLM providers, models, chains, and playground.
export const useLLMManageStore = create<LLMManageState>((set) => ({
  activeTab: 'providers',
  providerFilter: '',

  isProviderModalOpen: false,
  editingProvider: null,
  providerForm: { ...initialProviderForm },

  isModelModalOpen: false,
  editingModel: null,
  modelForm: { ...initialModelForm },

  isProbing: false,
  probeResultMsg: null,
  actionError: null,

  isChainModalOpen: false,
  editingChain: null,
  chainForm: { ...initialChainForm },

  isNodeModalOpen: false,
  selectedChainForNode: null,
  nodeForm: { ...initialNodeForm },

  testTargetType: 'model',
  selectedTestModelId: '',
  selectedTestChainId: '',
  testSystemPrompt: i18n.t('llm.default_test_system_prompt'),
  testPrompt: i18n.t('llm.default_test_prompt'),
  testTemperature: 0.7,
  testMaxTokens: 1024,
  testImages: [],
  isAdvancedOpen: false,
  testResult: null,
  isTesting: false,
  testError: null,
  activeResultTab: 'output',
  copiedRaw: false,

  setActiveTab: (activeTab) => set({ activeTab }),
  setProviderFilter: (providerFilter) => set({ providerFilter }),
  setIsProviderModalOpen: (isProviderModalOpen) => set({ isProviderModalOpen }),
  setEditingProvider: (editingProvider) => set({ editingProvider }),
  setProviderForm: (form) =>
    set((state) => ({
      providerForm: typeof form === 'function' ? form(state.providerForm) : { ...state.providerForm, ...form },
    })),
  setIsModelModalOpen: (isModelModalOpen) => set({ isModelModalOpen }),
  setEditingModel: (editingModel) => set({ editingModel }),
  setModelForm: (form) =>
    set((state) => ({
      modelForm: typeof form === 'function' ? form(state.modelForm) : { ...state.modelForm, ...form },
    })),
  setIsProbing: (isProbing) => set({ isProbing }),
  setProbeResultMsg: (probeResultMsg) => set({ probeResultMsg }),
  setActionError: (actionError) => set({ actionError }),
  setIsChainModalOpen: (isChainModalOpen) => set({ isChainModalOpen }),
  setEditingChain: (editingChain) => set({ editingChain }),
  setChainForm: (form) =>
    set((state) => ({
      chainForm: typeof form === 'function' ? form(state.chainForm) : { ...state.chainForm, ...form },
    })),
  setIsNodeModalOpen: (isNodeModalOpen) => set({ isNodeModalOpen }),
  setSelectedChainForNode: (selectedChainForNode) => set({ selectedChainForNode }),
  setNodeForm: (form) =>
    set((state) => ({
      nodeForm: typeof form === 'function' ? form(state.nodeForm) : { ...state.nodeForm, ...form },
    })),

  setTestTargetType: (testTargetType) => set({ testTargetType }),
  setSelectedTestModelId: (selectedTestModelId) => set({ selectedTestModelId }),
  setSelectedTestChainId: (selectedTestChainId) => set({ selectedTestChainId }),
  setTestSystemPrompt: (testSystemPrompt) => set({ testSystemPrompt }),
  setTestPrompt: (testPrompt) => set({ testPrompt }),
  setTestTemperature: (testTemperature) => set({ testTemperature }),
  setTestMaxTokens: (testMaxTokens) => set({ testMaxTokens }),
  setTestImages: (images) =>
    set((state) => ({
      testImages: typeof images === 'function' ? images(state.testImages) : images,
    })),
  setIsAdvancedOpen: (isAdvancedOpen) => set({ isAdvancedOpen }),
  setTestResult: (testResult) => set({ testResult }),
  setIsTesting: (isTesting) => set({ isTesting }),
  setTestError: (testError) => set({ testError }),
  setActiveResultTab: (activeResultTab) => set({ activeResultTab }),
  setCopiedRaw: (copiedRaw) => set({ copiedRaw }),
}));
