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
    <div className="space-y-6 sm:space-y-8 max-w-6xl mx-auto">
      <div className="relative overflow-hidden rounded-2xl bg-gradient-to-r from-indigo-600 via-indigo-700 to-violet-800 text-white p-5 sm:p-8 shadow-xl shadow-indigo-600/15">
        <div className="relative z-10 max-w-2xl">
          <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-white/15 border border-white/20 mb-4">
            <Sparkles className="w-3.5 h-3.5 text-amber-300 shrink-0" />
            {t('dashboard.welcome_badge')}
          </span>
          <h2 className="text-2xl sm:text-3xl font-extrabold tracking-tight break-words">
            {t('dashboard.welcome_title', { name: user?.full_name || '' })}
          </h2>
          <p className="text-indigo-100 text-sm mt-2 leading-relaxed break-words">
            {t('dashboard.welcome_desc')}
          </p>

          <div className="mt-6 flex flex-col sm:flex-row sm:flex-wrap gap-3">
            <Link
              to="/"
              className="inline-flex items-center justify-center gap-2 px-5 py-2.5 min-h-11 w-full sm:w-auto rounded-xl bg-white text-indigo-600 font-semibold text-sm shadow-md hover:bg-indigo-50 transition"
            >
              <MessageSquare className="w-4 h-4 shrink-0" />
              <span>{t('dashboard.action_chat')}</span>
            </Link>
            {canAccessAdmin && (
              <Link
                to="/admin/users"
                className="inline-flex items-center justify-center gap-2 px-5 py-2.5 min-h-11 w-full sm:w-auto rounded-xl bg-indigo-500/30 hover:bg-indigo-500/40 text-white font-semibold text-sm border border-white/20 transition"
              >
                <Users className="w-4 h-4 shrink-0" />
                <span>{t('dashboard.action_users')}</span>
              </Link>
            )}
          </div>
        </div>

        <div className="absolute right-0 top-0 bottom-0 w-1/3 bg-gradient-to-l from-violet-500/30 to-transparent pointer-events-none" />
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
        <div className="p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col justify-between min-w-0">
          <div>
            <div className="w-12 h-12 rounded-xl bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400 flex items-center justify-center mb-4 shrink-0">
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
            className="mt-4 pt-3 min-h-11 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between gap-2 text-xs font-semibold text-indigo-600 dark:text-indigo-400 hover:text-indigo-700"
          >
            <span className="min-w-0">{t('dashboard.card_advisory_link')}</span>
            <ArrowUpRight className="w-4 h-4 shrink-0" />
          </Link>
        </div>

        <div className="p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col justify-between min-w-0">
          <div>
            <div className="w-12 h-12 rounded-xl bg-violet-50 dark:bg-violet-950/60 text-violet-600 dark:text-violet-400 flex items-center justify-center mb-4 shrink-0">
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
            to="/"
            className="mt-4 pt-3 min-h-11 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between gap-2 text-xs font-semibold text-violet-600 dark:text-violet-400 hover:text-violet-700"
          >
            <span className="min-w-0">{t('dashboard.card_curriculum_link')}</span>
            <ArrowUpRight className="w-4 h-4 shrink-0" />
          </Link>
        </div>

        {canAccessAdmin ? (
          <div className="sm:col-span-2 lg:col-span-1 p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col justify-between min-w-0">
            <div>
              <div className="w-12 h-12 rounded-xl bg-amber-50 dark:bg-amber-950/60 text-amber-600 dark:text-amber-400 flex items-center justify-center mb-4 shrink-0">
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
              className="mt-4 pt-3 min-h-11 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between gap-2 text-xs font-semibold text-amber-600 dark:text-amber-400 hover:text-amber-700"
            >
              <span className="min-w-0">{t('dashboard.card_admin_link')}</span>
              <ArrowUpRight className="w-4 h-4 shrink-0" />
            </Link>
          </div>
        ) : (
          <div className="sm:col-span-2 lg:col-span-1 p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col justify-between min-w-0">
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
              className="mt-4 pt-3 min-h-11 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between gap-2 text-xs font-semibold text-emerald-600 dark:text-emerald-400 hover:text-emerald-700"
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
