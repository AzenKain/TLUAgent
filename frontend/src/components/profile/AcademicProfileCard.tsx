import React, { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { GraduationCap, Save, Trash2, Plus, Brain } from 'lucide-react';
import { memoryService, type UpsertStudentProfilePayload } from '@/services/memoryService';
import { ConfirmModal } from '@/components/common/ConfirmModal';
import { getApiErrorMessage } from '@/utils/apiError';

const BUNDLE_KEY = ['my-profile-bundle'];

const emptyForm: UpsertStudentProfilePayload = {
  major: '',
  cohort: '',
  academic_standing: '',
  completed_credits: 0,
  cumulative_gpa: 0,
  target_gpa: 0,
  advisor_notes: '',
};

const num = (v: string) => (v === '' ? 0 : Number(v));

const inputClass =
  'w-full px-3 py-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-sm text-slate-800 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500/30';

// AcademicProfileCard lets a student review and edit the academic profile and AI memories the advisor keeps about them.
export const AcademicProfileCard: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<UpsertStudentProfilePayload | null>(null);
  const [message, setMessage] = useState<{ text: string; type: 'success' | 'error' } | null>(null);
  const [newKey, setNewKey] = useState('');
  const [newValue, setNewValue] = useState('');
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const { data: bundle } = useQuery({
    queryKey: BUNDLE_KEY,
    queryFn: memoryService.getMyBundle,
  });

  const stored: UpsertStudentProfilePayload = bundle?.profile
    ? {
        major: bundle.profile.major || '',
        cohort: bundle.profile.cohort || '',
        academic_standing: bundle.profile.academic_standing || '',
        completed_credits: bundle.profile.completed_credits || 0,
        cumulative_gpa: bundle.profile.cumulative_gpa || 0,
        target_gpa: bundle.profile.target_gpa || 0,
        advisor_notes: bundle.profile.advisor_notes || '',
      }
    : emptyForm;
  const form = draft ?? stored;
  const setForm = (next: UpsertStudentProfilePayload) => setDraft(next);

  const flash = (text: string, type: 'success' | 'error') => {
    setMessage({ text, type });
    setTimeout(() => setMessage(null), 4000);
  };

  const saveProfile = useMutation({
    mutationFn: () => memoryService.updateMyProfile(form),
    onSuccess: () => {
      setDraft(null);
      queryClient.invalidateQueries({ queryKey: BUNDLE_KEY });
      flash(t('profile.academic_saved'), 'success');
    },
    onError: (err: unknown) => flash(getApiErrorMessage(err, t('profile.academic_save_failed')), 'error'),
  });

  const addMemory = useMutation({
    mutationFn: () =>
      memoryService.saveMyMemory({
        category: 'preference',
        memory_key: newKey.trim(),
        memory_value: newValue.trim(),
        confidence: 1,
      }),
    onSuccess: () => {
      setNewKey('');
      setNewValue('');
      queryClient.invalidateQueries({ queryKey: BUNDLE_KEY });
    },
    onError: (err: unknown) => flash(getApiErrorMessage(err, t('profile.memory_save_failed')), 'error'),
  });

  const removeMemory = useMutation({
    mutationFn: (id: string) => memoryService.deleteMyMemory(id),
    onSuccess: () => {
      setDeletingId(null);
      queryClient.invalidateQueries({ queryKey: BUNDLE_KEY });
    },
    onError: (err: unknown) => {
      setDeletingId(null);
      flash(getApiErrorMessage(err, t('profile.memory_delete_failed')), 'error');
    },
  });

  const memories = bundle?.memories ?? [];
  return (
    <div className="p-4 sm:p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs">
      <div className="flex items-center gap-3 mb-4 pb-4 border-b border-slate-100 dark:border-slate-800">
        <div className="p-2.5 rounded-xl bg-emerald-50 dark:bg-emerald-950/60 text-emerald-600 dark:text-emerald-400">
          <GraduationCap className="w-5 h-5" />
        </div>
        <div>
          <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">{t('profile.academic_title')}</h3>
          <p className="text-xs text-slate-400">{t('profile.academic_sub')}</p>
        </div>
      </div>

      {message && (
        <div
          className={`mb-4 px-3 py-2 rounded-xl text-xs font-medium ${
            message.type === 'success'
              ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300'
              : 'bg-rose-50 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300'
          }`}
        >
          {message.text}
        </div>
      )}

      <form
        onSubmit={(e) => {
          e.preventDefault();
          saveProfile.mutate();
        }}
        className="grid grid-cols-1 sm:grid-cols-2 gap-4"
      >
        <label className="text-xs font-semibold text-slate-600 dark:text-slate-300 space-y-1">
          <span>{t('profile.academic_cohort')}</span>
          <input
            className={inputClass}
            value={form.cohort}
            maxLength={50}
            placeholder="K35"
            onChange={(e) => setForm({ ...form, cohort: e.target.value })}
          />
        </label>
        <label className="text-xs font-semibold text-slate-600 dark:text-slate-300 space-y-1">
          <span>{t('profile.academic_major')}</span>
          <input
            className={inputClass}
            value={form.major}
            maxLength={100}
            onChange={(e) => setForm({ ...form, major: e.target.value })}
          />
        </label>
        <label className="text-xs font-semibold text-slate-600 dark:text-slate-300 space-y-1">
          <span>{t('profile.academic_credits')}</span>
          <input
            type="number"
            min={0}
            max={500}
            className={inputClass}
            value={form.completed_credits}
            onChange={(e) => setForm({ ...form, completed_credits: num(e.target.value) })}
          />
        </label>
        <label className="text-xs font-semibold text-slate-600 dark:text-slate-300 space-y-1">
          <span>{t('profile.academic_standing')}</span>
          <input
            className={inputClass}
            value={form.academic_standing}
            maxLength={50}
            onChange={(e) => setForm({ ...form, academic_standing: e.target.value })}
          />
        </label>
        <label className="text-xs font-semibold text-slate-600 dark:text-slate-300 space-y-1">
          <span>{t('profile.academic_gpa')}</span>
          <input
            type="number"
            step="0.01"
            min={0}
            max={4}
            className={inputClass}
            value={form.cumulative_gpa}
            onChange={(e) => setForm({ ...form, cumulative_gpa: num(e.target.value) })}
          />
        </label>
        <label className="text-xs font-semibold text-slate-600 dark:text-slate-300 space-y-1">
          <span>{t('profile.academic_target_gpa')}</span>
          <input
            type="number"
            step="0.01"
            min={0}
            max={4}
            className={inputClass}
            value={form.target_gpa}
            onChange={(e) => setForm({ ...form, target_gpa: num(e.target.value) })}
          />
        </label>
        <label className="sm:col-span-2 text-xs font-semibold text-slate-600 dark:text-slate-300 space-y-1">
          <span>{t('profile.academic_notes')}</span>
          <textarea
            rows={3}
            maxLength={2000}
            className={inputClass}
            value={form.advisor_notes}
            onChange={(e) => setForm({ ...form, advisor_notes: e.target.value })}
          />
        </label>
        <div className="sm:col-span-2 flex justify-end">
          <button
            type="submit"
            disabled={saveProfile.isPending}
            className="px-4 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 text-white font-medium text-xs flex items-center gap-2 shadow-sm transition disabled:opacity-50"
          >
            <Save className="w-4 h-4" />
            <span>{saveProfile.isPending ? t('common.saving') : t('common.save')}</span>
          </button>
        </div>
      </form>

      <div className="mt-6 pt-6 border-t border-slate-100 dark:border-slate-800">
        <div className="flex items-center gap-2 mb-1">
          <Brain className="w-4 h-4 text-violet-500" />
          <h4 className="text-sm font-bold text-slate-900 dark:text-slate-100">{t('profile.memory_title')}</h4>
        </div>
        <p className="text-xs text-slate-500 dark:text-slate-400 mb-4">{t('profile.memory_sub')}</p>

        {memories.length === 0 ? (
          <div className="text-xs text-slate-400 italic mb-4">{t('profile.memory_empty')}</div>
        ) : (
          <ul className="space-y-2 mb-4">
            {memories.map((m) => (
              <li
                key={m.id}
                className="flex items-start justify-between gap-3 p-3 rounded-xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/80 dark:border-slate-800"
              >
                <div className="min-w-0">
                  <div className="text-xs font-semibold text-slate-800 dark:text-slate-200 break-words">
                    {m.memory_key}
                  </div>
                  <div className="text-xs text-slate-600 dark:text-slate-400 break-words">{m.memory_value}</div>
                  <div className="text-[10px] text-slate-400 mt-1">
                    {m.category} · {m.source}
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => setDeletingId(m.id)}
                  aria-label={t('common.delete')}
                  className="shrink-0 p-1.5 rounded-lg text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 transition"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </li>
            ))}
          </ul>
        )}

        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (newKey.trim() && newValue.trim()) addMemory.mutate();
          }}
          className="grid grid-cols-1 sm:grid-cols-[1fr_2fr_auto] gap-2"
        >
          <input
            className={inputClass}
            value={newKey}
            maxLength={200}
            placeholder={t('profile.memory_key_placeholder')}
            onChange={(e) => setNewKey(e.target.value)}
          />
          <input
            className={inputClass}
            value={newValue}
            maxLength={2000}
            placeholder={t('profile.memory_value_placeholder')}
            onChange={(e) => setNewValue(e.target.value)}
          />
          <button
            type="submit"
            disabled={addMemory.isPending || !newKey.trim() || !newValue.trim()}
            className="px-3 py-2 rounded-xl bg-violet-600 hover:bg-violet-700 text-white font-medium text-xs flex items-center justify-center gap-1.5 transition disabled:opacity-50"
          >
            <Plus className="w-4 h-4" />
            <span>{t('profile.memory_add')}</span>
          </button>
        </form>
      </div>

      <ConfirmModal
        isOpen={deletingId !== null}
        onClose={() => setDeletingId(null)}
        onConfirm={() => deletingId && removeMemory.mutate(deletingId)}
        title={t('profile.memory_delete_title')}
        message={t('profile.memory_delete_desc')}
        variant="danger"
        confirmText={t('common.delete')}
        cancelText={t('common.cancel')}
        isLoading={removeMemory.isPending}
      />
    </div>
  );
};
