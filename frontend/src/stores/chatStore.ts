import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { chatService, type ChatMessage, type ChatModelDTO, type ToolCallInfo } from '@/services/chatService';
import { useAuthStore } from '@/stores/authStore';
import i18n from '@/i18n';

export interface ChatSession {
  id: string;
  title: string;
  updatedAt: number;
  modelId?: string;
  messages: ChatMessage[];
}

export interface ChatState {
  sessions: ChatSession[];
  activeSessionId: string | null;
  messages: ChatMessage[];
  input: string;
  isLoading: boolean;
  isSessionLoading: boolean;
  models: ChatModelDTO[];
  selectedModel: ChatModelDTO | null;
  attachedImages: string[];

  copiedId: string | null;
  feedbackMap: Record<string, 'up' | 'down'>;
  isModelDropdownOpen: boolean;
  isSidebarOpen: boolean;
  isUserMenuOpen: boolean;

  setInput: (input: string) => void;
  setLoading: (isLoading: boolean) => void;
  setSessionLoading: (loading: boolean) => void;
  addMessage: (message: ChatMessage) => void;
  setMessages: (messages: ChatMessage[]) => void;
  clearMessages: () => void;
  resetChat: () => void;
  createNewChat: () => void;
  selectSession: (sessionId: string) => Promise<void>;
  deleteSession: (sessionId: string) => Promise<void>;
  renameSession: (sessionId: string, newTitle: string) => Promise<void>;
  setModels: (models: ChatModelDTO[]) => void;
  setSelectedModel: (model: ChatModelDTO | null) => void;
  addAttachedImage: (base64: string) => void;
  removeAttachedImage: (index: number) => void;
  clearAttachedImages: () => void;
  updateMessage: (id: string, patch: Partial<ChatMessage>) => void;
  appendMessageChunk: (id: string, chunk: string) => void;
  appendMessageReasoning: (id: string, chunk: string) => void;
  updateMessageToolCall: (id: string, toolCall: ToolCallInfo) => void;
  setActiveSessionId: (id: string | null) => void;
  setCopiedId: (id: string | null) => void;
  setFeedback: (id: string, fb: 'up' | 'down') => Promise<void>;
  setIsModelDropdownOpen: (open: boolean) => void;
  toggleSidebar: () => void;
  setIsSidebarOpen: (open: boolean) => void;
  setIsUserMenuOpen: (open: boolean) => void;
  syncAuthMode: (isAuthenticated: boolean) => Promise<void>;
  loadUserSessions: () => Promise<void>;
}

const getInitialWelcomeMessage = (): ChatMessage => ({
  id: 'welcome',
  sender: 'assistant',
  text: i18n.t('chat.welcome_message'),
  timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
});

try {
  if (typeof window !== 'undefined' && window.localStorage) {
    localStorage.removeItem('tlu_guest_chat');
  }
} catch {}

export const useChatStore = create<ChatState>()(
  persist(
    (set, get) => ({
      sessions: [],
      activeSessionId: null,
      messages: [getInitialWelcomeMessage()],
      input: '',
      isLoading: false,
      isSessionLoading: false,
      models: [],
      selectedModel: null,
      attachedImages: [],
      copiedId: null,
      feedbackMap: {},
      isModelDropdownOpen: false,
      isSidebarOpen: true,
      isUserMenuOpen: false,

      setInput: (input) => set({ input }),
      setLoading: (isLoading) => set({ isLoading }),
      setSessionLoading: (isSessionLoading) => set({ isSessionLoading }),
      setActiveSessionId: (activeSessionId) => set({ activeSessionId }),
      setCopiedId: (copiedId) => set({ copiedId }),

      setFeedback: async (id, fb) => {
        const current = get().feedbackMap[id];
        const nextFb = current === fb ? undefined : fb;
        const nextMap = { ...get().feedbackMap };
        if (!nextFb) {
          delete nextMap[id];
        } else {
          nextMap[id] = nextFb;
        }
        set({ feedbackMap: nextMap });

        const isAuth = useAuthStore.getState().isAuthenticated;
        if (isAuth && id !== 'welcome' && !id.startsWith('temp-')) {
          try {
            await chatService.submitFeedback(id, nextFb || 'clear');
          } catch {}
        }
      },

      setIsModelDropdownOpen: (isModelDropdownOpen) => set({ isModelDropdownOpen }),
      toggleSidebar: () => set((state) => ({ isSidebarOpen: !state.isSidebarOpen })),
      setIsSidebarOpen: (isSidebarOpen) => set({ isSidebarOpen }),
      setIsUserMenuOpen: (isUserMenuOpen) => set({ isUserMenuOpen }),

      addMessage: (message) => {
        const isAuth = useAuthStore.getState().isAuthenticated;
        if (isAuth) {
          set((state) => ({
            messages: [...state.messages, message],
          }));
          return;
        }

        set((state) => {
          let sessionId = state.activeSessionId;
          let sessions = [...state.sessions];
          const newMessages = [...state.messages, message];
          const defaultTitle = i18n.t('chat.new_chat');

          if (!sessionId) {
            sessionId = `guest-session-${Date.now()}`;
            const title = message.sender === 'user' ? message.text.slice(0, 32) || defaultTitle : defaultTitle;
            sessions.unshift({
              id: sessionId,
              title,
              updatedAt: Date.now(),
              messages: newMessages,
            });
          } else {
            sessions = sessions.map((s) => {
              if (s.id === sessionId) {
                const isPlaceholder = s.title === defaultTitle || s.title === 'New Chat';
                const title = isPlaceholder && message.sender === 'user' ? message.text.slice(0, 32) || s.title : s.title;
                return { ...s, title, updatedAt: Date.now(), messages: newMessages };
              }
              return s;
            });
          }

          return { activeSessionId: sessionId, sessions, messages: newMessages };
        });
      },

      setMessages: (messages) => set({ messages }),

      clearMessages: () =>
        set({
          messages: [getInitialWelcomeMessage()],
          input: '',
          isLoading: false,
          attachedImages: [],
          copiedId: null,
          isModelDropdownOpen: false,
          activeSessionId: null,
        }),

      resetChat: () =>
        set({
          messages: [getInitialWelcomeMessage()],
          input: '',
          isLoading: false,
          attachedImages: [],
          copiedId: null,
          isModelDropdownOpen: false,
          activeSessionId: null,
        }),

      createNewChat: () =>
        set({
          messages: [getInitialWelcomeMessage()],
          input: '',
          isLoading: false,
          attachedImages: [],
          copiedId: null,
          isModelDropdownOpen: false,
          activeSessionId: null,
        }),

      selectSession: async (sessionId: string) => {
        const isAuth = useAuthStore.getState().isAuthenticated;
        if (isAuth) {
          set({ isSessionLoading: true, activeSessionId: sessionId });
          try {
            const detail = await chatService.getSession(sessionId);
            const msgs: ChatMessage[] = (detail.messages || []).map((m) => ({
              id: m.id,
              sender: m.sender,
              text: m.content,
              images: m.images,
              sources: m.sources,
              feedback: m.feedback,
              timestamp: new Date(m.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
            }));
            const feedbacks: Record<string, 'up' | 'down'> = {};
            for (const m of detail.messages || []) {
              if (m.feedback) {
                feedbacks[m.id] = m.feedback;
              }
            }
            set({
              messages: msgs.length > 0 ? msgs : [getInitialWelcomeMessage()],
              feedbackMap: { ...get().feedbackMap, ...feedbacks },
              isSessionLoading: false,
              input: '',
              isLoading: false,
              attachedImages: [],
              copiedId: null,
              isModelDropdownOpen: false,
            });
          } catch {
            set({ isSessionLoading: false });
          }
          return;
        }

        const session = get().sessions.find((s) => s.id === sessionId);
        if (!session) return;
        set({
          activeSessionId: sessionId,
          messages: session.messages,
          input: '',
          isLoading: false,
          attachedImages: [],
          copiedId: null,
          isModelDropdownOpen: false,
        });
      },

      deleteSession: async (sessionId: string) => {
        const isAuth = useAuthStore.getState().isAuthenticated;
        if (isAuth) {
          try {
            await chatService.deleteSession(sessionId);
          } catch {}
          const nextSessions = get().sessions.filter((s) => s.id !== sessionId);
          if (get().activeSessionId === sessionId) {
            set({
              sessions: nextSessions,
              activeSessionId: null,
              messages: [getInitialWelcomeMessage()],
            });
          } else {
            set({ sessions: nextSessions });
          }
          return;
        }

        const nextSessions = get().sessions.filter((s) => s.id !== sessionId);
        const nextActive = get().activeSessionId === sessionId ? null : get().activeSessionId;
        const nextMsgs = get().activeSessionId === sessionId ? [getInitialWelcomeMessage()] : get().messages;
        set({
          sessions: nextSessions,
          activeSessionId: nextActive,
          messages: nextMsgs,
        });
      },

      renameSession: async (sessionId: string, newTitle: string) => {
        const isAuth = useAuthStore.getState().isAuthenticated;
        if (isAuth) {
          try {
            await chatService.updateSessionTitle(sessionId, newTitle);
          } catch {}
        }
        const nextSessions = get().sessions.map((s) => (s.id === sessionId ? { ...s, title: newTitle } : s));
        set({ sessions: nextSessions });
      },

      setModels: (models) => {
        set((state) => {
          let selected = state.selectedModel;
          if (!selected && models.length > 0) {
            selected = models.find((m) => m.is_default) || models[0];
          }
          return { models, selectedModel: selected };
        });
      },

      setSelectedModel: (model) => set({ selectedModel: model, attachedImages: [], isModelDropdownOpen: false }),

      addAttachedImage: (base64) =>
        set((state) => ({
          attachedImages: [...state.attachedImages, base64],
        })),

      removeAttachedImage: (index) =>
        set((state) => ({
          attachedImages: state.attachedImages.filter((_, i) => i !== index),
        })),

      clearAttachedImages: () => set({ attachedImages: [] }),

      updateMessage: (id, patch) =>
        set((state) => {
          const newMessages = state.messages.map((m) => (m.id === id ? { ...m, ...patch } : m));
          const isAuth = useAuthStore.getState().isAuthenticated;
          if (!isAuth) {
            const nextSessions = state.sessions.map((s) =>
              s.id === state.activeSessionId ? { ...s, messages: newMessages, updatedAt: Date.now() } : s
            );
            return { messages: newMessages, sessions: nextSessions };
          }
          return { messages: newMessages };
        }),

      appendMessageChunk: (id, chunk) =>
        set((state) => ({
          messages: state.messages.map((m) => (m.id === id ? { ...m, text: m.text + chunk } : m)),
        })),

      appendMessageReasoning: (id, chunk) =>
        set((state) => ({
          messages: state.messages.map((m) =>
            m.id === id ? { ...m, reasoning: (m.reasoning || '') + chunk } : m
          ),
        })),

      updateMessageToolCall: (id, toolCall) =>
        set((state) => ({
          messages: state.messages.map((m) => {
            if (m.id !== id) return m;
            const existing = m.toolCalls || [];
            const idx = existing.findIndex((tc) => tc.id === toolCall.id || tc.name === toolCall.name);
            let nextCalls: ToolCallInfo[];
            if (idx >= 0) {
              nextCalls = [...existing];
              nextCalls[idx] = { ...nextCalls[idx], ...toolCall };
            } else {
              nextCalls = [...existing, toolCall];
            }
            return { ...m, toolCalls: nextCalls };
          }),
        })),

      syncAuthMode: async (isAuthenticated: boolean) => {
        if (isAuthenticated) {
          set({
            activeSessionId: null,
            messages: [getInitialWelcomeMessage()],
            sessions: [],
          });
          await get().loadUserSessions();
        } else {
          set({
            sessions: [],
            activeSessionId: null,
            messages: [getInitialWelcomeMessage()],
          });
        }
      },

      loadUserSessions: async () => {
        try {
          const res = await chatService.listSessions(1, 50);
          const sessions: ChatSession[] = (res.data || []).map((s) => ({
            id: s.id,
            title: s.title,
            updatedAt: new Date(s.updated_at).getTime(),
            modelId: s.model_id,
            messages: [],
          }));
          set({ sessions });
        } catch {}
      },
    }),
    {
      name: 'tluagent-chat-prefs',
      partialize: (state) => ({
        selectedModel: state.selectedModel,
        isSidebarOpen: state.isSidebarOpen,
      }),
    }
  )
);
