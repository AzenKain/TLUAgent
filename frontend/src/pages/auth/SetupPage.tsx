import React from 'react';
import { useAuth } from '@/hooks/useAuth';
import { Navigate } from 'react-router-dom';
import { ShieldCheck, Mail, Lock, User as UserIcon, Hash, AlertCircle, ArrowRight } from 'lucide-react';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { useTranslation } from 'react-i18next';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';
import { useShallow } from 'zustand/react/shallow';
import { useAuthFormStore } from '@/stores/authFormStore';

export const SetupPage: React.FC = () => {
  const { setup, isSettingUp, setupStatus, isSetupLoading } = useAuth();
  const { t } = useTranslation();

  const {
    setupEmail: email,
    setupPassword: password,
    setupFullName: fullName,
    setupStudentCode: studentCode,
    setupError: errorMsg,
    setSetupEmail: setEmail,
    setSetupPassword: setPassword,
    setSetupFullName: setFullName,
    setSetupStudentCode: setStudentCode,
    setSetupError: setErrorMsg,
  } = useAuthFormStore(
    useShallow((s) => ({
      setupEmail: s.setupEmail,
      setupPassword: s.setupPassword,
      setupFullName: s.setupFullName,
      setupStudentCode: s.setupStudentCode,
      setupError: s.setupError,
      setSetupEmail: s.setSetupEmail,
      setSetupPassword: s.setSetupPassword,
      setSetupFullName: s.setSetupFullName,
      setSetupStudentCode: s.setSetupStudentCode,
      setSetupError: s.setSetupError,
    }))
  );

  if (isSetupLoading) {
    return (
      <div className="min-h-dvh flex items-center justify-center bg-slate-50 dark:bg-slate-950">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  if (!setupStatus?.required) {
    return <Navigate to="/login" replace />;
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg('');

    try {
      await setup({
        email,
        password,
        full_name: fullName,
        student_code: studentCode,
      });
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        t('auth.setup_error_default');
      setErrorMsg(msg);
    }
  };

  return (
    <div className="min-h-dvh flex items-center justify-center px-4 py-6 sm:px-6 bg-slate-50 dark:bg-slate-950 relative">
      <div className="fixed top-4 right-4 z-10 sm:top-6 sm:right-6">
        <LanguageSwitcher />
      </div>

      <div className="w-full max-w-lg bg-white dark:bg-slate-900 rounded-2xl shadow-xl border border-slate-200 dark:border-slate-800 p-5 sm:p-8">
        <div className="text-center mb-8">
          <div className="w-16 h-16 mx-auto rounded-2xl bg-amber-500 text-white flex items-center justify-center shadow-lg shadow-amber-500/30 mb-4">
            <ShieldCheck className="w-9 h-9" />
          </div>
          <span className="text-xs font-bold text-amber-500 bg-amber-500/10 px-2.5 py-1 rounded-full uppercase tracking-wider">
            {t('auth.setup_badge')}
          </span>
          <h2 className="text-2xl font-bold text-slate-900 dark:text-slate-100 tracking-tight mt-3">
            {t('auth.setup_title')}
          </h2>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            {t('auth.setup_sub')}
          </p>
        </div>

        {errorMsg && (
          <div className="mb-6 p-4 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900/60 flex items-start gap-3 text-rose-700 dark:text-rose-300 text-sm">
            <AlertCircle className="w-5 h-5 shrink-0 mt-0.5" />
            <span>{errorMsg}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-1.5">
              {t('auth.full_name')}
            </label>
            <div className="relative">
              <UserIcon className="w-5 h-5 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                required
                value={fullName}
                onChange={(e) => setFullName(e.target.value)}
                placeholder={t('auth.full_name_placeholder')}
                className="w-full pl-11 pr-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 focus:outline-hidden focus:ring-2 focus:ring-amber-500 text-base sm:text-sm transition"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-1.5">
              {t('auth.student_code')}
            </label>
            <div className="relative">
              <Hash className="w-5 h-5 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                value={studentCode}
                onChange={(e) => setStudentCode(e.target.value)}
                placeholder={t('auth.setup_code_placeholder')}
                className="w-full pl-11 pr-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 focus:outline-hidden focus:ring-2 focus:ring-amber-500 text-base sm:text-sm transition"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-1.5">
              {t('auth.email')}
            </label>
            <div className="relative">
              <Mail className="w-5 h-5 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder={t('auth.setup_email_placeholder')}
                className="w-full pl-11 pr-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 focus:outline-hidden focus:ring-2 focus:ring-amber-500 text-base sm:text-sm transition"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-1.5">
              {t('auth.password')}
            </label>
            <div className="relative">
              <Lock className="w-5 h-5 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="password"
                required
                minLength={6}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                className="w-full pl-11 pr-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 focus:outline-hidden focus:ring-2 focus:ring-amber-500 text-base sm:text-sm transition"
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={isSettingUp}
            className="w-full min-h-11 py-3 px-4 rounded-xl bg-amber-500 hover:bg-amber-600 text-white font-medium text-sm flex items-center justify-center gap-2 shadow-lg shadow-amber-500/25 transition disabled:opacity-50 mt-4"
          >
            {isSettingUp ? (
              <>
                <LoadingSpinner size="sm" />
                <span>{t('auth.setting_up')}</span>
              </>
            ) : (
              <>
                <span>{t('auth.setup_button')}</span>
                <ArrowRight className="w-4 h-4" />
              </>
            )}
          </button>
        </form>
      </div>
    </div>
  );
};
