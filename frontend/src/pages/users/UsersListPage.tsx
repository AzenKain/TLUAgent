import React from 'react';
import { useUsers } from '@/hooks/useUsers';
import { useRoles } from '@/hooks/useRoles';
import { useAuthStore } from '@/stores/authStore';
import { usePermissions } from '@/hooks/usePermissions';
import { useUserManageStore } from '@/stores/userManageStore';
import { useShallow } from 'zustand/react/shallow';
import { useTranslation } from 'react-i18next';
import {
  Search,
  UserPlus,
  KeyRound,
  Shield,
  Edit2,
  Trash2,
  RotateCcw,
  CheckCircle2,
  AlertTriangle,
  User as UserIcon,
  Users,
} from 'lucide-react';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { Badge } from '@/components/common/Badge';
import { Modal } from '@/components/common/Modal';
import { ConfirmModal } from '@/components/common/ConfirmModal';
import { PasswordStrengthMeter } from '@/components/common/PasswordStrengthMeter';
import { PageHeader } from '@/components/common/PageHeader';
import { RefreshButton } from '@/components/common/RefreshButton';
import { getApiErrorMessage } from '@/utils/apiError';
import type { User } from '@/types';

const inputControlClass =
  'w-full min-h-11 px-3 py-2.5 rounded-lg border border-slate-200 dark:border-blue-900/40 bg-white dark:bg-[#111A35] text-slate-900 dark:text-slate-100 text-base focus:ring-2 focus:ring-tlu-blue focus:border-tlu-blue focus:outline-hidden';

const modalErrorClass =
  'p-3 rounded-lg bg-rose-50 dark:bg-rose-950/60 border border-rose-200 dark:border-rose-800 text-rose-700 dark:text-rose-300 text-xs flex items-start gap-2';

const modalSecondaryButtonClass =
  'min-h-11 w-full sm:w-auto px-4 py-2.5 text-sm font-medium rounded-lg text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 transition';

const modalPrimaryButtonClass =
  'min-h-11 w-full sm:w-auto px-4 py-2.5 text-sm font-semibold rounded-lg text-white transition disabled:opacity-50 bg-tlu-navy hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-blue-600 shadow-md shadow-blue-900/20';

interface UserRowActionsProps {
  user: User;
  isCurrent: boolean;
  canManageUsers: boolean;
  canManageRoles: boolean;
  isDeleting: boolean;
  isRestoring: boolean;
  onEdit: (user: User) => void;
  onRoles: (user: User) => void;
  onResetPass: (user: User) => void;
  onDelete: (user: User) => void;
  onRestore: (user: User) => void;
  variant: 'table' | 'card';
}

// Renders permission-gated user actions sized for touch in table or card layouts.
const UserRowActions: React.FC<UserRowActionsProps> = ({
  user,
  isCurrent,
  canManageUsers,
  canManageRoles,
  isDeleting,
  isRestoring,
  onEdit,
  onRoles,
  onResetPass,
  onDelete,
  onRestore,
  variant,
}) => {
  const { t } = useTranslation();
  const wrapperClass =
    variant === 'card' ? 'flex flex-wrap gap-1.5' : 'flex items-center justify-end gap-1';
  const iconButtonClass =
    variant === 'card'
      ? 'inline-flex h-11 w-11 items-center justify-center rounded-lg transition shrink-0'
      : 'inline-flex h-10 w-10 items-center justify-center rounded-lg transition shrink-0';

  if (user.is_deleted) {
    return (
      <div className={wrapperClass}>
        {canManageUsers && (
          <button
            onClick={() => onRestore(user)}
            disabled={isRestoring}
            title={t('common.restore')}
            aria-label={t('common.restore')}
            className="inline-flex min-h-11 items-center gap-1.5 px-3 py-2 rounded-lg text-xs font-semibold bg-emerald-50 dark:bg-emerald-950/60 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-100 transition disabled:opacity-50"
          >
            <RotateCcw className="w-4 h-4 shrink-0" />
            <span>{t('common.restore')}</span>
          </button>
        )}
      </div>
    );
  }

  return (
    <div className={wrapperClass}>
      {canManageUsers && (
        <button
          onClick={() => onEdit(user)}
          title={t('common.edit')}
          aria-label={t('common.edit')}
          className={`${iconButtonClass} text-slate-500 hover:text-tlu-blue hover:bg-blue-50 dark:hover:bg-blue-950/40`}
        >
          <Edit2 className="w-4 h-4" />
        </button>
      )}
      {canManageRoles && (
        <button
          onClick={() => onRoles(user)}
          title={t('users.modal_role_title', { name: user.full_name })}
          aria-label={t('users.modal_role_title', { name: user.full_name })}
          className={`${iconButtonClass} text-slate-500 hover:text-amber-600 hover:bg-amber-50 dark:hover:bg-amber-950/40`}
        >
          <Shield className="w-4 h-4" />
        </button>
      )}
      {canManageUsers && (
        <button
          onClick={() => onResetPass(user)}
          title={t('users.modal_reset_title', { name: user.full_name })}
          aria-label={t('users.modal_reset_title', { name: user.full_name })}
          className={`${iconButtonClass} text-slate-500 hover:text-tlu-blue hover:bg-blue-50 dark:hover:bg-blue-950/40`}
        >
          <KeyRound className="w-4 h-4" />
        </button>
      )}
      {canManageUsers && !isCurrent && !user.is_owner && (
        <button
          onClick={() => onDelete(user)}
          disabled={isDeleting}
          title={t('common.delete')}
          aria-label={t('common.delete')}
          className={`${iconButtonClass} text-slate-500 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 disabled:opacity-50`}
        >
          <Trash2 className="w-4 h-4" />
        </button>
      )}
    </div>
  );
};

// Renders the circular avatar with image or initial fallback.
const UserAvatar: React.FC<{ user: User; sizeClass?: string }> = ({
  user,
  sizeClass = 'h-9 w-9',
}) => (
  <div
    className={`${sizeClass} rounded-full bg-blue-100 dark:bg-blue-950 text-tlu-navy dark:text-blue-400 flex items-center justify-center font-bold text-xs uppercase border border-blue-200 dark:border-blue-900/60 shrink-0 overflow-hidden`}
  >
    {user.avatar_url ? (
      <img src={user.avatar_url} alt="" className="w-full h-full rounded-full object-cover" />
    ) : (
      user.full_name?.charAt(0) || <UserIcon className="w-4 h-4" />
    )}
  </div>
);

// Renders the role badge list with a dash placeholder when empty.
const UserRoleBadges: React.FC<{ user: User }> = ({ user }) => (
  <div className="flex flex-wrap items-center gap-1.5 min-w-0">
    {user.roles?.length > 0 ? (
      user.roles.map((r) => (
        <Badge
          key={r.id}
          variant={
            r.is_admin ? 'warning' : r.is_banned ? 'danger' : r.name === 'ADVISOR' ? 'primary' : 'secondary'
          }
          size="sm"
        >
          {r.name}
        </Badge>
      ))
    ) : (
      <span className="text-xs text-slate-400">—</span>
    )}
  </div>
);

// Admin user management page with filters, a responsive user list, and CRUD modals.
export const UsersListPage: React.FC = () => {
  const currentAuthUser = useAuthStore((s) => s.user);
  const { can, isAdmin } = usePermissions();
  const { t } = useTranslation();

  const canManageUsers = can('user.manage') || isAdmin;
  const canManageRoles = can('role.manage') || isAdmin;

  const {
    search,
    roleFilter,
    isDeletedFilter,
    page,
    selectedUser,
    isCreateModalOpen,
    isEditModalOpen,
    isRoleModalOpen,
    isResetPassModalOpen,
    createForm,
    editForm,
    selectedRoleIds,
    newPassword,
    resetPassConfirm,
    modalError,
    toastMessage,
    setSearch,
    setRoleFilter,
    setIsDeletedFilter,
    setPage,
    openCreateModal,
    closeCreateModal,
    setCreateForm,
    toggleCreateRoleId,
    openEditModal,
    closeEditModal,
    setEditForm,
    openRoleModal,
    closeRoleModal,
    toggleRoleId,
    openResetPassModal,
    closeResetPassModal,
    setNewPassword,
    setResetPassConfirm,
    setModalError,
    showToast,
  } = useUserManageStore(
    useShallow((s) => ({
      search: s.search,
      roleFilter: s.roleFilter,
      isDeletedFilter: s.isDeletedFilter,
      page: s.page,
      selectedUser: s.selectedUser,
      isCreateModalOpen: s.isCreateModalOpen,
      isEditModalOpen: s.isEditModalOpen,
      isRoleModalOpen: s.isRoleModalOpen,
      isResetPassModalOpen: s.isResetPassModalOpen,
      createForm: s.createForm,
      editForm: s.editForm,
      selectedRoleIds: s.selectedRoleIds,
      newPassword: s.newPassword,
      resetPassConfirm: s.resetPassConfirm,
      modalError: s.modalError,
      toastMessage: s.toastMessage,
      setSearch: s.setSearch,
      setRoleFilter: s.setRoleFilter,
      setIsDeletedFilter: s.setIsDeletedFilter,
      setPage: s.setPage,
      openCreateModal: s.openCreateModal,
      closeCreateModal: s.closeCreateModal,
      setCreateForm: s.setCreateForm,
      toggleCreateRoleId: s.toggleCreateRoleId,
      openEditModal: s.openEditModal,
      closeEditModal: s.closeEditModal,
      setEditForm: s.setEditForm,
      openRoleModal: s.openRoleModal,
      closeRoleModal: s.closeRoleModal,
      toggleRoleId: s.toggleRoleId,
      openResetPassModal: s.openResetPassModal,
      closeResetPassModal: s.closeResetPassModal,
      setNewPassword: s.setNewPassword,
      setResetPassConfirm: s.setResetPassConfirm,
      setModalError: s.setModalError,
      showToast: s.showToast,
    }))
  );

  const {
    users,
    pagination,
    isLoading,
    isFetching,
    refetch,
    createUser,
    isCreating,
    updateUser,
    isUpdating,
    changeRole,
    isChangingRole,
    resetPassword,
    isResettingPassword,
    deleteUser,
    isDeleting,
    restoreUser,
    isRestoring,
  } = useUsers({
    page,
    limit: 10,
    search: search.trim() || undefined,
    role_id: roleFilter || undefined,
    is_deleted: isDeletedFilter,
  });

  const { roles } = useRoles();

  const handleCreateSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setModalError('');
    if (createForm.password.length < 10) {
      setModalError(t('users.password_min_length_error'));
      return;
    }
    if (createForm.password !== createForm.confirm_password) {
      setModalError(t('users.password_mismatch'));
      return;
    }
    try {
      await createUser({
        email: createForm.email,
        password: createForm.password,
        full_name: createForm.full_name,
        student_code: createForm.student_code.trim() ? createForm.student_code.trim() : undefined,
        role_ids: createForm.role_ids.length > 0 ? createForm.role_ids : undefined,
      });
      closeCreateModal();
      showToast(t('users.toast_created'));
    } catch (err: unknown) {
      const errorMsg = getApiErrorMessage(err, t('users.error_create_failed'));
      setModalError(errorMsg);
    }
  };

  const handleEditSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedUser) return;
    setModalError('');
    try {
      await updateUser({ id: selectedUser.id, data: editForm });
      closeEditModal();
      showToast(t('users.toast_updated'));
    } catch (err: unknown) {
      const errorMsg = getApiErrorMessage(err, t('users.error_update_failed'));
      setModalError(errorMsg);
    }
  };

  const handleRoleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedUser) return;
    setModalError('');
    try {
      await changeRole({ id: selectedUser.id, data: { role_ids: selectedRoleIds } });
      closeRoleModal();
      showToast(t('users.toast_role_updated'));
    } catch (err: unknown) {
      const errorMsg = getApiErrorMessage(err, t('users.error_role_failed'));
      setModalError(errorMsg);
    }
  };

  const handleResetPassSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedUser) return;
    setModalError('');
    if (newPassword.length < 10) {
      setModalError(t('users.password_min_length_error'));
      return;
    }
    if (newPassword !== resetPassConfirm) {
      setModalError(t('users.password_mismatch'));
      return;
    }
    try {
      await resetPassword({ id: selectedUser.id, data: { new_password: newPassword } });
      closeResetPassModal();
      showToast(t('users.toast_password_reset'));
    } catch (err: unknown) {
      const errorMsg = getApiErrorMessage(err, t('users.error_reset_failed'));
      setModalError(errorMsg);
    }
  };

  const [deletingUser, setDeletingUser] = React.useState<User | null>(null);
  const [actionError, setActionError] = React.useState<string | null>(null);

  const handleDelete = async (u: User) => {
    setDeletingUser(u);
  };

  const confirmDelete = async () => {
    if (!deletingUser) return;
    try {
      await deleteUser(deletingUser.id);
      showToast(t('users.toast_deleted'));
      setDeletingUser(null);
    } catch (err: unknown) {
      const errorMsg = getApiErrorMessage(err, t('users.error_delete_failed'));
      setActionError(errorMsg);
    }
  };

  const handleRestore = async (u: User) => {
    try {
      await restoreUser(u.id);
      showToast(t('users.toast_restored'));
    } catch (err: unknown) {
      const errorMsg = getApiErrorMessage(err, t('users.error_restore_failed'));
      setActionError(errorMsg);
    }
  };

  const renderUserActions = (u: User, isCurrent: boolean, variant: 'table' | 'card') => (
    <UserRowActions
      user={u}
      isCurrent={isCurrent}
      canManageUsers={canManageUsers}
      canManageRoles={canManageRoles}
      isDeleting={isDeleting}
      isRestoring={isRestoring}
      onEdit={openEditModal}
      onRoles={openRoleModal}
      onResetPass={openResetPassModal}
      onDelete={handleDelete}
      onRestore={handleRestore}
      variant={variant}
    />
  );

  return (
    <div className="space-y-6 max-w-7xl mx-auto pb-10">
      {toastMessage && (
        <div className="fixed top-20 inset-x-4 sm:inset-x-auto sm:right-6 z-50 bg-emerald-600 text-white px-4 py-3 rounded-xl shadow-lg flex items-center gap-2 text-sm font-medium animate-in fade-in slide-in-from-top-3 duration-200">
          <CheckCircle2 className="w-4 h-4 shrink-0" />
          <span className="min-w-0 wrap-break-word">{toastMessage}</span>
        </div>
      )}

      <PageHeader
        icon={<Users className="h-6 w-6" />}
        title={t('users.title')}
        subtitle={t('users.subtitle')}
        actions={
          <div className="flex items-center gap-2">
            <RefreshButton
              onClick={() => refetch()}
              isLoading={isFetching}
            />
            {canManageUsers && (
              <button
                onClick={openCreateModal}
                aria-label={t('users.create_button')}
                className="inline-flex items-center justify-center gap-2 px-4 min-h-10 rounded-xl bg-tlu-navy hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-blue-600 text-white font-semibold text-xs sm:text-sm shadow-xs transition shrink-0 cursor-pointer"
              >
                <UserPlus className="w-4 h-4 shrink-0" />
                <span>{t('users.create_button')}</span>
              </button>
            )}
          </div>
        }
      />

      <div className="grid grid-cols-1 md:grid-cols-3 gap-3 bg-white dark:bg-[#111A35] p-4 rounded-xl border border-slate-200 dark:border-blue-900/30 shadow-xs">
        <div className="relative">
          <Search className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t('users.search_placeholder')}
            className={`${inputControlClass} pl-10`}
          />
        </div>

        <select
          value={roleFilter}
          onChange={(e) => setRoleFilter(e.target.value)}
          className={inputControlClass}
        >
          <option value="">{t('users.all_roles')}</option>
          {roles.map((r) => (
            <option key={r.id} value={r.id}>
              {r.name}
            </option>
          ))}
        </select>

        <select
          value={isDeletedFilter === undefined ? '' : String(isDeletedFilter)}
          onChange={(e) => {
            const val = e.target.value;
            setIsDeletedFilter(val === '' ? undefined : val === 'true');
          }}
          className={inputControlClass}
        >
          <option value="">{t('users.all_status')}</option>
          <option value="false">{t('users.status_active')}</option>
          <option value="true">{t('users.status_deleted')}</option>
        </select>
      </div>

      <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 overflow-hidden shadow-xs">
        {isLoading ? (
          <LoadingSpinner size="lg" className="py-24" />
        ) : users.length === 0 ? (
          <div className="py-16 px-4 text-center text-slate-400 text-sm">{t('users.empty')}</div>
        ) : (
          <>
            <div className="hidden md:block overflow-x-auto">
              <table className="w-full min-w-240 text-left text-sm">
                <thead className="bg-slate-50 dark:bg-slate-950/60 text-slate-500 dark:text-slate-400 text-xs uppercase tracking-wider border-b border-slate-200 dark:border-slate-800 font-semibold">
                  <tr>
                    <th className="px-6 py-3.5 whitespace-nowrap">{t('users.col_user')}</th>
                    <th className="px-6 py-3.5 whitespace-nowrap">{t('users.col_code')}</th>
                    <th className="px-6 py-3.5 whitespace-nowrap">{t('users.col_roles')}</th>
                    <th className="px-6 py-3.5 whitespace-nowrap">{t('users.col_status')}</th>
                    <th className="px-6 py-3.5 whitespace-nowrap text-right">
                      {t('users.col_actions')}
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-slate-800/80">
                  {users.map((u) => {
                    const isCurrent = currentAuthUser?.id === u.id;
                    return (
                      <tr
                        key={u.id}
                        className={`hover:bg-slate-50/60 dark:hover:bg-slate-800/40 transition ${
                          u.is_deleted ? 'opacity-60 bg-slate-50/30 dark:bg-slate-950/30' : ''
                        }`}
                      >
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-3 min-w-0">
                            <UserAvatar user={u} />
                            <div className="min-w-0">
                              <div className="font-semibold text-slate-900 dark:text-slate-100 flex items-center gap-1.5">
                                <span className="min-w-0 truncate max-w-55">{u.full_name}</span>
                                {u.is_owner && (
                                  <span className="shrink-0 text-[10px] bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/30 px-1 rounded font-bold">
                                    {t('common.owner')}
                                  </span>
                                )}
                                {isCurrent && (
                                  <span className="shrink-0 text-[10px] bg-blue-500/15 text-tlu-blue dark:text-blue-400 border border-blue-500/30 px-1 rounded font-bold">
                                    {t('common.you')}
                                  </span>
                                )}
                              </div>
                              <div className="text-xs text-slate-500 dark:text-slate-400 truncate max-w-60">
                                {u.email}
                              </div>
                            </div>
                          </div>
                        </td>

                        <td className="px-6 py-4 text-xs font-mono text-slate-600 dark:text-slate-300 whitespace-nowrap">
                          {u.student_code || '—'}
                        </td>

                        <td className="px-6 py-4">
                          <UserRoleBadges user={u} />
                        </td>

                        <td className="px-6 py-4">
                          {u.is_deleted ? (
                            <Badge variant="danger" size="sm" className="whitespace-nowrap">
                              {t('common.deleted')}
                            </Badge>
                          ) : (
                            <Badge variant="success" size="sm" className="whitespace-nowrap">
                              {t('common.active')}
                            </Badge>
                          )}
                        </td>

                        <td className="px-6 py-4 text-right">
                          {renderUserActions(u, isCurrent, 'table')}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>

            <div className="md:hidden divide-y divide-slate-100 dark:divide-slate-800/80">
              {users.map((u) => {
                const isCurrent = currentAuthUser?.id === u.id;
                return (
                  <div
                    key={u.id}
                    className={`p-4 space-y-3 ${
                      u.is_deleted ? 'opacity-60 bg-slate-50/30 dark:bg-slate-950/30' : ''
                    }`}
                  >
                    <div className="flex items-start gap-3 min-w-0">
                      <UserAvatar user={u} sizeClass="h-10 w-10" />
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-1.5 min-w-0">
                          <span className="min-w-0 truncate font-semibold text-sm text-slate-900 dark:text-slate-100">
                            {u.full_name}
                          </span>
                          {u.is_owner && (
                            <span className="shrink-0 text-[10px] bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/30 px-1 rounded font-bold">
                              {t('common.owner')}
                            </span>
                          )}
                          {isCurrent && (
                            <span className="shrink-0 text-[10px] bg-blue-500/15 text-tlu-blue dark:text-blue-400 border border-blue-500/30 px-1 rounded font-bold">
                              {t('common.you')}
                            </span>
                          )}
                        </div>
                        <div className="text-xs text-slate-500 dark:text-slate-400 truncate">
                          {u.email}
                        </div>
                        {u.student_code && (
                          <div className="text-xs font-mono text-slate-600 dark:text-slate-300 mt-0.5 truncate">
                            {u.student_code}
                          </div>
                        )}
                      </div>
                      {u.is_deleted ? (
                        <Badge variant="danger" size="sm" className="shrink-0 mt-0.5 whitespace-nowrap">
                          {t('common.deleted')}
                        </Badge>
                      ) : (
                        <Badge variant="success" size="sm" className="shrink-0 mt-0.5 whitespace-nowrap">
                          {t('common.active')}
                        </Badge>
                      )}
                    </div>

                    <UserRoleBadges user={u} />

                    {renderUserActions(u, isCurrent, 'card')}
                  </div>
                );
              })}
            </div>
          </>
        )}

        {pagination && pagination.total_pages > 1 && (
          <div className="px-4 sm:px-6 py-3 sm:py-4 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between gap-3 text-xs text-slate-500 dark:text-slate-400">
            <div className="min-w-0 truncate">
              <span className="sm:hidden">
                {t('common.page_compact', {
                  current: pagination.current_page,
                  total: pagination.total_pages,
                })}
              </span>
              <span className="hidden sm:inline">
                {t('common.page', {
                  current: pagination.current_page,
                  total: pagination.total_pages,
                  count: pagination.total_records,
                })}
              </span>
            </div>
            <div className="flex gap-2 shrink-0">
              <button
                disabled={page <= 1}
                onClick={() => setPage(Math.max(1, page - 1))}
                className="min-h-11 px-4 sm:px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-800 font-medium hover:bg-slate-50 dark:hover:bg-slate-800 transition disabled:opacity-40 disabled:hover:bg-transparent"
              >
                {t('common.previous')}
              </button>
              <button
                disabled={page >= pagination.total_pages}
                onClick={() => setPage(page + 1)}
                className="min-h-11 px-4 sm:px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-800 font-medium hover:bg-slate-50 dark:hover:bg-slate-800 transition disabled:opacity-40 disabled:hover:bg-transparent"
              >
                {t('common.next')}
              </button>
            </div>
          </div>
        )}
      </div>

      <Modal
        isOpen={isCreateModalOpen}
        onClose={closeCreateModal}
        title={t('users.modal_create_title')}
      >
        <form onSubmit={handleCreateSubmit} className="space-y-4">
          {modalError && (
            <div className={modalErrorClass}>
              <AlertTriangle className="w-4 h-4 shrink-0" />
              <span className="min-w-0 wrap-break-word">{modalError}</span>
            </div>
          )}

          <div>
            <label htmlFor="create-user-email" className="block text-xs font-semibold mb-1">
              {t('auth.email')}
            </label>
            <input
              id="create-user-email"
              type="email"
              required
              value={createForm.email}
              onChange={(e) => setCreateForm({ email: e.target.value })}
              className={inputControlClass}
            />
          </div>

          <div>
            <label htmlFor="create-user-password" className="block text-xs font-semibold mb-1">
              {t('users.modal_initial_password')}
            </label>
            <input
              id="create-user-password"
              type="password"
              required
              minLength={10}
              value={createForm.password}
              onChange={(e) => setCreateForm({ password: e.target.value })}
              className={inputControlClass}
            />
            <PasswordStrengthMeter password={createForm.password} />
          </div>

          <div>
            <label htmlFor="create-user-confirm-password" className="block text-xs font-semibold mb-1">
              {t('users.modal_confirm_password')}
            </label>
            <input
              id="create-user-confirm-password"
              type="password"
              required
              minLength={10}
              value={createForm.confirm_password}
              onChange={(e) => setCreateForm({ confirm_password: e.target.value })}
              placeholder={t('users.modal_confirm_password_placeholder')}
              className={inputControlClass}
            />
          </div>

          <div>
            <label htmlFor="create-user-fullname" className="block text-xs font-semibold mb-1">
              {t('users.modal_fullname')}
            </label>
            <input
              id="create-user-fullname"
              type="text"
              required
              value={createForm.full_name}
              onChange={(e) => setCreateForm({ full_name: e.target.value })}
              className={inputControlClass}
            />
          </div>

          <div>
            <label htmlFor="create-user-code" className="block text-xs font-semibold mb-1">
              {t('users.modal_student_code')}
            </label>
            <input
              id="create-user-code"
              type="text"
              value={createForm.student_code}
              onChange={(e) => setCreateForm({ student_code: e.target.value })}
              className={inputControlClass}
            />
          </div>

          <div>
            <div className="block text-xs font-semibold mb-1.5">{t('users.modal_assign_roles')}</div>
            <div className="space-y-1.5 border border-slate-100 dark:border-slate-800 p-3 rounded-lg max-h-48 sm:max-h-40 overflow-y-auto">
              {roles.map((r) => {
                const isChecked = createForm.role_ids.includes(r.id);
                return (
                  <label
                    key={r.id}
                    className={`flex items-center gap-2.5 p-2 rounded-lg text-xs cursor-pointer transition-colors ${
                      isChecked
                        ? 'bg-blue-50/70 dark:bg-blue-950/40 text-tlu-navy dark:text-blue-200 font-medium'
                        : 'hover:bg-slate-50 dark:hover:bg-slate-850 text-slate-700 dark:text-slate-300'
                    }`}
                  >
                    <input
                      type="checkbox"
                      checked={isChecked}
                      onChange={() => toggleCreateRoleId(r.id)}
                      className="h-4 w-4 shrink-0 rounded accent-tlu-blue"
                    />
                    <span className="min-w-0 wrap-break-word">{r.name}</span>
                    {r.description && (
                      <span className="text-[11px] text-slate-400 dark:text-slate-500 font-normal">
                        ({r.description})
                      </span>
                    )}
                  </label>
                );
              })}
            </div>
          </div>

          <div className="flex flex-col-reverse gap-2 pt-2 sm:flex-row sm:justify-end">
            <button type="button" onClick={closeCreateModal} className={modalSecondaryButtonClass}>
              {t('common.cancel')}
            </button>
            <button type="submit" disabled={isCreating} className={modalPrimaryButtonClass}>
              {isCreating ? t('common.loading') : t('users.create_button')}
            </button>
          </div>
        </form>
      </Modal>

      <Modal
        isOpen={isEditModalOpen}
        onClose={closeEditModal}
        title={t('users.modal_edit_title')}
      >
        <form onSubmit={handleEditSubmit} className="space-y-4">
          {modalError && (
            <div className={modalErrorClass}>
              <AlertTriangle className="w-4 h-4 shrink-0" />
              <span className="min-w-0 wrap-break-word">{modalError}</span>
            </div>
          )}

          <div>
            <label htmlFor="edit-user-fullname" className="block text-xs font-semibold mb-1">
              {t('users.modal_fullname')}
            </label>
            <input
              id="edit-user-fullname"
              type="text"
              required
              value={editForm.full_name}
              onChange={(e) => setEditForm({ full_name: e.target.value })}
              className={inputControlClass}
            />
          </div>

          <div>
            <label htmlFor="edit-user-code" className="block text-xs font-semibold mb-1">
              {t('users.modal_student_code')}
            </label>
            <input
              id="edit-user-code"
              type="text"
              value={editForm.student_code}
              onChange={(e) => setEditForm({ student_code: e.target.value })}
              className={inputControlClass}
            />
          </div>

          <div className="flex flex-col-reverse gap-2 pt-2 sm:flex-row sm:justify-end">
            <button type="button" onClick={closeEditModal} className={modalSecondaryButtonClass}>
              {t('common.cancel')}
            </button>
            <button type="submit" disabled={isUpdating} className={modalPrimaryButtonClass}>
              {isUpdating ? t('common.saving') : t('common.save')}
            </button>
          </div>
        </form>
      </Modal>

      <Modal
        isOpen={isRoleModalOpen}
        onClose={closeRoleModal}
        title={t('users.modal_role_title', { name: selectedUser?.full_name || '' })}
      >
        <form onSubmit={handleRoleSubmit} className="space-y-4">
          {modalError && (
            <div className={modalErrorClass}>
              <AlertTriangle className="w-4 h-4 shrink-0" />
              <span className="min-w-0 wrap-break-word">{modalError}</span>
            </div>
          )}

          <div className="space-y-1 border border-slate-100 dark:border-slate-800 p-3 rounded-lg max-h-56 sm:max-h-48 overflow-y-auto">
            {roles.map((r) => (
              <label key={r.id} className="flex items-center gap-2.5 min-h-11 text-xs cursor-pointer">
                <input
                  type="checkbox"
                  checked={selectedRoleIds.includes(r.id)}
                  onChange={() => toggleRoleId(r.id)}
                  className="h-4 w-4 shrink-0 rounded accent-tlu-blue"
                />
                <span className="font-medium text-slate-800 dark:text-slate-200 shrink-0">{r.name}</span>
                <span className="text-[11px] text-slate-400 min-w-0 wrap-break-word">({r.description})</span>
              </label>
            ))}
          </div>

          <div className="flex flex-col-reverse gap-2 pt-2 sm:flex-row sm:justify-end">
            <button type="button" onClick={closeRoleModal} className={modalSecondaryButtonClass}>
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              disabled={isChangingRole || selectedRoleIds.length === 0}
              className={modalPrimaryButtonClass}
            >
              {isChangingRole ? t('common.saving') : t('common.save')}
            </button>
          </div>
        </form>
      </Modal>

      <Modal
        isOpen={isResetPassModalOpen}
        onClose={closeResetPassModal}
        title={t('users.modal_reset_title', { name: selectedUser?.full_name || '' })}
      >
        <form onSubmit={handleResetPassSubmit} className="space-y-4">
          {modalError && (
            <div className={modalErrorClass}>
              <AlertTriangle className="w-4 h-4 shrink-0" />
              <span className="min-w-0 wrap-break-word">{modalError}</span>
            </div>
          )}

          <div>
            <label htmlFor="reset-user-password" className="block text-xs font-semibold mb-1">
              {t('users.new_password_label')}
            </label>
            <input
              id="reset-user-password"
              type="password"
              required
              minLength={10}
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder={t('users.new_password_placeholder')}
              className={inputControlClass}
            />
            <PasswordStrengthMeter password={newPassword} />
          </div>

          <div>
            <label htmlFor="reset-user-confirm-password" className="block text-xs font-semibold mb-1">
              {t('users.confirm_new_password_label')}
            </label>
            <input
              id="reset-user-confirm-password"
              type="password"
              required
              minLength={10}
              value={resetPassConfirm}
              onChange={(e) => setResetPassConfirm(e.target.value)}
              placeholder={t('users.confirm_new_password_placeholder')}
              className={inputControlClass}
            />
          </div>

          <div className="flex flex-col-reverse gap-2 pt-2 sm:flex-row sm:justify-end">
            <button type="button" onClick={closeResetPassModal} className={modalSecondaryButtonClass}>
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              disabled={isResettingPassword}
              className={modalPrimaryButtonClass}
            >
              {isResettingPassword ? t('common.loading') : t('users.confirm_reset_action')}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmModal
        isOpen={Boolean(deletingUser)}
        onClose={() => setDeletingUser(null)}
        onConfirm={confirmDelete}
        title={t('common.delete')}
        message={deletingUser ? t('users.confirm_delete', { name: deletingUser.full_name }) : ''}
        variant="danger"
        isLoading={isDeleting}
      />

      <ConfirmModal
        isOpen={Boolean(actionError)}
        onClose={() => setActionError(null)}
        title={t('common.error')}
        message={actionError || ''}
        variant="warning"
        confirmText={t('common.close')}
      />
    </div>
  );
};
