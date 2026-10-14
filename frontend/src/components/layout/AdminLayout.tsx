import React, { useEffect, useRef, useState } from 'react';
import { Outlet, useLocation } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Navbar } from './Navbar';

// AdminLayout provides the layout with management sidebar and header for administrative and profile screens.
export const AdminLayout: React.FC = () => {
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(() => {
    return localStorage.getItem('admin_sidebar_collapsed') === 'true';
  });
  const location = useLocation();
  const prevPathRef = useRef(location.pathname);

  useEffect(() => {
    if (prevPathRef.current === location.pathname) return;
    prevPathRef.current = location.pathname;
    setIsDrawerOpen(false);
  }, [location.pathname]);

  const toggleSidebarCollapse = () => {
    setIsSidebarCollapsed((prev) => {
      const next = !prev;
      localStorage.setItem('admin_sidebar_collapsed', String(next));
      return next;
    });
  };

  const isFullHeightPage = location.pathname.startsWith('/admin/tluclaw');

  return (
    <div className="flex h-dvh overflow-hidden bg-slate-50 dark:bg-slate-950">
      <Sidebar
        isMobileOpen={isDrawerOpen}
        onMobileClose={() => setIsDrawerOpen(false)}
        isCollapsed={isSidebarCollapsed}
        onToggleCollapse={toggleSidebarCollapse}
      />
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <Navbar onMenuClick={() => setIsDrawerOpen(true)} />
        <main
          className={`flex-1 min-h-0 ${
            isFullHeightPage
              ? 'overflow-hidden p-0 flex flex-col'
              : 'overflow-y-auto p-4 sm:p-6 lg:p-8'
          }`}
        >
          <Outlet />
        </main>
      </div>
    </div>
  );
};
