import React from 'react';
import { Link } from 'react-router-dom';
import { Home } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export const NotFoundPage: React.FC = () => {
  const { t } = useTranslation();

  return (
    <div className="min-h-dvh flex flex-col items-center justify-center p-4 bg-slate-50 dark:bg-slate-950 text-center">
      <h1 className="text-6xl sm:text-7xl font-extrabold text-indigo-600 dark:text-indigo-400 mb-2">404</h1>
      <h2 className="text-xl sm:text-2xl font-bold text-slate-800 dark:text-slate-100 mb-2 break-words max-w-full">
        {t('not_found.title')}
      </h2>
      <p className="text-sm text-slate-500 dark:text-slate-400 max-w-sm mb-6 break-words">
        {t('not_found.desc')}
      </p>
      <Link
        to="/"
        className="inline-flex min-h-11 items-center justify-center gap-2 px-6 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white font-medium text-sm transition"
      >
        <Home className="w-4 h-4" />
        <span>{t('not_found.back_home')}</span>
      </Link>
    </div>
  );
};
