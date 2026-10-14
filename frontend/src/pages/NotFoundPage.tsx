import React from 'react';
import { Link } from 'react-router-dom';
import { Home } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export const NotFoundPage: React.FC = () => {
  const { t } = useTranslation();

  return (
    <div className="min-h-dvh flex flex-col items-center justify-center p-4 bg-[#F8FAFC] dark:bg-[#0B1021] text-center">
      <div className="mb-4 flex items-center justify-center">
        <div className="flex h-20 w-20 items-center justify-center rounded-2xl bg-white p-2.5 border border-slate-200 shadow-md">
          <img src="/logo.png" alt="Thang Long University" className="h-full w-full object-contain" />
        </div>
      </div>
      <h1 className="text-6xl sm:text-7xl font-extrabold text-tlu-navy dark:text-tlu-blue mb-2 font-mono">404</h1>
      <h2 className="text-xl sm:text-2xl font-bold text-slate-900 dark:text-slate-100 mb-2 wrap-break-word max-w-full">
        {t('not_found.title')}
      </h2>
      <p className="text-sm text-slate-600 dark:text-slate-400 max-w-sm mb-6 wrap-break-word">
        {t('not_found.desc')}
      </p>
      <Link
        to="/"
        className="inline-flex min-h-11 items-center justify-center gap-2 px-6 py-2.5 rounded-xl bg-tlu-navy hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-blue-600 text-white font-semibold text-sm shadow-md shadow-blue-900/20 transition"
      >
        <Home className="w-4 h-4" />
        <span>{t('not_found.back_home')}</span>
      </Link>
    </div>
  );
};
