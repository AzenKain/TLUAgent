import React from 'react';
import { useAuth } from '@/hooks/useAuth';
import { Navigate } from 'react-router-dom';
import { ShieldCheck, Mail, Lock, User as UserIcon, Hash, AlertCircle, ArrowRight } from 'lucide-react';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { PasswordStrengthMeter } from '@/components/common/PasswordStrengthMeter';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/react/shallow';
import { useAuthFormStore } from '@/stores/authFormStore';
import { getApiErrorMessage } from '@/utils/apiError';

const CURRENT_YEAR = new Date().getFullYear();

export const SetupPage: React.FC = () => {
  const { setup, isSettingUp, setupStatus, isSetupLoading } = useAuth();
  const { t } = useTranslation();

  const {
    setupEmail: email,
    setupPassword: password,
    setupConfirmPassword: confirmPassword,
    setupFullName: fullName,
    setupStudentCode: studentCode,
    setupError: errorMsg,
    setSetupEmail: setEmail,
    setSetupPassword: setPassword,
    setSetupConfirmPassword: setConfirmPassword,
    setSetupFullName: setFullName,
    setSetupStudentCode: setStudentCode,
    setSetupError: setErrorMsg,
  } = useAuthFormStore(
    useShallow((s) => ({
      setupEmail: s.setupEmail,
      setupPassword: s.setupPassword,
      setupConfirmPassword: s.setupConfirmPassword,
      setupFullName: s.setupFullName,
      setupStudentCode: s.setupStudentCode,
      setupError: s.setupError,
      setSetupEmail: s.setSetupEmail,
      setSetupPassword: s.setSetupPassword,
      setSetupConfirmPassword: s.setSetupConfirmPassword,
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

    if (password.length < 10) {
      setErrorMsg(t('auth.password_min_length_error'));
      return;
    }

    if (password !== confirmPassword) {
      setErrorMsg(t('auth.password_mismatch'));
      return;
    }

    try {
      await setup({
        email,
        password,
        full_name: fullName,
        student_code: studentCode,
      });
    } catch (err: unknown) {
      const msg = getApiErrorMessage(err, t('auth.setup_error_default'));
      setErrorMsg(msg);
    }
  };

  return (
    <div className="min-h-dvh flex flex-col items-center justify-center px-4 py-8 sm:px-6 bg-[#F8FAFC] dark:bg-[#0B1021] relative">
      <div className="fixed top-4 right-4 z-10 sm:top-6 sm:right-6">
        <LanguageSwitcher />
      </div>

      <div className="w-full max-w-lg bg-white dark:bg-[#111A35] rounded-2xl shadow-xl border border-slate-200/80 dark:border-blue-900/30 p-6 sm:p-8 relative overflow-hidden">
        <div className="absolute top-0 left-0 right-0 h-1.5 bg-linear-to-r from-tlu-navy via-tlu-blue to-tlu-red" />

        <div className="text-center mb-8">
          <div className="w-20 h-20 mx-auto rounded-2xl bg-white p-2 border border-slate-200/80 shadow-md mb-3 flex items-center justify-center">
            <img src="/logo.png" alt="Thang Long University Logo" className="w-full h-full object-contain" />
          </div>
          <div className="inline-flex items-center gap-1.5 text-xs font-semibold text-tlu-blue dark:text-tlu-cyan bg-blue-50 dark:bg-blue-950/60 px-3 py-1 rounded-full uppercase tracking-wider mb-2 border border-blue-200/60 dark:border-blue-800/60">
            <ShieldCheck className="w-3.5 h-3.5" />
            <span>{t('auth.setup_badge')}</span>
          </div>
          <h2 className="text-2xl font-bold text-slate-900 dark:text-slate-100 tracking-tight">
            {t('auth.setup_title')}
          </h2>
          <p className="text-xs sm:text-sm text-slate-500 dark:text-slate-400 mt-1.5">
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
                className="w-full pl-11 pr-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 focus:outline-hidden focus:ring-2 focus:ring-tlu-blue text-base sm:text-sm transition"
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
                className="w-full pl-11 pr-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 focus:outline-hidden focus:ring-2 focus:ring-tlu-blue text-base sm:text-sm transition"
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
                className="w-full pl-11 pr-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 focus:outline-hidden focus:ring-2 focus:ring-tlu-blue text-base sm:text-sm transition"
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
                minLength={10}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                className="w-full pl-11 pr-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 focus:outline-hidden focus:ring-2 focus:ring-tlu-blue text-base sm:text-sm transition"
              />
            </div>
            <PasswordStrengthMeter password={password} />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-1.5">
              {t('auth.confirm_password')}
            </label>
            <div className="relative">
              <Lock className="w-5 h-5 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="password"
                required
                minLength={10}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder={t('auth.confirm_password_placeholder')}
                className="w-full pl-11 pr-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 text-slate-900 dark:text-slate-100 focus:outline-hidden focus:ring-2 focus:ring-tlu-blue text-base sm:text-sm transition"
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={isSettingUp}
            className="w-full min-h-11 py-3 px-4 rounded-xl bg-tlu-navy hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-[#1541B8] text-white font-medium text-sm flex items-center justify-center gap-2 shadow-lg shadow-blue-900/20 transition disabled:opacity-50 mt-4 cursor-pointer"
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

      <div className="mt-8 text-center text-xs text-slate-400 dark:text-slate-600">
        <p>© {CURRENT_YEAR} Thang Long University. All rights reserved.</p>
      </div>
    </div>
  );
};
