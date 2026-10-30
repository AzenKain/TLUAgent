import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from '@/components/common/Modal';
import type {
  CourseSubstitution,
  CreateSubstitutionPayload,
  UpdateSubstitutionPayload,
} from '@/types/curriculum';

interface SubstitutionModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (payload: CreateSubstitutionPayload | UpdateSubstitutionPayload) => Promise<void>;
  initialSubstitution?: CourseSubstitution | null;
  defaultCohort: string;
  defaultMajor: string;
  availableMajors?: Array<{ major_name: string; major_code: string }>;
  isSubmitting: boolean;
}

const parseInitialCohortBounds = (
  sub?: CourseSubstitution | null,
  defaultCohort?: string
): { min: number; max: number } => {
  if (
    sub &&
    sub.min_cohort !== undefined &&
    sub.max_cohort !== undefined &&
    (sub.min_cohort > 0 || sub.max_cohort > 0)
  ) {
    return { min: sub.min_cohort, max: sub.max_cohort };
  }
  const raw = (sub?.scope_cohort || defaultCohort || '').toLowerCase().trim();
  const nums = raw.match(/\d+/g)?.map(Number) || [];
  if (raw.includes('<') && nums.length > 0) {
    return { min: 1, max: nums[0] };
  }
  if (raw.includes('>') && nums.length > 0) {
    return { min: nums[0], max: 999 };
  }
  if (nums.length >= 2) {
    return { min: nums[0], max: nums[1] };
  }
  if (nums.length === 1) {
    if (nums[0] <= 35) {
      return { min: 1, max: nums[0] };
    }
    return { min: nums[0], max: nums[0] };
  }
  return { min: 36, max: 37 };
};

const getDerivedScopeCohort = (min: number, max: number): string => {
  if (min <= 1 && max >= 999) return 'ALL';
  if (min > 1 && max >= 999) return `K${min}-K39`;
  if (min === max) return `K${min}`;
  return `K${min}-K${max}`;
};

// SubstitutionModal renders a dialog for administrators to configure pairwise course replacement rules with explicit min/max cohort scopes.
export const SubstitutionModal: React.FC<SubstitutionModalProps> = ({
  isOpen,
  onClose,
  onSubmit,
  initialSubstitution,
  defaultCohort,
  defaultMajor,
  availableMajors,
  isSubmitting,
}) => {
  const { t } = useTranslation();

  const initialBounds = parseInitialCohortBounds(initialSubstitution, defaultCohort);
  const [origCode, setOrigCode] = useState(initialSubstitution?.original_course_code || '');
  const [origName, setOrigName] = useState(initialSubstitution?.original_course_name || '');
  const [origCredits, setOrigCredits] = useState<number>(initialSubstitution?.credits_original ?? 3.0);
  const [replCode, setReplCode] = useState(initialSubstitution?.replacement_course_code || '');
  const [replName, setReplName] = useState(initialSubstitution?.replacement_course_name || '');
  const [replCredits, setReplCredits] = useState<number>(initialSubstitution?.credits_replacement ?? 3.0);
  const [minCohort, setMinCohort] = useState<number>(initialBounds.min);
  const [maxCohort, setMaxCohort] = useState<number>(initialBounds.max);
  const [scopeMajor, setScopeMajor] = useState(initialSubstitution?.scope_major || defaultMajor || 'ALL');
  const [isBidirectional, setIsBidirectional] = useState(initialSubstitution?.is_bidirectional ?? true);
  const [ruleType, setRuleType] = useState(initialSubstitution?.rule_type || 'DIRECT_EQUIVALENT');
  const [ruleNote, setRuleNote] = useState(initialSubstitution?.rule_note || '');
  const [sourceReference, setSourceReference] = useState(initialSubstitution?.source_reference || '');
  const [errorMessage, setErrorMessage] = useState('');

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!initialSubstitution && (!origCode.trim() || !replCode.trim())) {
      setErrorMessage(t('curriculum.substitutions.save_error'));
      return;
    }
    if (!origName.trim() || !replName.trim()) {
      setErrorMessage(t('curriculum.substitutions.save_error'));
      return;
    }

    try {
      const derivedScope = getDerivedScopeCohort(minCohort, maxCohort);
      if (initialSubstitution) {
        const payload: UpdateSubstitutionPayload = {
          original_course_name: origName.trim(),
          replacement_course_name: replName.trim(),
          scope_cohort: derivedScope,
          scope_major: scopeMajor.trim(),
          min_cohort: minCohort,
          max_cohort: maxCohort,
          is_bidirectional: isBidirectional,
          credits_original: Number(origCredits),
          credits_replacement: Number(replCredits),
          rule_type: ruleType.trim(),
          rule_note: ruleNote.trim(),
          source_reference: sourceReference.trim(),
        };
        await onSubmit(payload);
      } else {
        const payload: CreateSubstitutionPayload = {
          original_course_code: origCode.trim().toUpperCase(),
          original_course_name: origName.trim(),
          replacement_course_code: replCode.trim().toUpperCase(),
          replacement_course_name: replName.trim(),
          scope_cohort: derivedScope,
          scope_major: scopeMajor.trim(),
          min_cohort: minCohort,
          max_cohort: maxCohort,
          is_bidirectional: isBidirectional,
          credits_original: Number(origCredits),
          credits_replacement: Number(replCredits),
          rule_type: ruleType.trim(),
          rule_note: ruleNote.trim(),
          source_reference: sourceReference.trim(),
        };
        await onSubmit(payload);
      }
      onClose();
    } catch {
      setErrorMessage(t('curriculum.substitutions.save_error'));
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={
        initialSubstitution
          ? t('curriculum.substitutions.edit_title')
          : t('curriculum.substitutions.create_title')
      }
      maxWidth="2xl"
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        {errorMessage && (
          <div className="rounded-lg bg-rose-500/10 p-3 text-sm text-rose-500 dark:bg-rose-950/30">
            {errorMessage}
          </div>
        )}

        <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-700 dark:text-amber-400">
          {t('curriculum.substitutions.non_transitive_notice')}
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.substitutions.orig_code')} *
            </label>
            <input
              type="text"
              required
              disabled={Boolean(initialSubstitution)}
              value={origCode}
              onChange={(e) => setOrigCode(e.target.value)}
              placeholder={t('curriculum.form.placeholder_code')}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 disabled:opacity-60 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.substitutions.orig_name')} *
            </label>
            <input
              type="text"
              required
              value={origName}
              onChange={(e) => setOrigName(e.target.value)}
              placeholder={t('curriculum.substitutions.placeholder_course_name')}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.substitutions.repl_code')} *
            </label>
            <input
              type="text"
              required
              disabled={Boolean(initialSubstitution)}
              value={replCode}
              onChange={(e) => setReplCode(e.target.value)}
              placeholder={t('curriculum.form.placeholder_code')}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 disabled:opacity-60 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.substitutions.repl_name')} *
            </label>
            <input
              type="text"
              required
              value={replName}
              onChange={(e) => setReplName(e.target.value)}
              placeholder={t('curriculum.substitutions.placeholder_course_name')}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.substitutions.orig_credits')}
            </label>
            <input
              type="number"
              step="0.5"
              min="0"
              value={origCredits}
              onChange={(e) => setOrigCredits(parseFloat(e.target.value) || 0)}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.substitutions.repl_credits')}
            </label>
            <input
              type="number"
              step="0.5"
              min="0"
              value={replCredits}
              onChange={(e) => setReplCredits(parseFloat(e.target.value) || 0)}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          {/* Structured Cohort Scope with Min / Max and Quick Presets */}
          <div className="sm:col-span-2 rounded-xl border border-slate-200 bg-slate-50/60 p-3.5 space-y-2.5 dark:border-slate-700/60 dark:bg-slate-800/40">
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-600 dark:text-slate-300">
              {t('curriculum.substitutions.scope_cohort_range')} *
            </label>
            <div className="flex flex-wrap items-center gap-3">
              <div className="flex items-center space-x-2">
                <span className="text-xs text-slate-500 dark:text-slate-400 font-medium">
                  {t('curriculum.substitutions.min_cohort')}:
                </span>
                <input
                  type="number"
                  min={1}
                  max={999}
                  value={minCohort}
                  onChange={(e) => setMinCohort(Math.max(1, parseInt(e.target.value, 10) || 1))}
                  className="w-24 rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm font-mono text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
                />
              </div>

              <div className="flex items-center space-x-2">
                <span className="text-xs text-slate-500 dark:text-slate-400 font-medium">
                  {t('curriculum.substitutions.max_cohort')}:
                </span>
                <input
                  type="number"
                  min={1}
                  max={999}
                  value={maxCohort}
                  onChange={(e) => setMaxCohort(Math.max(1, parseInt(e.target.value, 10) || 1))}
                  className="w-24 rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm font-mono text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
                />
              </div>

              <div className="flex flex-wrap items-center gap-1.5 pt-1 sm:pt-0">
                <button
                  type="button"
                  onClick={() => {
                    setMinCohort(1);
                    setMaxCohort(35);
                  }}
                  className={`rounded-lg border px-2.5 py-1 text-xs font-medium transition-colors ${
                    minCohort <= 1 && maxCohort === 35
                      ? 'border-tlu-blue bg-blue-50 text-tlu-blue dark:bg-blue-950/60 dark:text-blue-300'
                      : 'border-slate-200 bg-white text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700'
                  }`}
                >
                  {t('curriculum.substitutions.scope_preset_k35_down')}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setMinCohort(36);
                    setMaxCohort(37);
                  }}
                  className={`rounded-lg border px-2.5 py-1 text-xs font-medium transition-colors ${
                    minCohort === 36 && maxCohort === 37
                      ? 'border-tlu-blue bg-blue-50 text-tlu-blue dark:bg-blue-950/60 dark:text-blue-300'
                      : 'border-slate-200 bg-white text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700'
                  }`}
                >
                  {t('curriculum.substitutions.scope_preset_k36_k37')}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setMinCohort(1);
                    setMaxCohort(999);
                  }}
                  className={`rounded-lg border px-2.5 py-1 text-xs font-medium transition-colors ${
                    minCohort <= 1 && maxCohort >= 999
                      ? 'border-tlu-blue bg-blue-50 text-tlu-blue dark:bg-blue-950/60 dark:text-blue-300'
                      : 'border-slate-200 bg-white text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700'
                  }`}
                >
                  {t('curriculum.substitutions.scope_preset_all')}
                </button>
              </div>
            </div>

            <div className="flex items-center space-x-2 text-xs text-slate-500 dark:text-slate-400">
              <span className="font-semibold text-tlu-navy dark:text-blue-400">
                {t('curriculum.substitutions.range_preview', {
                  range: getDerivedScopeCohort(minCohort, maxCohort),
                })}
              </span>
              <span className="font-mono text-slate-400">
                (Min: {minCohort}, Max: {maxCohort >= 999 ? '∞' : maxCohort})
              </span>
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.substitutions.scope_major')}
            </label>
            <div className="mt-1 flex items-center space-x-2">
              <select
                value={scopeMajor}
                onChange={(e) => setScopeMajor(e.target.value)}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
              >
                <option value="ALL">{t('curriculum.substitutions.filter_all_majors')}</option>
                {availableMajors?.map((m) => (
                  <option key={m.major_code || m.major_name} value={m.major_code || m.major_name}>
                    {m.major_code ? `${m.major_code} - ` : ''}
                    {m.major_name}
                  </option>
                ))}
                {scopeMajor &&
                  scopeMajor !== 'ALL' &&
                  !availableMajors?.some(
                    (m) => m.major_code === scopeMajor || m.major_name === scopeMajor
                  ) && <option value={scopeMajor}>{scopeMajor}</option>}
              </select>
            </div>
          </div>

          <div className="sm:col-span-2">
            <label className="flex items-center space-x-2">
              <input
                type="checkbox"
                checked={isBidirectional}
                onChange={(e) => setIsBidirectional(e.target.checked)}
                className="h-4 w-4 rounded border-slate-300 text-tlu-blue focus:ring-tlu-blue dark:border-slate-700"
              />
              <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
                {t('curriculum.substitutions.is_bidirectional_label')}
              </span>
            </label>
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.substitutions.rule_type')}
            </label>
            <input
              type="text"
              value={ruleType}
              onChange={(e) => setRuleType(e.target.value)}
              placeholder={t('curriculum.substitutions.placeholder_rule_type')}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.substitutions.source_reference')}
            </label>
            <input
              type="text"
              value={sourceReference}
              onChange={(e) => setSourceReference(e.target.value)}
              placeholder={t('curriculum.substitutions.placeholder_source_reference')}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div className="sm:col-span-2">
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.substitutions.rule_note')}
            </label>
            <input
              type="text"
              value={ruleNote}
              onChange={(e) => setRuleNote(e.target.value)}
              placeholder={t('curriculum.substitutions.placeholder_rule_note')}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>
        </div>

        <div className="flex items-center justify-end space-x-3 pt-3">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
          >
            {t('curriculum.form.cancel')}
          </button>
          <button
            type="submit"
            disabled={isSubmitting}
            className="rounded-lg bg-tlu-navy px-4 py-2 text-sm font-medium text-white shadow hover:bg-tlu-navy-light dark:bg-tlu-blue dark:hover:bg-blue-600 disabled:opacity-50"
          >
            {isSubmitting ? t('common.saving') : t('curriculum.form.save')}
          </button>
        </div>
      </form>
    </Modal>
  );
};
