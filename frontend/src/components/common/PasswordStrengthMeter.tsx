import React, { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Check, X } from 'lucide-react';

export interface PasswordStrengthMeterProps {
  password: string;
  showRules?: boolean;
  className?: string;
}

interface RuleItem {
  id: string;
  labelKey: string;
  passed: boolean;
}

/** Password strength meter component showing visual strength levels and requirement checklist. */
export const PasswordStrengthMeter: React.FC<PasswordStrengthMeterProps> = ({
  password,
  showRules = true,
  className = '',
}) => {
  const { t } = useTranslation();

  const rules: RuleItem[] = useMemo(() => {
    const hasMinLength = password.length >= 10;
    const hasLetter = /[a-zA-Z\p{L}]/u.test(password);
    const hasNumber = /[0-9]/.test(password);
    const hasSpecialOrMixed =
      (/[a-z]/.test(password) && /[A-Z]/.test(password)) || /[^a-zA-Z0-9\s]/.test(password);

    return [
      { id: 'min_length', labelKey: 'auth.rule_min_length', passed: hasMinLength },
      { id: 'has_letter', labelKey: 'auth.rule_has_letter', passed: hasLetter },
      { id: 'has_number', labelKey: 'auth.rule_has_number', passed: hasNumber },
      { id: 'has_special_or_mixed', labelKey: 'auth.rule_has_special_or_mixed', passed: hasSpecialOrMixed },
    ];
  }, [password]);

  const { score, levelKey, levelColor, barBg } = useMemo(() => {
    if (!password) {
      return { score: 0, levelKey: '', levelColor: 'text-slate-400', barBg: 'bg-slate-200 dark:bg-slate-700/60' };
    }

    const passedCount = rules.filter((r) => r.passed).length;
    const hasMinLength = rules.find((r) => r.id === 'min_length')?.passed;

    let computedScore = passedCount;
    if (!hasMinLength && computedScore > 2) {
      computedScore = 2;
    }

    if (computedScore <= 1) {
      return { score: 1, levelKey: 'auth.password_strength_weak', levelColor: 'text-rose-500 dark:text-rose-400', barBg: 'bg-rose-500' };
    }
    if (computedScore === 2) {
      return { score: 2, levelKey: 'auth.password_strength_fair', levelColor: 'text-amber-500 dark:text-amber-400', barBg: 'bg-amber-500' };
    }
    if (computedScore === 3) {
      return { score: 3, levelKey: 'auth.password_strength_good', levelColor: 'text-sky-500 dark:text-sky-400', barBg: 'bg-sky-500' };
    }
    return { score: 4, levelKey: 'auth.password_strength_strong', levelColor: 'text-emerald-500 dark:text-emerald-400', barBg: 'bg-emerald-500' };
  }, [password, rules]);

  return (
    <div className={`space-y-2 mt-1.5 ${className}`}>
      <div className="flex items-center justify-between text-[11px]">
        <span className="text-slate-500 dark:text-slate-400 font-medium">
          {t('auth.password_strength_label')}
        </span>
        {levelKey && (
          <span className={`font-semibold ${levelColor}`}>
            {t(levelKey)}
          </span>
        )}
      </div>

      <div className="grid grid-cols-4 gap-1.5 h-1.5 w-full">
        {[1, 2, 3, 4].map((step) => {
          const isActive = score >= step;
          return (
            <div
              key={step}
              className={`h-full rounded-full transition-colors duration-300 ${
                isActive ? barBg : 'bg-slate-200 dark:bg-slate-700/60'
              }`}
            />
          );
        })}
      </div>

      {showRules && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-1.5 pt-1">
          {rules.map((rule) => (
            <div
              key={rule.id}
              className={`flex items-center gap-1.5 text-[11px] transition-colors ${
                rule.passed
                  ? 'text-emerald-600 dark:text-emerald-400 font-medium'
                  : 'text-slate-400 dark:text-slate-500'
              }`}
            >
              {rule.passed ? (
                <Check className="w-3.5 h-3.5 shrink-0 text-emerald-500" />
              ) : (
                <X className="w-3.5 h-3.5 shrink-0 text-slate-300 dark:text-slate-600" />
              )}
              <span className="leading-tight">{t(rule.labelKey)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
