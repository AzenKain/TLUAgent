import { create } from 'zustand';
import type { User } from '@/types';

interface CreateUserFormData {
  email: string;
  password: string;
  full_name: string;
  student_code: string;
  role_ids: string[];
}

interface EditUserFormData {
  full_name: string;
  student_code: string;
}

interface UserManageState {
  // Filter & pagination
  search: string;
  roleFilter: string;
  isDeletedFilter: boolean | undefined;
  page: number;
  limit: number;

  // Selected user for action modals
  selectedUser: User | null;

  // Modals visibility
  isCreateModalOpen: boolean;
  isEditModalOpen: boolean;
  isRoleModalOpen: boolean;
  isResetPassModalOpen: boolean;

  // Form values
  createForm: CreateUserFormData;
  editForm: EditUserFormData;
  selectedRoleIds: string[];
  newPassword: string;

  // Error and Toast
  modalError: string;
  toastMessage: string | null;

  // Actions - Filters
  setSearch: (search: string) => void;
  setRoleFilter: (roleId: string) => void;
  setIsDeletedFilter: (isDeleted: boolean | undefined) => void;
  setPage: (page: number) => void;
  resetFilters: () => void;

  // Actions - Modals & Forms
  openCreateModal: () => void;
  closeCreateModal: () => void;
  setCreateForm: (patch: Partial<CreateUserFormData>) => void;

  openEditModal: (user: User) => void;
  closeEditModal: () => void;
  setEditForm: (patch: Partial<EditUserFormData>) => void;

  openRoleModal: (user: User) => void;
  closeRoleModal: () => void;
  toggleRoleId: (roleId: string) => void;

  openResetPassModal: (user: User) => void;
  closeResetPassModal: () => void;
  setNewPassword: (password: string) => void;

  setModalError: (error: string) => void;
  showToast: (message: string) => void;
  clearToast: () => void;
}

const initialCreateForm: CreateUserFormData = {
  email: '',
  password: '',
  full_name: '',
  student_code: '',
  role_ids: [],
};

const initialEditForm: EditUserFormData = {
  full_name: '',
  student_code: '',
};

let toastTimer: ReturnType<typeof setTimeout> | null = null;

export const useUserManageStore = create<UserManageState>((set) => ({
  // Initial filter state
  search: '',
  roleFilter: '',
  isDeletedFilter: undefined,
  page: 1,
  limit: 10,

  selectedUser: null,

  isCreateModalOpen: false,
  isEditModalOpen: false,
  isRoleModalOpen: false,
  isResetPassModalOpen: false,

  createForm: { ...initialCreateForm },
  editForm: { ...initialEditForm },
  selectedRoleIds: [],
  newPassword: '',

  modalError: '',
  toastMessage: null,

  // Filters actions
  setSearch: (search) => set({ search, page: 1 }),
  setRoleFilter: (roleFilter) => set({ roleFilter, page: 1 }),
  setIsDeletedFilter: (isDeletedFilter) => set({ isDeletedFilter, page: 1 }),
  setPage: (page) => set({ page }),
  resetFilters: () => set({ search: '', roleFilter: '', isDeletedFilter: undefined, page: 1 }),

  // Create Modal
  openCreateModal: () =>
    set({
      isCreateModalOpen: true,
      createForm: { ...initialCreateForm },
      modalError: '',
    }),
  closeCreateModal: () => set({ isCreateModalOpen: false, modalError: '' }),
  setCreateForm: (patch) =>
    set((state) => ({
      createForm: { ...state.createForm, ...patch },
    })),

  // Edit Modal
  openEditModal: (user) =>
    set({
      selectedUser: user,
      isEditModalOpen: true,
      editForm: {
        full_name: user.full_name,
        student_code: user.student_code || '',
      },
      modalError: '',
    }),
  closeEditModal: () => set({ isEditModalOpen: false, selectedUser: null, modalError: '' }),
  setEditForm: (patch) =>
    set((state) => ({
      editForm: { ...state.editForm, ...patch },
    })),

  // Role Modal
  openRoleModal: (user) =>
    set({
      selectedUser: user,
      isRoleModalOpen: true,
      selectedRoleIds: user.roles?.map((r) => r.id) || [],
      modalError: '',
    }),
  closeRoleModal: () => set({ isRoleModalOpen: false, selectedUser: null, modalError: '' }),
  toggleRoleId: (roleId) =>
    set((state) => {
      const exists = state.selectedRoleIds.includes(roleId);
      return {
        selectedRoleIds: exists
          ? state.selectedRoleIds.filter((id) => id !== roleId)
          : [...state.selectedRoleIds, roleId],
      };
    }),

  // Reset Password Modal
  openResetPassModal: (user) =>
    set({
      selectedUser: user,
      isResetPassModalOpen: true,
      newPassword: '',
      modalError: '',
    }),
  closeResetPassModal: () =>
    set({ isResetPassModalOpen: false, selectedUser: null, modalError: '', newPassword: '' }),
  setNewPassword: (newPassword) => set({ newPassword }),

  // Errors & Toasts
  setModalError: (modalError) => set({ modalError }),
  showToast: (message) => {
    if (toastTimer) {
      clearTimeout(toastTimer);
    }
    set({ toastMessage: message });
    toastTimer = setTimeout(() => {
      set({ toastMessage: null });
    }, 3000);
  },
  clearToast: () => {
    if (toastTimer) {
      clearTimeout(toastTimer);
    }
    set({ toastMessage: null });
  },
}));
