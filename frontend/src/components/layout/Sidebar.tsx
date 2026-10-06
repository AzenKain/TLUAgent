import React, { useEffect } from 'react';
import { NavLink } from 'react-router-dom';
import {
  MessageSquare,
  MessagesSquare,
  Users,
  ShieldCheck,
  LayoutDashboard,
  UserCheck,
  Bot,
  GraduationCap,
  Cpu,
  BookOpen,
  Network,
  Workflow,
} from 'lucide-react';
import { usePermissions } from '@/hooks/usePermissions';
import { useTranslation } from 'react-i18next';

interface SidebarProps {
  isMobileOpen?: boolean;
  onMobileClose?: () => void;
}

// Sidebar renders the persistent admin rail on lg+ and an off-canvas drawer with backdrop below lg.
export const Sidebar: React.FC<SidebarProps> = ({ isMobileOpen = false, onMobileClose }) => {
  const { canAny } = usePermissions();
  const { t } = useTranslation();

  useEffect(() => {
    if (!isMobileOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onMobileClose?.();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isMobileOpen, onMobileClose]);

  const navItems = [
    {
      to: '/',
      label: '← ' + t('nav.back_to_chat'),
      icon: MessageSquare,
      show: true,
      end: true,
    },
    {
      to: '/admin',
      label: t('nav.dashboard'),
      icon: LayoutDashboard,
      show: canAny([
        'user.manage',
        'role.manage',
        'setting.manage',
        'llm.manage',
        'chat.manage',
        'admin.access',
      ]),
      end: true,
    },
    {
      to: '/admin/chats',
      label: t('nav.chats'),
      icon: MessagesSquare,
      show: canAny(['chat.manage', 'admin.access']),
      end: false,
    },
    {
      to: '/admin/users',
      label: t('nav.users'),
      icon: Users,
      show: canAny(['user.manage', 'admin.access']),
      end: false,
    },
    {
      to: '/admin/roles',
      label: t('nav.roles'),
      icon: ShieldCheck,
      show: canAny(['role.manage', 'admin.access']),
      end: false,
    },
    {
      to: '/admin/llm',
      label: t('nav.llm_settings'),
      icon: Cpu,
      show: canAny(['llm.manage', 'setting.manage', 'admin.access']),
      end: false,
    },
    {
      to: '/admin/agent',
      label: t('nav.agent'),
      icon: Bot,
      show: canAny(['llm.manage', 'setting.manage', 'admin.access']),
      end: false,
    },
    {
      to: '/admin/documents',
      label: t('nav.documents'),
      icon: BookOpen,
      show: canAny(['rag.manage', 'admin.access']),
      end: false,
    },
    {
      to: '/admin/knowledge-graph',
      label: t('nav.knowledge_graph'),
      icon: Network,
      show: canAny(['rag.manage', 'admin.access']),
      end: false,
    },
    {
      to: '/admin/jobs',
      label: t('nav.jobs'),
      icon: Workflow,
      show: canAny(['job.manage', 'job.read', 'admin.access']),
      end: false,
    },
    {
      to: '/admin/inquiries',
      label: t('nav.inquiries'),
      icon: GraduationCap,
      show: canAny(['inquiry.answer', 'inquiry.read', 'inquiry.manage', 'admin.access']),
      end: false,
    },
    {
      to: '/profile',
      label: t('nav.profile'),
      icon: UserCheck,
      show: true,
      end: false,
    },
  ];

  return (
    <>
      <div
        onClick={onMobileClose}
        aria-hidden="true"
        className={`fixed inset-0 z-40 bg-slate-950/60 transition-opacity duration-200 lg:hidden ${
          isMobileOpen ? 'opacity-100' : 'pointer-events-none opacity-0'
        }`}
      />
      <aside
        id="admin-sidebar"
        role={isMobileOpen ? 'dialog' : undefined}
        aria-modal={isMobileOpen ? true : undefined}
        aria-labelledby="admin-sidebar-heading"
        className={`fixed inset-y-0 left-0 z-50 flex w-64 max-w-[85vw] shrink-0 flex-col border-r border-slate-200 bg-white transition-transform duration-200 ease-in-out dark:border-slate-800 dark:bg-slate-900 lg:static lg:max-w-none lg:translate-x-0 ${
          isMobileOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="flex h-16 shrink-0 items-center gap-3 border-b border-slate-200 px-4 dark:border-slate-800 sm:px-6">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gradient-to-tr from-indigo-600 to-violet-500 text-white shadow-md shadow-indigo-500/20">
            <GraduationCap className="h-6 w-6" />
          </div>
          <div className="min-w-0">
            <h1
              id="admin-sidebar-heading"
              className="flex items-center gap-1.5 text-base font-bold tracking-tight text-slate-900 dark:text-slate-100"
            >
              TLUAgent
              <span className="rounded-md border border-indigo-200 bg-indigo-50 px-1.5 py-0.5 text-[10px] font-semibold text-indigo-600 dark:border-indigo-800 dark:bg-indigo-950 dark:text-indigo-400">
                {t('common.admin_badge')}
              </span>
            </h1>
            <p className="truncate text-[11px] font-medium text-slate-400">{t('common.university_name')}</p>
          </div>
        </div>

        <nav className="flex-1 space-y-1.5 overflow-y-auto p-4">
          {navItems
            .filter((item) => item.show)
            .map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  onClick={onMobileClose}
                  className={({ isActive }) =>
                    `flex min-h-11 items-center gap-3 rounded-lg px-3.5 py-2.5 text-sm font-medium transition ${
                      isActive
                        ? 'bg-indigo-600 text-white shadow-xs shadow-indigo-600/30'
                        : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100 hover:bg-slate-100 dark:hover:bg-slate-800/60'
                    }`
                  }
                >
                  <Icon className="h-4 w-4 shrink-0" />
                  <span className="min-w-0">{item.label}</span>
                </NavLink>
              );
            })}
        </nav>

        <div className="shrink-0 border-t border-slate-200 p-4 dark:border-slate-800">
          <div className="flex items-center gap-3 rounded-lg border border-indigo-100 bg-indigo-50/60 p-3 dark:border-indigo-900/50 dark:bg-indigo-950/30">
            <div className="rounded-md bg-indigo-600 p-2 text-white">
              <Bot className="h-4 w-4" />
            </div>
            <div className="min-w-0">
              <div className="text-xs font-semibold text-indigo-900 dark:text-indigo-200">
                {t('nav.ai_advisor')}
              </div>
              <div className="text-[10px] text-indigo-700/80 dark:text-indigo-400">
                {t('nav.advisor_desc')}
              </div>
            </div>
          </div>
        </div>
      </aside>
    </>
  );
};
