import React from 'react';
import { useTranslation } from 'react-i18next';
import { CheckCircle2, Circle } from 'lucide-react';
import type { RequirementProgress } from '@/types/curriculum';

interface RequirementProgressListProps {
  items?: RequirementProgress[];
}

// RequirementProgressList shows which graduation requirements are done and what each group still needs.
export const RequirementProgressList: React.FC<RequirementProgressListProps> = ({ items }) => {
  const { t } = useTranslation();
  if (!items || items.length === 0) return null;

  return (
    <div className="rounded-xl border border-emerald-100 dark:border-emerald-900/30 bg-emerald-50/40 dark:bg-emerald-950/20 p-3.5">
      <h5 className="text-xs font-bold text-emerald-950 dark:text-emerald-200 mb-2">
        {t('roadmap_requirements.title')}
      </h5>
      <ul className="space-y-1.5 text-xs text-slate-700 dark:text-slate-300">
        {items.map((it) => {
          const remaining = Math.max(0, it.required_credits - it.earned_credits);
          return (
            <li key={`${it.rule_type}-${it.group_key}`} className="flex items-start gap-2">
              {it.satisfied ? (
                <CheckCircle2 className="h-3.5 w-3.5 mt-0.5 shrink-0 text-emerald-500" />
              ) : (
                <Circle className="h-3.5 w-3.5 mt-0.5 shrink-0 text-amber-500" />
              )}
              <span className="leading-relaxed">
                <span className="font-semibold">
                  {t(`roadmap_requirements.type_${it.rule_type}`, { defaultValue: it.rule_type })}
                </span>
                {it.group_key ? ` · ${it.group_key}` : ''}
                {it.rule_type === 'CHOOSE_ONE' && it.chosen
                  ? ` — ${t('roadmap_requirements.chosen', { codes: it.chosen.split(',').join(', ') })}`
                  : ` — ${t('roadmap_requirements.earned', {
                      earned: it.earned_credits,
                      required: it.required_credits,
                    })}`}
                {!it.satisfied && remaining > 0
                  ? ` · ${t('roadmap_requirements.missing', { credits: remaining })}`
                  : ''}
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
};
