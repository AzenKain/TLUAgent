import React from 'react';
import { useAuth } from '@/hooks/useAuth';
import { LogOut, Menu, User as UserIcon } from 'lucide-react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';
import { NotificationBell } from '@/components/notifications/NotificationBell';

interface NavbarProps {
  onMenuClick?: () => void;
}

// Navbar renders the sticky top bar with a hamburger drawer toggle below lg and user actions.
export const Navbar: React.FC<NavbarProps> = ({ onMenuClick }) => {
  const { user, logout, isLoggingOut } = useAuth();
  const { t } = useTranslation();

  return (
    <header className="sticky top-0 z-30 flex h-16 shrink-0 items-center justify-between gap-2 border-b border-slate-200 bg-white/80 px-3 backdrop-blur-md dark:border-slate-800 dark:bg-slate-900/80 sm:px-4 md:px-6">
      <div className="flex min-w-0 flex-1 items-center gap-2 sm:gap-3">
        <button
          type="button"
          onClick={onMenuClick}
          aria-label={t('chat.open_sidebar')}
          aria-controls="admin-sidebar"
          className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg text-slate-600 transition hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800 lg:hidden"
        >
          <Menu className="h-5 w-5" />
        </button>
        <span className="hidden truncate text-sm font-medium text-slate-500 dark:text-slate-400 md:block">
          {t('common.system_title')}
        </span>
      </div>

      <div className="flex shrink-0 items-center gap-1.5 sm:gap-2">
        {user && <NotificationBell />}
        <LanguageSwitcher />

        {user && (
          <Link
            to="/profile"
            className="flex items-center gap-3 rounded-lg p-1.5 transition hover:bg-slate-100 dark:hover:bg-slate-800 sm:px-2"
          >
            <div className="flex h-8 w-8 items-center justify-center rounded-full border border-indigo-200 bg-indigo-100 text-xs font-bold uppercase text-indigo-600 dark:border-indigo-800 dark:bg-indigo-950/80 dark:text-indigo-400">
              {user.avatar_url ? (
                <img src={user.avatar_url} alt="" className="h-full w-full rounded-full object-cover" />
              ) : (
                user.full_name?.charAt(0) || <UserIcon className="h-4 w-4" />
              )}
            </div>
            <div className="hidden text-left min-[420px]:block">
              <div className="flex items-center gap-1.5 text-sm font-semibold text-slate-800 dark:text-slate-100">
                <span className="max-w-[36vw] truncate sm:max-w-none">{user.full_name}</span>
                {user.is_owner && (
                  <span className="rounded border border-amber-500/20 bg-amber-500/10 px-1 text-[10px] font-bold text-amber-500">
                    {t('common.owner')}
                  </span>
                )}
              </div>
              <div className="max-w-[36vw] truncate text-xs text-slate-500 dark:text-slate-400 sm:max-w-none">
                {user.student_code || user.email}
              </div>
            </div>
          </Link>
        )}

        <button
          type="button"
          onClick={() => logout()}
          disabled={isLoggingOut}
          title={t('nav.logout')}
          aria-label={t('nav.logout')}
          className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg text-slate-500 transition hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-950/40"
        >
          <LogOut className="h-5 w-5" />
        </button>
      </div>
    </header>
  );
};
