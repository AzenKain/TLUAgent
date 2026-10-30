import React, { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Plus, Trash2 } from 'lucide-react';
import { Modal } from '@/components/common/Modal';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { curriculumAdminService, type RequirementRule, type RequirementRuleType } from '@/services/curriculumAdminService';
import { getApiErrorMessage } from '@/utils/apiError';

interface RequirementRulesModalProps {
  isOpen: boolean;
  onClose: () => void;
  cohort: string;
  major: string;
  canManage: boolean;
}

const RULE_TYPES: RequirementRuleType[] = ['TOTAL_MIN', 'CHOOSE_ONE', 'ELECTIVE_POOL'];

const inputClass =
  'w-full px-2.5 py-1.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-xs text-slate-800 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500/30';

const emptyRule: RequirementRule = { rule_type: 'ELECTIVE_POOL', group_key: '', min_credits: 0, course_codes: '', note: '', source: '' };

interface EditorProps {
  cohort: string;
  major: string;
  canManage: boolean;
  initial: RequirementRule[];
  onSaved: () => void;
}

const RulesEditor: React.FC<EditorProps> = ({ cohort, major, canManage, initial, onSaved }) => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [rules, setRules] = useState<RequirementRule[]>(initial);
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: () => curriculumAdminService.replaceRequirementRules(cohort, major, rules),
    onSuccess: (saved) => {
      setRules(saved);
      setError(null);
      queryClient.invalidateQueries({ queryKey: ['admin-requirement-rules', cohort, major] });
      onSaved();
    },
    onError: (err: unknown) => setError(getApiErrorMessage(err, t('curriculum.requirements.save_failed'))),
  });

  const update = (index: number, patch: Partial<RequirementRule>) =>
    setRules(rules.map((r, i) => (i === index ? { ...r, ...patch } : r)));

  return (
    <div className="space-y-3">
      <p className="text-xs text-slate-500 dark:text-slate-400 whitespace-pre-line">{t('curriculum.requirements.help')}</p>
      {rules.length === 0 && (
        <div className="text-xs text-slate-400 italic">{t('curriculum.requirements.empty')}</div>
      )}
      {rules.map((rule, index) => (
        <div
          key={index}
          className="grid grid-cols-1 md:grid-cols-[9rem_8rem_6rem_1fr_auto] gap-2 p-3 rounded-xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/80 dark:border-slate-800"
        >
          <select
            className={inputClass}
            value={rule.rule_type}
            disabled={!canManage}
            onChange={(e) => update(index, { rule_type: e.target.value as RequirementRuleType })}
          >
            {RULE_TYPES.map((type) => (
              <option key={type} value={type}>
                {t(`curriculum.requirements.type_${type}`)}
              </option>
            ))}
          </select>
          <input
            className={inputClass}
            value={rule.group_key}
            disabled={!canManage}
            placeholder={t('curriculum.requirements.group_key')}
            onChange={(e) => update(index, { group_key: e.target.value })}
          />
          <input
            type="number"
            min={0}
            className={inputClass}
            value={rule.min_credits}
            disabled={!canManage}
            title={t('curriculum.requirements.min_credits')}
            onChange={(e) => update(index, { min_credits: Number(e.target.value) || 0 })}
          />
          <input
            className={inputClass}
            value={rule.course_codes}
            disabled={!canManage}
            placeholder={t('curriculum.requirements.course_codes')}
            onChange={(e) => update(index, { course_codes: e.target.value })}
          />
          {canManage && (
            <button
              type="button"
              onClick={() => setRules(rules.filter((_, i) => i !== index))}
              aria-label={t('common.delete')}
              className="p-1.5 rounded-lg text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          )}
          <input
            className={`${inputClass} md:col-span-2`}
            value={rule.note}
            disabled={!canManage}
            placeholder={t('curriculum.requirements.note')}
            onChange={(e) => update(index, { note: e.target.value })}
          />
          <input
            className={`${inputClass} md:col-span-3`}
            value={rule.source}
            disabled={!canManage}
            placeholder={t('curriculum.requirements.source')}
            onChange={(e) => update(index, { source: e.target.value })}
          />
        </div>
      ))}

      {error && <div className="text-xs text-rose-600">{error}</div>}

      {canManage && (
        <div className="flex flex-wrap justify-between gap-2 pt-1">
          <button
            type="button"
            onClick={() => setRules([...rules, { ...emptyRule }])}
            className="inline-flex items-center gap-1.5 px-3 py-2 rounded-xl border border-slate-200 dark:border-slate-700 text-xs font-semibold text-slate-700 dark:text-slate-200"
          >
            <Plus className="w-4 h-4" />
            <span>{t('curriculum.requirements.add')}</span>
          </button>
          <button
            type="button"
            onClick={() => save.mutate()}
            disabled={save.isPending}
            className="px-4 py-2 rounded-xl bg-tlu-blue text-white text-xs font-semibold disabled:opacity-50"
          >
            {save.isPending ? t('common.saving') : t('common.save')}
          </button>
        </div>
      )}
    </div>
  );
};

// RequirementRulesModal lets an administrator edit the graduation requirement rules the roadmap optimizer follows.
export const RequirementRulesModal: React.FC<RequirementRulesModalProps> = ({ isOpen, onClose, cohort, major, canManage }) => {
  const { t } = useTranslation();
  const { data, isLoading } = useQuery({
    queryKey: ['admin-requirement-rules', cohort, major],
    queryFn: () => curriculumAdminService.getRequirementRules(cohort, major),
    enabled: isOpen && Boolean(cohort && major),
  });

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={t('curriculum.requirements.title', { cohort, major })}
      maxWidth="5xl"
    >
      {isLoading || !data ? (
        <div className="py-8 flex justify-center">
          <LoadingSpinner />
        </div>
      ) : (
        <RulesEditor key={`${cohort}|${major}|${data.length}`} cohort={cohort} major={major} canManage={canManage} initial={data} onSaved={() => undefined} />
      )}
    </Modal>
  );
};
