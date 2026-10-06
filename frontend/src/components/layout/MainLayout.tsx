import React, { useEffect, useRef, useState } from 'react';
import { Outlet, useLocation } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Navbar } from './Navbar';

// MainLayout provides the app shell with responsive sidebar navigation and a chat-aware content pane.
export const MainLayout: React.FC = () => {
  const location = useLocation();
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const prevPathRef = useRef(location.pathname);
  const isChat = location.pathname === '/' || location.pathname === '/chat';

  useEffect(() => {
    if (prevPathRef.current === location.pathname) return;
    prevPathRef.current = location.pathname;
    setIsDrawerOpen(false);
  }, [location.pathname]);

  return (
    <div className="flex h-dvh overflow-hidden bg-slate-50 dark:bg-slate-950">
      <Sidebar isMobileOpen={isDrawerOpen} onMobileClose={() => setIsDrawerOpen(false)} />
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <Navbar onMenuClick={() => setIsDrawerOpen(true)} />
        <main
          className={
            isChat
              ? 'relative flex min-w-0 flex-1 flex-col overflow-hidden p-0'
              : 'flex-1 overflow-y-auto p-4 sm:p-6 lg:p-8'
          }
        >
          <Outlet />
        </main>
      </div>
    </div>
  );
};
