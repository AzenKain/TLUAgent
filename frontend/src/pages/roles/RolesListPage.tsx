import React, { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/react/shallow';
import {
  Shield,
  ShieldAlert,
  Pencil,
  Trash2,
  ChevronUp,
  ChevronDown,
  ChevronsUp,
  ChevronsDown,
  GripVertical,
  Plus,
  RefreshCw,
  Save,
  Search,
  X,
  Loader2,
  AlertCircle,
  CheckCircle2,
  MessageSquare,
  BookOpen,
  GraduationCap,
  Settings,
} from 'lucide-react';
import { useRoles } from '@/hooks/useRoles';
import { useRoleManageStore } from '@/stores/roleManageStore';
import { Modal } from '@/components/common/Modal';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import type { PermissionAssignment, Permission } from '@/types';

interface PermCategory {
  id: string;
  nameKey: string;
  icon: React.ComponentType<{ className?: string }>;
  keys: string[];
}

const PERM_CATEGORIES: PermCategory[] = [
  {
    id: 'advisory',
    nameKey: 'roles.perm_cat_advisory',
    icon: MessageSquare,
    keys: ['chat.ask', 'chat.persist', 'chat.manage'],
  },
  {
    id: 'regulation',
    nameKey: 'roles.perm_cat_regulation',
    icon: BookOpen,
    keys: ['regulation.read', 'regulation.write'],
  },
  {
    id: 'curriculum',
    nameKey: 'roles.perm_cat_curriculum',
    icon: GraduationCap,
    keys: ['curriculum.read', 'curriculum.manage'],
  },
  {
    id: 'crawler',
    nameKey: 'roles.perm_cat_crawler',
    icon: RefreshCw,
    keys: ['crawler.trigger'],
  },
  {
    id: 'system',
    nameKey: 'roles.perm_cat_system',
    icon: Settings,
    keys: [
      'admin.access',
      'user.manage',
      'role.manage',
      'setting.manage',
      'audit.read',
    ],
  },
];

const handleDragOver = (e: React.DragEvent) => {
  e.preventDefault();
};

// Admin page listing roles with reorder controls and a per-role permission matrix.
export const RolesListPage: React.FC = () => {
  const { t } = useTranslation();
  const {
    roles,
    isLoadingRoles,
    isFetchingRoles,
    refetchRoles,
    permissions,
    isLoadingPermissions,
    isFetchingPermissions,
    refetchPermissions,
    createRole,
    isCreatingRole,
    updateRole,
    isUpdatingRole,
    deleteRole,
    isDeletingRole,
    updateRolePermissions,
    isUpdatingPermissions,
    reorderRoles,
    isReorderingRoles,
  } = useRoles();

  const {
    selectedRole,
    setSelectedRole,
    assignments,
    setAssignments,
    showModal,
    setShowModal,
    modalMode,
    setModalMode,
    form,
    setForm,
    roleToDelete,
    setRoleToDelete,
    errorMsg,
    setErrorMsg,
    permSearch,
    setPermSearch,
    expandedCategories,
    toggleCategory,
    setAllCategories,
    toastMessage,
    setToastMessage,
  } = useRoleManageStore(
    useShallow((s) => ({
      selectedRole: s.selectedRole,
      setSelectedRole: s.setSelectedRole,
      assignments: s.assignments,
      setAssignments: s.setAssignments,
      showModal: s.showModal,
      setShowModal: s.setShowModal,
      modalMode: s.modalMode,
      setModalMode: s.setModalMode,
      form: s.form,
      setForm: s.setForm,
      roleToDelete: s.roleToDelete,
      setRoleToDelete: s.setRoleToDelete,
      errorMsg: s.errorMsg,
      setErrorMsg: s.setErrorMsg,
      permSearch: s.permSearch,
      setPermSearch: s.setPermSearch,
      expandedCategories: s.expandedCategories,
      toggleCategory: s.toggleCategory,
      setAllCategories: s.setAllCategories,
      toastMessage: s.toastMessage,
      setToastMessage: s.setToastMessage,
    }))
  );

  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);

  useEffect(() => {
    if (roles.length > 0) {
      if (!selectedRole || !roles.some((r) => r.id === selectedRole.id)) {
        setSelectedRole(roles[0]);
      } else {
        const updated = roles.find((r) => r.id === selectedRole.id);
        if (updated) {
          setSelectedRole(updated);
        }
      }
    }
  }, [roles, selectedRole, setSelectedRole]);

  useEffect(() => {
    if (selectedRole?.permissions) {
      setAssignments(
        selectedRole.permissions.map((rp) => ({
          permission_key: rp.permission_key,
          effect: rp.effect || 'allow',
          conditions: rp.conditions || {},
        }))
      );
    } else {
      setAssignments([]);
    }
  }, [selectedRole?.permissions, setAssignments]);

  const hasChanges = useMemo(() => {
    if (!selectedRole) return false;
    const orig = selectedRole.permissions || [];
    if (orig.length !== assignments.length) return true;
    return assignments.some((a) => {
      const found = orig.find((o) => o.permission_key === a.permission_key);
      if (!found) return true;
      if ((found.effect || 'allow') !== a.effect) return true;
      return false;
    });
  }, [selectedRole, assignments]);

  const canModify = Boolean(
    selectedRole && !selectedRole.is_admin && !selectedRole.is_banned
  );

  const allCategoryIds = useMemo(() => PERM_CATEGORIES.map((c) => c.id), []);
  const allExpanded = useMemo(() => {
    return allCategoryIds.every((id) => expandedCategories[id] !== false);
  }, [allCategoryIds, expandedCategories]);

  const handleToggleAll = () => {
    const nextState = !allExpanded;
    setAllCategories(nextState, allCategoryIds);
  };

  const isAssigned = (key: string): boolean => {
    return assignments.some((a) => a.permission_key === key);
  };

  const getAssignment = (key: string): PermissionAssignment | undefined => {
    return assignments.find((a) => a.permission_key === key);
  };

  const handleTogglePermission = (key: string) => {
    setAssignments((prev) => {
      if (prev.some((a) => a.permission_key === key)) {
        return prev.filter((a) => a.permission_key !== key);
      }
      return [...prev, { permission_key: key, effect: 'allow', conditions: {} }];
    });
  };

  const handleSetEffect = (key: string, effect: 'allow' | 'deny') => {
    setAssignments((prev) =>
      prev.map((a) => (a.permission_key === key ? { ...a, effect } : a))
    );
  };

  const handleRefresh = async () => {
    await Promise.all([refetchRoles(), refetchPermissions()]);
    setToastMessage(t('roles.roles_reordered'));
  };

  const handleMoveUp = async (index: number, e: React.MouseEvent) => {
    e.stopPropagation();
    if (index <= 0) return;
    const newRoles = [...roles];
    const temp = newRoles[index - 1];
    newRoles[index - 1] = newRoles[index];
    newRoles[index] = temp;
    try {
      await reorderRoles(newRoles.map((r) => r.id));
      setToastMessage(t('roles.roles_reordered'));
    } catch {
      setToastMessage(t('roles.error_reorder_failed'));
    }
  };

  const handleMoveDown = async (index: number, e: React.MouseEvent) => {
    e.stopPropagation();
    if (index >= roles.length - 1) return;
    const newRoles = [...roles];
    const temp = newRoles[index + 1];
    newRoles[index + 1] = newRoles[index];
    newRoles[index] = temp;
    try {
      await reorderRoles(newRoles.map((r) => r.id));
      setToastMessage(t('roles.roles_reordered'));
    } catch {
      setToastMessage(t('roles.error_reorder_failed'));
    }
  };

  const handleDragStart = (index: number) => {
    setDraggedIndex(index);
  };

  const handleDrop = async (dropIndex: number) => {
    if (draggedIndex === null || draggedIndex === dropIndex) return;
    const newRoles = [...roles];
    const [moved] = newRoles.splice(draggedIndex, 1);
    newRoles.splice(dropIndex, 0, moved);
    setDraggedIndex(null);
    try {
      await reorderRoles(newRoles.map((r) => r.id));
      setToastMessage(t('roles.roles_reordered'));
    } catch {
      setToastMessage(t('roles.error_reorder_failed'));
    }
  };

  const openCreate = () => {
    setErrorMsg('');
    setForm({ name: '', description: '', auto_assign: false });
    setModalMode('create');
    setShowModal(true);
  };

  const openEdit = () => {
    if (!selectedRole) return;
    if (selectedRole.is_admin) {
      setToastMessage(t('roles.cannot_modify_admin'));
      return;
    }
    if (selectedRole.is_banned) {
      setToastMessage(t('roles.cannot_modify_banned'));
      return;
    }
    setErrorMsg('');
    setForm({
      name: selectedRole.name,
      description: selectedRole.description || '',
      auto_assign: selectedRole.auto_assign || false,
    });
    setModalMode('edit');
    setShowModal(true);
  };

  const handleSaveRole = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg('');
    try {
      if (modalMode === 'create') {
        const created = await createRole({
          name: form.name.toUpperCase().trim(),
          description: form.description.trim(),
          auto_assign: form.auto_assign,
        });
        setSelectedRole(created);
        setShowModal(false);
        setToastMessage(t('roles.role_created'));
      } else if (selectedRole) {
        const updated = await updateRole({
          id: selectedRole.id,
          data: {
            name: form.name.toUpperCase().trim(),
            description: form.description.trim(),
            auto_assign: form.auto_assign,
          },
        });
        setSelectedRole(updated);
        setShowModal(false);
        setToastMessage(t('roles.role_updated'));
      }
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        (modalMode === 'create'
          ? t('roles.error_create_failed')
          : t('roles.error_update_failed'));
      setErrorMsg(msg);
    }
  };

  const handleConfirmDelete = async () => {
    if (!roleToDelete) return;
    try {
      await deleteRole(roleToDelete.id);
      if (selectedRole?.id === roleToDelete.id) {
        const remaining = roles.filter((r) => r.id !== roleToDelete.id);
        setSelectedRole(remaining[0] || null);
      }
      setRoleToDelete(null);
      setToastMessage(t('roles.role_deleted'));
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        t('roles.error_delete_failed');
      setToastMessage(msg);
    }
  };

  const handleSavePermissions = async () => {
    if (!selectedRole || !canModify) return;
    try {
      await updateRolePermissions({
        id: selectedRole.id,
        permissions: assignments,
      });
      setToastMessage(t('roles.permissions_saved'));
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        t('roles.error_permissions_failed');
      setToastMessage(msg);
    }
  };

  const getPermissionDescription = (perm: Permission): string => {
    const key = `roles.perm_${perm.key.replace(/\./g, '_')}`;
    const translated = t(key);
    if (translated && translated !== key) {
      return translated;
    }
    return perm.description || perm.key;
  };

  const isSaving = isCreatingRole || isUpdatingRole || isDeletingRole || isUpdatingPermissions;

  return (
    <div className="space-y-6 max-w-7xl mx-auto pb-12">
      <div className="flex min-w-0 flex-col gap-4 border-b border-slate-200 pb-4 dark:border-slate-800 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <h2 className="break-words text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            {t('roles.title')}
          </h2>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            {t('roles.subtitle')}
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={handleRefresh}
            disabled={isFetchingRoles || isFetchingPermissions}
            aria-label={t('roles.refresh')}
            title={t('roles.refresh')}
            className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-700 shadow-xs transition hover:bg-slate-50 disabled:opacity-50 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800"
          >
            <RefreshCw
              className={`h-4 w-4 ${isFetchingRoles || isFetchingPermissions ? 'animate-spin' : ''}`}
            />
          </button>
          <button
            onClick={openCreate}
            className="inline-flex min-h-11 shrink-0 items-center justify-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-md shadow-indigo-600/20 transition hover:bg-indigo-700"
          >
            <Plus className="h-4 w-4" />
            <span>{t('roles.create_button')}</span>
          </button>
        </div>
      </div>

      <div className="flex flex-col lg:flex-row gap-6 items-start">
        <div className="w-full lg:w-88 xl:w-96 shrink-0 space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="text-xs font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('roles.roles_count', { count: roles.length })}
            </h3>
          </div>

          {isLoadingRoles ? (
            <div className="p-8 flex justify-center bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800">
              <LoadingSpinner size="md" />
            </div>
          ) : (
            <div className="space-y-2.5">
              {roles.map((r, index) => {
                const isSelected = selectedRole?.id === r.id;
                return (
                  <div
                    key={r.id}
                    draggable
                    onDragStart={() => handleDragStart(index)}
                    onDragOver={handleDragOver}
                    onDrop={() => handleDrop(index)}
                    onClick={() => setSelectedRole(r)}
                    className={`flex flex-col gap-2 rounded-xl border p-4 transition-all cursor-pointer sm:flex-row sm:items-center sm:justify-between sm:gap-3 ${
                      isSelected
                        ? 'bg-indigo-50/60 dark:bg-indigo-950/30 border-indigo-500/80 dark:border-indigo-500/80 shadow-xs ring-1 ring-indigo-500/50'
                        : 'bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 hover:border-slate-300 dark:hover:border-slate-700'
                    }`}
                  >
                    <div className="min-w-0 flex-1 flex items-start gap-2.5">
                      <GripVertical className="h-4 w-4 shrink-0 text-slate-400 dark:text-slate-600 cursor-grab active:cursor-grabbing mt-0.5" />
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-1.5 flex-wrap">
                          <Shield
                            className={`h-4 w-4 shrink-0 ${
                              isSelected
                                ? 'text-indigo-600 dark:text-indigo-400'
                                : 'text-slate-400 dark:text-slate-500'
                            }`}
                          />
                          <span className="break-words font-bold text-sm text-slate-900 dark:text-slate-100">
                            {r.name}
                          </span>
                          <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400">
                            {t('roles.role_rank', { n: index + 1 })}
                          </span>

                          {r.is_admin && (
                            <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-rose-100 text-rose-700 dark:bg-rose-950/60 dark:text-rose-300">
                              {t('roles.badge_admin')}
                            </span>
                          )}
                          {r.is_system && !r.is_admin && (
                            <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-blue-100 text-blue-700 dark:bg-blue-950/60 dark:text-blue-300">
                              {t('roles.badge_system')}
                            </span>
                          )}
                          {!r.is_system && (
                            <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300">
                              {t('roles.badge_custom')}
                            </span>
                          )}
                          {r.auto_assign && (
                            <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-emerald-100 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300">
                              {t('roles.badge_auto')}
                            </span>
                          )}
                        </div>

                        {r.description && (
                          <p className="text-xs text-slate-500 dark:text-slate-400 truncate mt-1 pl-6">
                            {r.description}
                          </p>
                        )}
                      </div>
                    </div>

                    <div className="flex shrink-0 items-center gap-1 self-end sm:gap-0.5 sm:self-center">
                      <button
                        type="button"
                        disabled={index === 0 || isReorderingRoles}
                        onClick={(e) => handleMoveUp(index, e)}
                        aria-label={t('roles.move_up')}
                        title={t('roles.move_up')}
                        className="inline-flex h-11 w-11 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-100 hover:text-slate-700 disabled:opacity-30 dark:hover:bg-slate-800 dark:hover:text-slate-200 lg:h-8 lg:w-8"
                      >
                        <ChevronUp className="h-5 w-5 lg:h-4 lg:w-4" />
                      </button>
                      <button
                        type="button"
                        disabled={index === roles.length - 1 || isReorderingRoles}
                        onClick={(e) => handleMoveDown(index, e)}
                        aria-label={t('roles.move_down')}
                        title={t('roles.move_down')}
                        className="inline-flex h-11 w-11 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-100 hover:text-slate-700 disabled:opacity-30 dark:hover:bg-slate-800 dark:hover:text-slate-200 lg:h-8 lg:w-8"
                      >
                        <ChevronDown className="h-5 w-5 lg:h-4 lg:w-4" />
                      </button>
                      {!r.is_admin && !r.is_system && (
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            setRoleToDelete(r);
                          }}
                          aria-label={t('roles.delete_role')}
                          title={t('roles.delete_role')}
                          className="inline-flex h-11 w-11 items-center justify-center rounded-lg text-rose-500 transition hover:bg-rose-50 hover:text-rose-700 dark:hover:bg-rose-950/30 lg:h-8 lg:w-8"
                        >
                          <Trash2 className="h-5 w-5 lg:h-4 lg:w-4" />
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <div className="flex-1 min-w-0 w-full space-y-4">
          {selectedRole ? (
            <>
              <div className="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                <div className="min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <h3 className="break-words text-xl font-bold text-slate-900 dark:text-slate-100">
                      {selectedRole.name}
                    </h3>
                    {selectedRole.is_admin && (
                      <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-rose-100 text-rose-700 dark:bg-rose-950/60 dark:text-rose-300">
                        {t('roles.full_admin_access')}
                      </span>
                    )}
                    {selectedRole.is_banned && (
                      <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-amber-100 text-amber-700 dark:bg-amber-950/60 dark:text-amber-300">
                        {t('roles.blocked_account')}
                      </span>
                    )}
                  </div>
                  <p className="break-words text-xs text-slate-500 dark:text-slate-400 mt-1">
                    {selectedRole.description || t('roles.no_desc')}
                  </p>
                </div>

                <div className="flex flex-wrap items-center gap-2 self-start sm:self-center shrink-0">
                  {!selectedRole.is_admin && !selectedRole.is_banned && (
                    <button
                      type="button"
                      onClick={openEdit}
                      className="inline-flex min-h-11 items-center gap-1.5 rounded-xl border border-slate-200 bg-white px-4 py-2 text-xs font-semibold text-slate-700 transition hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800 lg:min-h-0 lg:px-3"
                    >
                      <Pencil className="h-3.5 w-3.5" />
                      <span>{t('roles.edit_role')}</span>
                    </button>
                  )}
                  {canModify && (
                    <button
                      type="button"
                      onClick={handleSavePermissions}
                      disabled={isSaving}
                      className="inline-flex min-h-11 items-center gap-1.5 rounded-xl bg-indigo-600 px-4 py-2 text-xs font-semibold text-white shadow-md shadow-indigo-600/20 transition hover:bg-indigo-700 disabled:opacity-50 lg:min-h-0 lg:px-3.5"
                    >
                      {isUpdatingPermissions ? (
                        <Loader2 className="h-3.5 w-3.5 animate-spin" />
                      ) : (
                        <Save className="h-3.5 w-3.5" />
                      )}
                      <span>{t('roles.save_permissions')}</span>
                    </button>
                  )}
                </div>
              </div>

              {selectedRole.is_admin ? (
                <div className="bg-slate-50 dark:bg-slate-900/60 border border-slate-200 dark:border-slate-800 rounded-2xl p-6 sm:p-10 text-center flex flex-col items-center gap-3">
                  <div className="p-3 rounded-2xl bg-rose-100 text-rose-600 dark:bg-rose-950/50 dark:text-rose-400">
                    <Shield className="h-10 w-10" />
                  </div>
                  <h4 className="font-bold text-base text-slate-900 dark:text-slate-100">
                    {t('roles.role_admin_title')}
                  </h4>
                  <p className="break-words text-xs text-slate-500 dark:text-slate-400 max-w-md leading-relaxed">
                    {t('roles.role_admin_desc')}
                  </p>
                </div>
              ) : selectedRole.is_banned ? (
                <div className="bg-slate-50 dark:bg-slate-900/60 border border-slate-200 dark:border-slate-800 rounded-2xl p-6 sm:p-10 text-center flex flex-col items-center gap-3">
                  <div className="p-3 rounded-2xl bg-amber-100 text-amber-600 dark:bg-amber-950/50 dark:text-amber-400">
                    <ShieldAlert className="h-10 w-10" />
                  </div>
                  <h4 className="font-bold text-base text-slate-900 dark:text-slate-100">
                    {t('roles.role_banned_title')}
                  </h4>
                  <p className="break-words text-xs text-slate-500 dark:text-slate-400 max-w-md leading-relaxed">
                    {t('roles.role_banned_desc')}
                  </p>
                </div>
              ) : (
                <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl overflow-hidden shadow-xs">
                  <div className="p-3 sm:p-4 border-b border-slate-200 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-950/30 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-bold text-xs uppercase tracking-wider text-slate-700 dark:text-slate-300">
                        {t('roles.permissions_count', { count: permissions.length })}
                      </span>
                      <span className="px-2 py-0.5 rounded-full text-[11px] font-mono bg-indigo-100 dark:bg-indigo-950/60 text-indigo-700 dark:text-indigo-300 font-semibold">
                        {t('roles.assigned_count', { count: assignments.length })}
                      </span>
                    </div>

                    <div className="flex items-center gap-2">
                      <div className="relative flex-1 sm:w-60">
                        <Search className="w-3.5 h-3.5 text-slate-400 absolute left-2.5 top-1/2 -translate-y-1/2 pointer-events-none" />
                        <input
                          type="text"
                          value={permSearch}
                          onChange={(e) => setPermSearch(e.target.value)}
                          placeholder={t('roles.search_permissions')}
                          className="h-11 w-full rounded-xl border border-slate-200 bg-white pl-9 pr-10 text-base text-slate-900 focus:outline-hidden focus:ring-1 focus:ring-indigo-500 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-100 lg:h-9 lg:pl-8 lg:pr-7 lg:text-xs"
                        />
                        {permSearch && (
                          <button
                            type="button"
                            onClick={() => setPermSearch('')}
                            aria-label={t('roles.clear_search')}
                            className="absolute right-1 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center text-slate-400 transition hover:text-slate-600 dark:hover:text-slate-200 lg:h-9 lg:w-9"
                          >
                            <X className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>

                      <button
                        type="button"
                        onClick={handleToggleAll}
                        aria-label={
                          allExpanded ? t('roles.collapse_all') : t('roles.expand_all')
                        }
                        className="inline-flex min-h-11 shrink-0 items-center gap-1 rounded-xl border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-300 dark:hover:bg-slate-800 lg:min-h-0 lg:px-2.5"
                      >
                        {allExpanded ? (
                          <>
                            <ChevronsUp className="w-3.5 h-3.5" />
                            <span className="hidden sm:inline">{t('roles.collapse_all')}</span>
                          </>
                        ) : (
                          <>
                            <ChevronsDown className="w-3.5 h-3.5" />
                            <span className="hidden sm:inline">{t('roles.expand_all')}</span>
                          </>
                        )}
                      </button>
                    </div>
                  </div>

                  {isLoadingPermissions ? (
                    <div className="p-8 flex justify-center">
                      <LoadingSpinner size="md" />
                    </div>
                  ) : (
                    <div className="divide-y divide-slate-200 dark:divide-slate-800">
                      {(() => {
                        const query = permSearch.trim().toLowerCase();
                        const allCategoryKeys = PERM_CATEGORIES.flatMap((c) => c.keys);
                        let totalMatches = 0;

                        const rendered = PERM_CATEGORIES.map((category) => {
                          let categoryPerms = permissions.filter((p) =>
                            category.keys.includes(p.key)
                          );
                          const otherPerms = permissions.filter(
                            (p) => !allCategoryKeys.includes(p.key)
                          );
                          if (category.id === 'system' && otherPerms.length > 0) {
                            categoryPerms = [...categoryPerms, ...otherPerms];
                          }
                          if (categoryPerms.length === 0) return null;

                          const totalCategoryPerms = categoryPerms.length;
                          const assignedCount = categoryPerms.filter((p) =>
                            isAssigned(p.key)
                          ).length;

                          if (query) {
                            categoryPerms = categoryPerms.filter((p) => {
                              const desc = getPermissionDescription(p).toLowerCase();
                              const k = p.key.toLowerCase();
                              return desc.includes(query) || k.includes(query);
                            });
                            if (categoryPerms.length === 0) return null;
                          }

                          totalMatches += categoryPerms.length;
                          const isExpanded = query
                            ? true
                            : expandedCategories[category.id] !== false;

                          const IconComponent = category.icon;

                          return (
                            <div key={category.id} className="border-b border-slate-200 dark:border-slate-800 last:border-b-0">
                              <button
                                type="button"
                                onClick={() => toggleCategory(category.id)}
                                className="flex min-h-11 w-full items-center justify-between bg-slate-50/70 px-4 py-3 text-left transition select-none group cursor-pointer hover:bg-slate-100/70 dark:bg-slate-950/40 dark:hover:bg-slate-800/50"
                              >
                                <div className="flex items-center gap-2.5 min-w-0">
                                  <ChevronDown
                                    className={`w-4 h-4 text-slate-400 group-hover:text-slate-600 dark:group-hover:text-slate-300 transition-transform duration-200 shrink-0 ${
                                      isExpanded ? 'rotate-0' : '-rotate-90'
                                    }`}
                                  />
                                  <IconComponent className="w-4 h-4 text-indigo-600 dark:text-indigo-400 shrink-0" />
                                  <span className="font-semibold text-xs uppercase tracking-wide text-slate-700 dark:text-slate-300 group-hover:text-slate-900 dark:group-hover:text-slate-100 truncate">
                                    {t(category.nameKey)}
                                  </span>
                                </div>
                                <div className="flex items-center gap-2 shrink-0 ml-2">
                                  <span
                                    className={`text-[11px] font-mono px-2 py-0.5 rounded-full font-semibold ${
                                      assignedCount > 0
                                        ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-950/80 dark:text-indigo-300'
                                        : 'bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400'
                                    }`}
                                  >
                                    {assignedCount}/{totalCategoryPerms}
                                  </span>
                                </div>
                              </button>

                              {isExpanded && (
                                <div className="divide-y divide-slate-100 dark:divide-slate-800/60 bg-white dark:bg-slate-900/60">
                                  {categoryPerms.map((perm) => {
                                    const assigned = isAssigned(perm.key);
                                    const assignment = getAssignment(perm.key);

                                    return (
                                      <div
                                        key={perm.key}
                                        className="flex flex-col gap-2 p-3.5 transition hover:bg-slate-50/50 sm:flex-row sm:items-center sm:justify-between sm:gap-4 sm:p-4 dark:hover:bg-slate-800/30"
                                      >
                                        <label className="flex items-start gap-3 flex-1 min-w-0 cursor-pointer select-none">
                                          <input
                                            type="checkbox"
                                            checked={assigned}
                                            onChange={() => handleTogglePermission(perm.key)}
                                            className="w-5 h-5 rounded text-indigo-600 border-slate-300 dark:border-slate-700 mt-0.5 shrink-0 lg:h-4 lg:w-4"
                                          />
                                          <div className="min-w-0 flex-1">
                                            <div className="flex items-baseline gap-2 flex-wrap">
                                              <span
                                                className={`text-xs leading-snug break-words ${
                                                  assigned
                                                    ? 'font-bold text-slate-900 dark:text-slate-100'
                                                    : 'font-medium text-slate-700 dark:text-slate-300'
                                                }`}
                                              >
                                                {getPermissionDescription(perm)}
                                              </span>
                                              <span className="font-mono text-[10px] bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400 px-1.5 py-0.5 rounded break-all">
                                                {perm.key}
                                              </span>
                                            </div>
                                          </div>
                                        </label>

                                        {assigned && (
                                          <div className="flex items-center gap-1 self-start rounded-lg border border-slate-200 bg-slate-50 p-0.5 dark:border-slate-800 dark:bg-slate-950 sm:self-auto">
                                            <button
                                              type="button"
                                              onClick={() => handleSetEffect(perm.key, 'allow')}
                                              className={`inline-flex min-h-11 items-center justify-center rounded-md px-4 py-1 text-sm font-medium transition lg:min-h-0 lg:px-2.5 lg:text-xs ${
                                                assignment?.effect === 'allow'
                                                  ? 'bg-emerald-600 text-white font-bold shadow-xs'
                                                  : 'text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100'
                                              }`}
                                            >
                                              {t('roles.effect_allow')}
                                            </button>
                                            <button
                                              type="button"
                                              onClick={() => handleSetEffect(perm.key, 'deny')}
                                              className={`inline-flex min-h-11 items-center justify-center rounded-md px-4 py-1 text-sm font-medium transition lg:min-h-0 lg:px-2.5 lg:text-xs ${
                                                assignment?.effect === 'deny'
                                                  ? 'bg-rose-600 text-white font-bold shadow-xs'
                                                  : 'text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100'
                                              }`}
                                            >
                                              {t('roles.effect_deny')}
                                            </button>
                                          </div>
                                        )}
                                      </div>
                                    );
                                  })}
                                </div>
                              )}
                            </div>
                          );
                        });

                        if (query && totalMatches === 0) {
                          return (
                            <div className="p-8 text-center text-xs text-slate-400 flex flex-col items-center gap-2">
                              <Search className="w-8 h-8 opacity-40" />
                              <p>{t('roles.no_perms_found')}</p>
                            </div>
                          );
                        }

                        return rendered;
                      })()}
                    </div>
                  )}
                </div>
              )}
            </>
          ) : (
            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-8 sm:p-12 text-center flex flex-col items-center gap-2">
              <Shield className="h-10 w-10 text-slate-400" />
              <p className="text-sm font-medium text-slate-500 dark:text-slate-400">
                {t('roles.select_role_prompt')}
              </p>
            </div>
          )}
        </div>
      </div>

      {hasChanges && canModify && (
        <div className="fixed inset-x-4 bottom-4 z-40 bg-white/95 dark:bg-slate-900/95 backdrop-blur-md border border-indigo-500/40 shadow-2xl rounded-2xl p-3 flex items-center justify-between gap-3 animate-in slide-in-from-bottom duration-200 lg:hidden">
          <div className="flex items-center gap-2 min-w-0">
            <span className="relative flex h-2.5 w-2.5 shrink-0">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-indigo-500 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-indigo-600"></span>
            </span>
            <span className="text-xs font-semibold truncate text-slate-900 dark:text-slate-100">
              {t('roles.unsaved_changes')}
            </span>
          </div>
          <button
            type="button"
            onClick={handleSavePermissions}
            disabled={isSaving}
            className="inline-flex min-h-11 shrink-0 items-center gap-1.5 rounded-xl bg-indigo-600 px-4 py-1.5 text-xs font-semibold text-white shadow-md shadow-indigo-600/20 transition hover:bg-indigo-700 disabled:opacity-50"
          >
            {isUpdatingPermissions ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Save className="h-3.5 w-3.5" />
            )}
            <span>{t('roles.save_permissions')}</span>
          </button>
        </div>
      )}

      <Modal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title={
          modalMode === 'create'
            ? t('roles.modal_create_title')
            : t('roles.modal_edit_title')
        }
      >
        <form onSubmit={handleSaveRole} className="space-y-4">
          {errorMsg && (
            <div className="p-3 rounded-xl bg-rose-50 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300 text-xs flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{errorMsg}</span>
            </div>
          )}

          <div>
            <label className="block text-xs font-semibold mb-1 text-slate-700 dark:text-slate-300">
              {t('roles.role_name')}
            </label>
            <input
              type="text"
              required
              disabled={modalMode === 'edit' && selectedRole?.is_system}
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value.toUpperCase() })}
              placeholder={t('roles.name_placeholder')}
              className="min-h-11 w-full px-3 py-2 rounded-xl border border-slate-200 dark:border-slate-800 text-base bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 uppercase font-mono disabled:opacity-60 disabled:bg-slate-100 dark:disabled:bg-slate-900 lg:min-h-9 lg:text-xs"
            />
            {modalMode === 'edit' && selectedRole?.is_system && (
              <p className="text-[11px] text-slate-400 mt-1">
                {t('roles.system_role_immutable_note')}
              </p>
            )}
          </div>

          <div>
            <label className="block text-xs font-semibold mb-1 text-slate-700 dark:text-slate-300">
              {t('roles.role_desc')}
            </label>
            <textarea
              rows={3}
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              placeholder={t('roles.desc_placeholder')}
              className="w-full px-3 py-2 rounded-xl border border-slate-200 dark:border-slate-800 text-base bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 lg:text-xs"
            />
          </div>

          <div className="flex items-start gap-2.5 pt-1">
            <input
              type="checkbox"
              id="role_auto_assign"
              checked={form.auto_assign}
              onChange={(e) => setForm({ ...form, auto_assign: e.target.checked })}
              className="w-5 h-5 rounded text-indigo-600 border-slate-300 dark:border-slate-700 mt-0.5 lg:h-4 lg:w-4"
            />
            <label htmlFor="role_auto_assign" className="text-xs cursor-pointer select-none">
              <span className="font-semibold text-slate-800 dark:text-slate-200 block">
                {t('roles.auto_assign_label')}
              </span>
              <span className="text-[11px] text-slate-400 block mt-0.5">
                {t('roles.auto_assign_desc')}
              </span>
            </label>
          </div>

          <div className="flex justify-end gap-2 pt-4 border-t border-slate-100 dark:border-slate-800">
            <button
              type="button"
              onClick={() => setShowModal(false)}
              className="min-h-11 flex-1 rounded-xl px-4 py-2 text-xs font-medium text-slate-600 transition hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800 sm:flex-none lg:min-h-0"
            >
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              disabled={isSaving}
              className="min-h-11 flex-1 rounded-xl bg-indigo-600 px-4 py-2 text-xs font-semibold text-white shadow-md shadow-indigo-600/20 transition hover:bg-indigo-700 disabled:opacity-50 sm:flex-none lg:min-h-0"
            >
              {isSaving ? (
                <Loader2 className="w-4 h-4 animate-spin mx-auto" />
              ) : modalMode === 'create' ? (
                t('roles.create_action')
              ) : (
                t('roles.save_action')
              )}
            </button>
          </div>
        </form>
      </Modal>

      {roleToDelete && (
        <Modal
          isOpen={Boolean(roleToDelete)}
          onClose={() => setRoleToDelete(null)}
          title={t('roles.delete_role_title')}
        >
          <div className="space-y-4">
            <div className="flex items-start gap-3">
              <div className="p-2 rounded-xl bg-rose-100 text-rose-600 dark:bg-rose-950/60 dark:text-rose-400 shrink-0">
                <AlertCircle className="w-5 h-5" />
              </div>
              <p className="text-xs text-slate-600 dark:text-slate-300 leading-relaxed pt-0.5 break-words">
                {t('roles.delete_role_confirm', { name: roleToDelete.name })}
              </p>
            </div>

            <div className="flex justify-end gap-2 pt-4 border-t border-slate-100 dark:border-slate-800">
              <button
                type="button"
                onClick={() => setRoleToDelete(null)}
                className="min-h-11 flex-1 rounded-xl px-4 py-2 text-xs font-medium text-slate-600 transition hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800 sm:flex-none lg:min-h-0"
              >
                {t('common.cancel')}
              </button>
              <button
                type="button"
                onClick={handleConfirmDelete}
                disabled={isDeletingRole}
                className="min-h-11 flex-1 rounded-xl bg-rose-600 px-4 py-2 text-xs font-semibold text-white shadow-md shadow-rose-600/20 transition hover:bg-rose-700 disabled:opacity-50 sm:flex-none lg:min-h-0"
              >
                {isDeletingRole ? (
                  <Loader2 className="w-4 h-4 animate-spin mx-auto" />
                ) : (
                  t('common.delete')
                )}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {toastMessage && (
        <div className="fixed inset-x-4 bottom-24 z-50 flex items-center gap-2 rounded-xl bg-slate-900 text-white dark:bg-white dark:text-slate-900 px-4 py-3 shadow-lg text-xs font-medium border border-slate-700 dark:border-slate-200 animate-in fade-in slide-in-from-bottom-2 duration-200 sm:inset-x-auto sm:right-5 sm:max-w-sm lg:bottom-5">
          <CheckCircle2 className="w-4 h-4 text-emerald-400 dark:text-emerald-600 shrink-0" />
          <span className="break-words">{toastMessage}</span>
        </div>
      )}
    </div>
  );
};
