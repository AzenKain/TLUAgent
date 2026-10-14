import React from 'react';

export interface StatCardProps {
  icon: React.ReactNode;
  label: string;
  value: string | number;
  subtext?: React.ReactNode;
  variant?: 'blue' | 'emerald' | 'amber' | 'rose' | 'sky';
  className?: string;
}

// StatCard displays unified administrative key performance metrics and counts.
export const StatCard: React.FC<StatCardProps> = ({
  icon,
  label,
  value,
  subtext,
  variant = 'blue',
  className = '',
}) => {
  const variantStyles = {
    blue: 'bg-blue-50 text-tlu-blue dark:bg-blue-950/60 dark:text-tlu-cyan border-blue-200/80 dark:border-blue-900/50',
    emerald: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-950/60 dark:text-emerald-400 border-emerald-200/80 dark:border-emerald-900/50',
    amber: 'bg-amber-50 text-amber-600 dark:bg-amber-950/60 dark:text-amber-400 border-amber-200/80 dark:border-amber-900/50',
    rose: 'bg-rose-50 text-tlu-red dark:bg-rose-950/60 dark:text-rose-400 border-rose-200/80 dark:border-rose-900/50',
    sky: 'bg-sky-50 text-sky-600 dark:bg-sky-950/60 dark:text-sky-400 border-sky-200/80 dark:border-sky-900/50',
  };

  return (
    <div
      className={`rounded-2xl border border-slate-200 bg-white p-4 sm:p-5 shadow-xs transition hover:shadow-md dark:border-blue-900/30 dark:bg-[#111A35] ${className}`}
    >
      <div className="flex items-center gap-3.5">
        <div
          className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border ${variantStyles[variant]}`}
        >
          {icon}
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-xs font-medium text-slate-500 dark:text-slate-400 truncate">
            {label}
          </p>
          <p className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100 mt-0.5">
            {value}
          </p>
          {subtext && <div className="mt-1 text-xs">{subtext}</div>}
        </div>
      </div>
    </div>
  );
};
