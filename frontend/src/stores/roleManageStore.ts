import { create } from 'zustand';
import type { Role, PermissionAssignment } from '@/types';

interface RoleFormState {
  name: string;
  description: string;
  auto_assign: boolean;
}

interface RoleManageState {
  selectedRole: Role | null;
  assignments: PermissionAssignment[];
  showModal: boolean;
  modalMode: 'create' | 'edit';
  form: RoleFormState;
  roleToDelete: Role | null;
  errorMsg: string;
  permSearch: string;
  expandedCategories: Record<string, boolean>;
  toastMessage: string | null;

  setSelectedRole: (role: Role | null | ((prev: Role | null) => Role | null)) => void;
  setAssignments: (
    assignments: PermissionAssignment[] | ((prev: PermissionAssignment[]) => PermissionAssignment[])
  ) => void;
  setShowModal: (show: boolean) => void;
  setModalMode: (mode: 'create' | 'edit') => void;
  setForm: (form: RoleFormState | ((prev: RoleFormState) => RoleFormState)) => void;
  setRoleToDelete: (role: Role | null) => void;
  setErrorMsg: (error: string) => void;
  setPermSearch: (search: string) => void;
  toggleCategory: (categoryId: string) => void;
  setAllCategories: (expanded: boolean, categoryIds: string[]) => void;
  setToastMessage: (msg: string | null) => void;
  reset: () => void;
}

const initialForm: RoleFormState = {
  name: '',
  description: '',
  auto_assign: false,
};

let toastTimeout: ReturnType<typeof setTimeout> | null = null;

export const useRoleManageStore = create<RoleManageState>((set) => ({
  selectedRole: null,
  assignments: [],
  showModal: false,
  modalMode: 'create',
  form: { ...initialForm },
  roleToDelete: null,
  errorMsg: '',
  permSearch: '',
  expandedCategories: {},
  toastMessage: null,

  setSelectedRole: (selectedRole) =>
    set((state) => ({
      selectedRole:
        typeof selectedRole === 'function' ? selectedRole(state.selectedRole) : selectedRole,
    })),

  setAssignments: (assignments) =>
    set((state) => ({
      assignments:
        typeof assignments === 'function' ? assignments(state.assignments) : assignments,
    })),

  setShowModal: (showModal) => set({ showModal }),
  setModalMode: (modalMode) => set({ modalMode }),

  setForm: (form) =>
    set((state) => ({
      form: typeof form === 'function' ? form(state.form) : form,
    })),

  setRoleToDelete: (roleToDelete) => set({ roleToDelete }),
  setErrorMsg: (errorMsg) => set({ errorMsg }),
  setPermSearch: (permSearch) => set({ permSearch }),

  toggleCategory: (categoryId) =>
    set((state) => ({
      expandedCategories: {
        ...state.expandedCategories,
        [categoryId]: !state.expandedCategories[categoryId],
      },
    })),

  setAllCategories: (expanded, categoryIds) =>
    set(() => {
      const next: Record<string, boolean> = {};
      for (const id of categoryIds) {
        next[id] = expanded;
      }
      return { expandedCategories: next };
    }),

  setToastMessage: (msg) => {
    if (toastTimeout) {
      clearTimeout(toastTimeout);
      toastTimeout = null;
    }
    set({ toastMessage: msg });
    if (msg) {
      toastTimeout = setTimeout(() => {
        set({ toastMessage: null });
        toastTimeout = null;
      }, 3500);
    }
  },

  reset: () =>
    set({
      selectedRole: null,
      assignments: [],
      showModal: false,
      modalMode: 'create',
      form: { ...initialForm },
      roleToDelete: null,
      errorMsg: '',
      permSearch: '',
      expandedCategories: {},
      toastMessage: null,
    }),
}));
