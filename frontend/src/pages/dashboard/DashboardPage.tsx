import React from 'react';
import { useAuth } from '@/hooks/useAuth';
import { usePermissions } from '@/hooks/usePermissions';
import { Link } from 'react-router-dom';
import {
  MessageSquare,
  Users,
  ShieldCheck,
  Sparkles,
  ArrowUpRight,
  BookOpen,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

// Admin dashboard landing page with a welcome banner and quick action cards.
export const DashboardPage: React.FC = () => {
  const { user } = useAuth();
  const { canAny } = usePermissions();
  const { t } = useTranslation();

  const canAccessAdmin = canAny(['user.manage', 'role.manage', 'setting.manage', 'admin.access']);

  return (
    <div className="space-y-6 max-w-7xl mx-auto pb-10">
      <div className="relative overflow-hidden rounded-2xl bg-linear-to-r from-tlu-navy via-tlu-navy-light to-tlu-blue text-white p-6 sm:p-8 shadow-xl shadow-blue-950/20 border border-blue-900/40">
        <div className="relative z-10 max-w-2xl">
          <div className="flex items-center gap-2 mb-4">
            <div className="w-8 h-8 rounded-lg bg-white p-1 border border-white/20 shadow-xs flex items-center justify-center shrink-0">
              <img src="/logo.png" alt="TLU Emblem" className="w-full h-full object-contain" />
            </div>
            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-white/15 border border-white/20">
              <Sparkles className="w-3.5 h-3.5 text-tlu-cyan shrink-0" />
              <span>{t('dashboard.welcome_badge')}</span>
            </span>
          </div>

          <h2 className="text-2xl sm:text-3xl font-extrabold tracking-tight wrap-break-word">
            {t('dashboard.welcome_title', { name: user?.full_name || '' })}
          </h2>
          <p className="text-blue-100 text-sm mt-2 leading-relaxed wrap-break-word">
            {t('dashboard.welcome_desc')}
          </p>

          <div className="mt-6 flex flex-col sm:flex-row sm:flex-wrap gap-3">
            <Link
              to="/"
              className="inline-flex items-center justify-center gap-2 px-5 py-2.5 min-h-11 w-full sm:w-auto rounded-xl bg-white text-tlu-navy font-semibold text-sm shadow-md hover:bg-slate-100 transition"
            >
              <MessageSquare className="w-4 h-4 shrink-0 text-tlu-blue" />
              <span>{t('dashboard.action_chat')}</span>
            </Link>
            {canAccessAdmin && (
              <Link
                to="/admin/users"
                className="inline-flex items-center justify-center gap-2 px-5 py-2.5 min-h-11 w-full sm:w-auto rounded-xl bg-white/10 hover:bg-white/20 text-white font-semibold text-sm border border-white/20 transition"
              >
                <Users className="w-4 h-4 shrink-0" />
                <span>{t('dashboard.action_users')}</span>
              </Link>
            )}
          </div>
        </div>

        <div className="absolute right-0 top-0 bottom-0 w-1/3 bg-linear-to-l from-white/10 to-transparent pointer-events-none" />
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
        <div className="p-6 rounded-2xl bg-white dark:bg-[#111A35] border border-slate-200 dark:border-blue-900/30 shadow-2xs flex flex-col justify-between min-w-0">
          <div>
            <div className="w-12 h-12 rounded-xl bg-blue-50 dark:bg-blue-950/60 text-tlu-blue dark:text-tlu-cyan flex items-center justify-center mb-4 shrink-0">
              <MessageSquare className="w-6 h-6" />
            </div>
            <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">
              {t('dashboard.card_advisory_title')}
            </h3>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1.5 leading-relaxed">
              {t('dashboard.card_advisory_desc')}
            </p>
          </div>
          <Link
            to="/"
            className="mt-4 pt-3 min-h-11 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between gap-2 text-xs font-semibold text-tlu-blue dark:text-tlu-cyan hover:underline"
          >
            <span className="min-w-0">{t('dashboard.card_advisory_link')}</span>
            <ArrowUpRight className="w-4 h-4 shrink-0" />
          </Link>
        </div>

        <div className="p-6 rounded-2xl bg-white dark:bg-[#111A35] border border-slate-200 dark:border-blue-900/30 shadow-2xs flex flex-col justify-between min-w-0">
          <div>
            <div className="w-12 h-12 rounded-xl bg-rose-50 dark:bg-rose-950/60 text-tlu-red dark:text-rose-400 flex items-center justify-center mb-4 shrink-0">
              <BookOpen className="w-6 h-6" />
            </div>
            <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">
              {t('dashboard.card_curriculum_title')}
            </h3>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1.5 leading-relaxed">
              {t('dashboard.card_curriculum_desc')}
            </p>
          </div>
          <Link
            to="/roadmap"
            className="mt-4 pt-3 min-h-11 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between gap-2 text-xs font-semibold text-tlu-red dark:text-rose-400 hover:underline"
          >
            <span className="min-w-0">{t('dashboard.card_curriculum_link')}</span>
            <ArrowUpRight className="w-4 h-4 shrink-0" />
          </Link>
        </div>

        {canAccessAdmin ? (
          <div className="sm:col-span-2 lg:col-span-1 p-6 rounded-2xl bg-white dark:bg-[#111A35] border border-slate-200 dark:border-blue-900/30 shadow-2xs flex flex-col justify-between min-w-0">
            <div>
              <div className="w-12 h-12 rounded-xl bg-blue-50 dark:bg-blue-950/60 text-tlu-navy dark:text-tlu-cyan flex items-center justify-center mb-4 shrink-0">
                <ShieldCheck className="w-6 h-6" />
              </div>
              <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">
                {t('dashboard.card_admin_title')}
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1.5 leading-relaxed">
                {t('dashboard.card_admin_desc')}
              </p>
            </div>
            <Link
              to="/admin/users"
              className="mt-4 pt-3 min-h-11 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between gap-2 text-xs font-semibold text-tlu-blue dark:text-tlu-cyan hover:underline"
            >
              <span className="min-w-0">{t('dashboard.card_admin_link')}</span>
              <ArrowUpRight className="w-4 h-4 shrink-0" />
            </Link>
          </div>
        ) : (
          <div className="sm:col-span-2 lg:col-span-1 p-6 rounded-2xl bg-white dark:bg-[#111A35] border border-slate-200 dark:border-blue-900/30 shadow-2xs flex flex-col justify-between min-w-0">
            <div>
              <div className="w-12 h-12 rounded-xl bg-emerald-50 dark:bg-emerald-950/60 text-emerald-600 dark:text-emerald-400 flex items-center justify-center mb-4 shrink-0">
                <Sparkles className="w-6 h-6" />
              </div>
              <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">
                {t('dashboard.card_profile_title')}
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1.5 leading-relaxed">
                {t('dashboard.card_profile_desc')}
              </p>
            </div>
            <Link
              to="/profile"
              className="mt-4 pt-3 min-h-11 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between gap-2 text-xs font-semibold text-emerald-600 dark:text-emerald-400 hover:underline"
            >
              <span className="min-w-0">{t('dashboard.card_profile_link')}</span>
              <ArrowUpRight className="w-4 h-4 shrink-0" />
            </Link>
          </div>
        )}
      </div>
    </div>
  );
};
