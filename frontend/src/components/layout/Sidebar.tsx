import React, { useEffect } from 'react';
import { NavLink, Link } from 'react-router-dom';
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
  Layers,
  Compass,
  Sparkles,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';
import { usePermissions } from '@/hooks/usePermissions';
import { useTranslation } from 'react-i18next';

interface SidebarProps {
  isMobileOpen?: boolean;
  onMobileClose?: () => void;
  isCollapsed?: boolean;
  onToggleCollapse?: () => void;
}

// Sidebar renders the persistent admin rail on lg+ and an off-canvas drawer with backdrop below lg.
export const Sidebar: React.FC<SidebarProps> = ({
  isMobileOpen = false,
  onMobileClose,
  isCollapsed = false,
  onToggleCollapse,
}) => {
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
      label: t('nav.back_to_chat'),
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
      to: '/admin/tluclaw',
      label: t('nav.tluclaw'),
      icon: Sparkles,
      show: canAny(['tluclaw.manage', 'tluclaw.read', 'admin.access']),
      end: false,
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
      to: '/admin/curriculums',
      label: t('nav.curriculum'),
      icon: Layers,
      show: canAny(['curriculum.manage', 'curriculum.read', 'admin.access']),
      end: false,
    },
    {
      to: '/my-roadmap',
      label: t('nav.my_roadmap'),
      icon: Compass,
      show: true,
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
        className={`fixed inset-y-0 left-0 z-50 flex max-w-[85vw] shrink-0 flex-col border-r border-slate-200 bg-white transition-all duration-300 ease-in-out dark:border-slate-800 dark:bg-slate-900 lg:static lg:max-w-none ${
          isCollapsed ? 'lg:w-20' : 'lg:w-64'
        } ${
          isMobileOpen ? 'translate-x-0 w-64' : '-translate-x-full lg:translate-x-0'
        }`}
      >
        <div
          className={`flex h-16 shrink-0 items-center border-b border-slate-200 dark:border-slate-800 ${
            isCollapsed ? 'justify-center px-2' : 'justify-between px-4 sm:px-5'
          }`}
        >
          {!isCollapsed ? (
            <>
              <Link to="/" className="flex items-center gap-3 min-w-0 group">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-white p-1 border border-slate-200 shadow-xs transition group-hover:scale-105">
                  <img src="/logo.png" alt="TLU Logo" className="h-full w-full object-contain" />
                </div>
                <div className="min-w-0">
                  <h1
                    id="admin-sidebar-heading"
                    className="flex items-center gap-1.5 text-base font-bold tracking-tight text-slate-900 dark:text-slate-100"
                  >
                    <span className="text-tlu-navy dark:text-blue-400 font-extrabold">TLU</span>
                    <span className="text-tlu-red font-extrabold">Agent</span>
                    <span className="rounded-md border border-tlu-navy/20 bg-tlu-navy/5 px-1.5 py-0.5 text-[10px] font-semibold text-tlu-navy dark:border-blue-800 dark:bg-blue-950/80 dark:text-blue-300">
                      {t('common.admin_badge')}
                    </span>
                  </h1>
                  <p className="truncate text-[10px] font-medium text-slate-500 dark:text-slate-400">
                    {t('common.university_name')}
                  </p>
                </div>
              </Link>
              <button
                type="button"
                onClick={onToggleCollapse}
                title={t('common.collapse_sidebar')}
                className="hidden lg:flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition cursor-pointer shrink-0"
              >
                <ChevronLeft className="h-4 w-4" />
              </button>
            </>
          ) : (
            <button
              type="button"
              onClick={onToggleCollapse}
              title={t('common.expand_sidebar')}
              className="flex h-10 w-10 items-center justify-center rounded-xl bg-white p-1 border border-slate-200 shadow-xs hover:border-tlu-blue dark:hover:border-blue-400 transition cursor-pointer group"
            >
              <img src="/logo.png" alt="TLU Logo" className="h-full w-full object-contain group-hover:scale-95" />
            </button>
          )}
        </div>

        <nav className={`flex-1 space-y-1.5 overflow-y-auto ${isCollapsed ? 'p-2' : 'p-4'}`}>
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
                  title={isCollapsed ? item.label : undefined}
                  className={({ isActive }) =>
                    `flex min-h-11 items-center rounded-xl transition ${
                      isCollapsed
                        ? 'justify-center p-2.5'
                        : 'gap-3 px-3.5 py-2.5 text-sm font-medium'
                    } ${
                      isActive
                        ? 'bg-tlu-blue text-white shadow-xs shadow-blue-600/30 font-semibold'
                        : 'text-slate-700 dark:text-slate-300 hover:text-tlu-navy dark:hover:text-blue-300 hover:bg-slate-100/80 dark:hover:bg-slate-800/60'
                    }`
                  }
                >
                  <Icon className={`${isCollapsed ? 'h-5 w-5' : 'h-4 w-4'} shrink-0`} />
                  {!isCollapsed && <span className="min-w-0 truncate">{item.label}</span>}
                </NavLink>
              );
            })}
        </nav>

        <div className={`shrink-0 border-t border-slate-200 dark:border-blue-900/30 ${isCollapsed ? 'p-2' : 'p-3'}`}>
          {!isCollapsed ? (
            <div className="flex items-center gap-3 rounded-xl border border-blue-900/10 bg-linear-to-r from-blue-50/90 to-blue-100/50 p-3 dark:border-blue-900/40 dark:from-[#111A35] dark:to-[#0B1021]">
              <div className="rounded-lg bg-tlu-navy p-2 text-white shadow-xs">
                <Bot className="h-4 w-4 text-tlu-cyan" />
              </div>
              <div className="min-w-0">
                <div className="text-xs font-bold text-tlu-navy dark:text-blue-200 truncate">
                  {t('nav.ai_advisor')}
                </div>
                <div className="text-[10px] text-slate-500 dark:text-slate-400 truncate">
                  {t('nav.advisor_desc')}
                </div>
              </div>
            </div>
          ) : (
            <div className="flex justify-center p-1" title={`${t('nav.ai_advisor')} - ${t('nav.advisor_desc')}`}>
              <div className="rounded-xl bg-tlu-navy p-2.5 text-white shadow-xs">
                <Bot className="h-4 w-4 text-tlu-cyan" />
              </div>
            </div>
          )}

          <div className="mt-2 hidden lg:block">
            <button
              type="button"
              onClick={onToggleCollapse}
              title={isCollapsed ? t('common.expand_sidebar') : t('common.collapse_sidebar')}
              className={`flex w-full items-center rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition cursor-pointer py-1.5 ${
                isCollapsed ? 'justify-center' : 'gap-2 px-2.5 text-xs font-medium'
              }`}
            >
              {isCollapsed ? (
                <ChevronRight className="h-4 w-4" />
              ) : (
                <>
                  <ChevronLeft className="h-4 w-4 shrink-0" />
                  <span className="truncate">{t('common.collapse_sidebar')}</span>
                </>
              )}
            </button>
          </div>
        </div>
      </aside>
    </>
  );
};
