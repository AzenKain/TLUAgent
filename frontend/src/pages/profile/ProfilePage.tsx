import React, { useEffect, useState } from 'react';
import { useAuth } from '@/hooks/useAuth';
import { userService } from '@/services/userService';
import { useMutation } from '@tanstack/react-query';
import { User, Lock, Save, KeyRound, CheckCircle2, AlertTriangle, Trash2 } from 'lucide-react';
import { Badge } from '@/components/common/Badge';
import { Modal } from '@/components/common/Modal';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/react/shallow';
import { useProfileStore } from '@/stores/profileStore';

export const ProfilePage: React.FC = () => {
  const { user, refreshMe } = useAuth();
  const { t } = useTranslation();
  const [isClearModalOpen, setIsClearModalOpen] = useState(false);

  const {
    fullName,
    studentCode,
    oldPassword,
    newPassword,
    confirmPassword,
    profileMsg,
    passwordMsg,
    setFullName,
    setStudentCode,
    setOldPassword,
    setNewPassword,
    setConfirmPassword,
    setProfileMsg,
    setPasswordMsg,
    resetPasswordForm,
  } = useProfileStore(
    useShallow((s) => ({
      fullName: s.fullName,
      studentCode: s.studentCode,
      oldPassword: s.oldPassword,
      newPassword: s.newPassword,
      confirmPassword: s.confirmPassword,
      profileMsg: s.profileMsg,
      passwordMsg: s.passwordMsg,
      setFullName: s.setFullName,
      setStudentCode: s.setStudentCode,
      setOldPassword: s.setOldPassword,
      setNewPassword: s.setNewPassword,
      setConfirmPassword: s.setConfirmPassword,
      setProfileMsg: s.setProfileMsg,
      setPasswordMsg: s.setPasswordMsg,
      resetPasswordForm: s.resetPasswordForm,
    }))
  );

  useEffect(() => {
    if (user) {
      if (user.full_name) setFullName(user.full_name);
      if (user.student_code) setStudentCode(user.student_code);
    }
  }, [user, setFullName, setStudentCode]);

  const updateProfileMutation = useMutation({
    mutationFn: () =>
      userService.updateCurrentProfile({
        full_name: fullName,
        student_code: studentCode,
      }),
    onSuccess: () => {
      refreshMe();
      setProfileMsg({ text: t('profile.toast_profile_success'), type: 'success' });
      setTimeout(() => setProfileMsg({ text: '', type: '' }), 3000);
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        t('profile.error_profile_failed');
      setProfileMsg({
        text: msg,
        type: 'error',
      });
    },
  });

  const changePasswordMutation = useMutation({
    mutationFn: () =>
      userService.changeCurrentPassword({
        old_password: oldPassword,
        new_password: newPassword,
      }),
    onSuccess: () => {
      resetPasswordForm();
      setPasswordMsg({ text: t('profile.toast_password_success'), type: 'success' });
      setTimeout(() => setPasswordMsg({ text: '', type: '' }), 3000);
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        t('profile.error_password_failed');
      setPasswordMsg({
        text: msg,
        type: 'error',
      });
    },
  });

  const clearDataMutation = useMutation({
    mutationFn: async () => {
      await userService.clearCurrentProfileAndMemories();
      await userService.updateCurrentProfile({
        full_name: fullName,
        student_code: '',
      });
    },
    onSuccess: () => {
      setStudentCode('');
      refreshMe();
      setProfileMsg({ text: t('profile.toast_clear_success'), type: 'success' });
      setTimeout(() => setProfileMsg({ text: '', type: '' }), 4000);
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        t('profile.error_clear_failed');
      setProfileMsg({
        text: msg,
        type: 'error',
      });
    },
  });

  const handleProfileSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    updateProfileMutation.mutate();
  };

  const handlePasswordSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      setPasswordMsg({ text: t('profile.password_mismatch'), type: 'error' });
      return;
    }
    changePasswordMutation.mutate();
  };

  return (
    <div className="min-w-0 w-full space-y-8 max-w-4xl mx-auto">
      <div>
        <h2 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
          {t('profile.title')}
        </h2>
        <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
          {t('profile.subtitle')}
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Profile Details Form */}
        <div className="p-4 sm:p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs">
          <div className="flex items-center gap-3 mb-6 pb-4 border-b border-slate-100 dark:border-slate-800">
            <div className="p-2.5 rounded-xl bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400">
              <User className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">
                {t('profile.card_personal_title')}
              </h3>
              <p className="text-xs text-slate-400">{t('profile.card_personal_sub')}</p>
            </div>
          </div>

          {profileMsg.text && (
            <div
              className={`p-3 rounded-xl mb-4 text-xs flex items-center gap-2 ${
                profileMsg.type === 'success'
                  ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300'
                  : 'bg-rose-50 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300'
              }`}
            >
              {profileMsg.type === 'success' ? (
                <CheckCircle2 className="w-4 h-4 shrink-0" />
              ) : (
                <AlertTriangle className="w-4 h-4 shrink-0" />
              )}
              <span>{profileMsg.text}</span>
            </div>
          )}

          <form onSubmit={handleProfileSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('profile.email')}
              </label>
              <input
                type="text"
                disabled
                value={user?.email || ''}
                className="w-full px-3 py-2.5 sm:py-2 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-100 dark:bg-slate-800/50 text-slate-500 text-base sm:text-xs cursor-not-allowed"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('auth.full_name')}
              </label>
              <input
                type="text"
                required
                value={fullName}
                onChange={(e) => setFullName(e.target.value)}
                className="w-full px-3 py-2.5 sm:py-2 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 text-base sm:text-xs focus:ring-2 focus:ring-indigo-500 focus:outline-hidden"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('auth.student_code')}
              </label>
              <input
                type="text"
                value={studentCode}
                onChange={(e) => setStudentCode(e.target.value)}
                className="w-full px-3 py-2.5 sm:py-2 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 text-base sm:text-xs focus:ring-2 focus:ring-indigo-500 focus:outline-hidden"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-2">
                {t('profile.current_roles')}
              </label>
              <div className="flex flex-wrap gap-1.5">
                {user?.roles?.map((r) => (
                  <Badge key={r.id} variant="primary" size="sm">
                    {r.name}
                  </Badge>
                ))}
              </div>
            </div>

            <button
              type="submit"
              disabled={updateProfileMutation.isPending}
              className="w-full min-h-11 py-2.5 px-4 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white font-medium text-xs flex items-center justify-center gap-2 shadow-sm transition disabled:opacity-50 mt-4"
            >
              <Save className="w-4 h-4" />
              <span>{updateProfileMutation.isPending ? t('common.saving') : t('common.save')}</span>
            </button>
          </form>
        </div>

        {/* Change Password Form */}
        <div className="p-4 sm:p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs">
          <div className="flex items-center gap-3 mb-6 pb-4 border-b border-slate-100 dark:border-slate-800">
            <div className="p-2.5 rounded-xl bg-violet-50 dark:bg-violet-950/60 text-violet-600 dark:text-violet-400">
              <Lock className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">
                {t('profile.card_password_title')}
              </h3>
              <p className="text-xs text-slate-400">{t('profile.card_password_sub')}</p>
            </div>
          </div>

          {passwordMsg.text && (
            <div
              className={`p-3 rounded-xl mb-4 text-xs flex items-center gap-2 ${
                passwordMsg.type === 'success'
                  ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300'
                  : 'bg-rose-50 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300'
              }`}
            >
              {passwordMsg.type === 'success' ? (
                <CheckCircle2 className="w-4 h-4 shrink-0" />
              ) : (
                <AlertTriangle className="w-4 h-4 shrink-0" />
              )}
              <span>{passwordMsg.text}</span>
            </div>
          )}

          <form onSubmit={handlePasswordSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('profile.old_password')}
              </label>
              <input
                type="password"
                required
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                placeholder="••••••••"
                className="w-full px-3 py-2.5 sm:py-2 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 text-base sm:text-xs focus:ring-2 focus:ring-violet-500 focus:outline-hidden"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('profile.new_password')}
              </label>
              <input
                type="password"
                required
                minLength={6}
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="••••••••"
                className="w-full px-3 py-2.5 sm:py-2 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 text-base sm:text-xs focus:ring-2 focus:ring-violet-500 focus:outline-hidden"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                {t('profile.confirm_new_password')}
              </label>
              <input
                type="password"
                required
                minLength={6}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="••••••••"
                className="w-full px-3 py-2.5 sm:py-2 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 text-base sm:text-xs focus:ring-2 focus:ring-violet-500 focus:outline-hidden"
              />
            </div>

            <button
              type="submit"
              disabled={changePasswordMutation.isPending}
              className="w-full min-h-11 py-2.5 px-4 rounded-xl bg-violet-600 hover:bg-violet-700 text-white font-medium text-xs flex items-center justify-center gap-2 shadow-sm transition disabled:opacity-50 mt-4"
            >
              <KeyRound className="w-4 h-4" />
              <span>
                {changePasswordMutation.isPending
                  ? t('common.saving')
                  : t('profile.change_password_action')}
              </span>
            </button>
          </form>
        </div>
      </div>

      {/* AI Memory & Privacy Management Card */}
      <div className="p-4 sm:p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs">
        <div className="flex items-center gap-3 mb-4 pb-4 border-b border-slate-100 dark:border-slate-800">
          <div className="p-2.5 rounded-xl bg-rose-50 dark:bg-rose-950/60 text-rose-600 dark:text-rose-400">
            <Trash2 className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">
              {t('profile.card_privacy_title')}
            </h3>
            <p className="text-xs text-slate-400">{t('profile.card_privacy_sub')}</p>
          </div>
        </div>

        <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed mb-6">
          {t('profile.privacy_desc')}
        </p>

        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 p-4 rounded-xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/80 dark:border-slate-800">
          <div>
            <div className="text-xs font-semibold text-slate-800 dark:text-slate-200">
              {user?.student_code ? `${t('auth.student_code')}: ${user.student_code}` : t('chat.guest_mode')}
            </div>
            <div className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5">
              {t('profile.confirm_clear_desc')}
            </div>
          </div>
          <button
            type="button"
            onClick={() => setIsClearModalOpen(true)}
            disabled={clearDataMutation.isPending}
            className="w-full sm:w-auto shrink-0 px-4 py-2.5 rounded-xl bg-rose-600 hover:bg-rose-700 text-white font-medium text-xs flex items-center justify-center gap-2 shadow-sm transition disabled:opacity-50"
          >
            <Trash2 className="w-4 h-4" />
            <span>{clearDataMutation.isPending ? t('profile.clearing_data') : t('profile.btn_clear_data')}</span>
          </button>
        </div>
      </div>

      {/* Confirmation Modal */}
      <Modal
        isOpen={isClearModalOpen}
        onClose={() => setIsClearModalOpen(false)}
        title={t('profile.confirm_clear_title')}
      >
        <div className="space-y-4">
          <p className="text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
            {t('profile.confirm_clear_desc')}
          </p>
          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={() => setIsClearModalOpen(false)}
              className="px-4 py-2 rounded-xl text-xs font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 transition"
            >
              {t('common.cancel')}
            </button>
            <button
              type="button"
              onClick={() => {
                setIsClearModalOpen(false);
                clearDataMutation.mutate();
              }}
              className="px-4 py-2 rounded-xl text-xs font-medium bg-rose-600 hover:bg-rose-700 text-white transition shadow-xs"
            >
              {t('common.confirm')}
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

