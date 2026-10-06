import { create } from 'zustand';
import type { EmbeddingProvider, OnnxModelDTO } from '@/types/embedding';

// EmbeddingManageState defines UI state for the embedding management tab.
export interface EmbeddingManageState {
  isEmbeddingTabOpen: boolean;
  providerDraft: EmbeddingProvider;
  providerTouched: boolean;
  isDownloadModalOpen: boolean;
  deleteTarget: OnnxModelDTO | null;
  copiedModelId: string | null;
  toastMessage: string | null;
  toastTone: 'success' | 'error';

  setIsEmbeddingTabOpen: (open: boolean) => void;
  setProviderDraft: (provider: EmbeddingProvider) => void;
  resetProviderDraft: (provider: EmbeddingProvider) => void;
  setIsDownloadModalOpen: (open: boolean) => void;
  setDeleteTarget: (model: OnnxModelDTO | null) => void;
  setCopiedModelId: (id: string | null) => void;
  setToast: (message: string | null, tone?: 'success' | 'error') => void;
}

let toastTimeout: ReturnType<typeof setTimeout> | null = null;
let copiedTimeout: ReturnType<typeof setTimeout> | null = null;

// useEmbeddingManageStore provides centralized state for the embedding tab, modals, and toasts.
export const useEmbeddingManageStore = create<EmbeddingManageState>((set) => ({
  isEmbeddingTabOpen: false,
  providerDraft: 'api',
  providerTouched: false,
  isDownloadModalOpen: false,
  deleteTarget: null,
  copiedModelId: null,
  toastMessage: null,
  toastTone: 'success',

  setIsEmbeddingTabOpen: (isEmbeddingTabOpen) => set({ isEmbeddingTabOpen }),
  setProviderDraft: (providerDraft) => set({ providerDraft, providerTouched: true }),
  resetProviderDraft: (providerDraft) => set({ providerDraft, providerTouched: false }),
  setIsDownloadModalOpen: (isDownloadModalOpen) => set({ isDownloadModalOpen }),
  setDeleteTarget: (deleteTarget) => set({ deleteTarget }),
  setCopiedModelId: (copiedModelId) => {
    if (copiedTimeout) {
      clearTimeout(copiedTimeout);
      copiedTimeout = null;
    }
    set({ copiedModelId });
    if (copiedModelId) {
      copiedTimeout = setTimeout(() => {
        set({ copiedModelId: null });
        copiedTimeout = null;
      }, 2000);
    }
  },
  setToast: (toastMessage, toastTone = 'success') => {
    if (toastTimeout) {
      clearTimeout(toastTimeout);
      toastTimeout = null;
    }
    set({ toastMessage, toastTone });
    if (toastMessage) {
      toastTimeout = setTimeout(() => {
        set({ toastMessage: null });
        toastTimeout = null;
      }, 3500);
    }
  },
}));
