import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Modal } from '@/components/common/Modal';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { memoryService } from '@/services/memoryService';
import type { User } from '@/types';

interface StudentMemoryModalProps {
  user: User | null;
  onClose: () => void;
}

// StudentMemoryModal shows an administrator the academic profile and AI memories retained for a student.
export const StudentMemoryModal: React.FC<StudentMemoryModalProps> = ({ user, onClose }) => {
  const { t } = useTranslation();
  const { data, isLoading, isError } = useQuery({
    queryKey: ['admin-student-memory', user?.id],
    queryFn: () => memoryService.getStudentBundle(user!.id),
    enabled: Boolean(user),
  });

  const profile = data?.profile;
  const memories = data?.memories ?? [];
  const rows: Array<[string, string | number]> = profile
    ? [
        [t('profile.academic_cohort'), profile.cohort || '—'],
        [t('profile.academic_major'), profile.major || '—'],
        [t('profile.academic_credits'), profile.completed_credits],
        [t('profile.academic_standing'), profile.academic_standing || '—'],
        [t('profile.academic_gpa'), profile.cumulative_gpa],
        [t('profile.academic_target_gpa'), profile.target_gpa],
      ]
    : [];

  return (
    <Modal
      isOpen={Boolean(user)}
      onClose={onClose}
      title={t('users.memory_modal_title', { name: user?.full_name ?? '' })}
      maxWidth="2xl"
    >
      {isLoading ? (
        <div className="py-8 flex justify-center">
          <LoadingSpinner />
        </div>
      ) : isError ? (
        <div className="text-sm text-rose-600">{t('users.memory_load_failed')}</div>
      ) : (
        <div className="space-y-5">
          <section>
            <h4 className="text-sm font-bold text-slate-900 dark:text-slate-100 mb-2">
              {t('profile.academic_title')}
            </h4>
            {profile ? (
              <dl className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-2 text-xs">
                {rows.map(([label, value]) => (
                  <div key={label} className="flex justify-between gap-3 border-b border-slate-100 dark:border-slate-800 py-1">
                    <dt className="text-slate-500">{label}</dt>
                    <dd className="font-semibold text-slate-800 dark:text-slate-200">{value}</dd>
                  </div>
                ))}
                {profile.advisor_notes && (
                  <div className="sm:col-span-2 text-slate-600 dark:text-slate-300">{profile.advisor_notes}</div>
                )}
              </dl>
            ) : (
              <div className="text-xs text-slate-400 italic">{t('users.memory_no_profile')}</div>
            )}
          </section>

          <section>
            <h4 className="text-sm font-bold text-slate-900 dark:text-slate-100 mb-2">{t('profile.memory_title')}</h4>
            {memories.length === 0 ? (
              <div className="text-xs text-slate-400 italic">{t('profile.memory_empty')}</div>
            ) : (
              <ul className="space-y-2">
                {memories.map((m) => (
                  <li
                    key={m.id}
                    className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/80 dark:border-slate-800"
                  >
                    <div className="text-xs font-semibold text-slate-800 dark:text-slate-200 break-words">
                      {m.memory_key}
                    </div>
                    <div className="text-xs text-slate-600 dark:text-slate-400 break-words">{m.memory_value}</div>
                    <div className="text-[10px] text-slate-400 mt-1">
                      {m.category} · {m.source} · {Math.round(m.confidence * 100)}%
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>
      )}
    </Modal>
  );
};
