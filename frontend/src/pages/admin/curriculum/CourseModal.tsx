import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from '@/components/common/Modal';
import type { CurriculumCourse, CreateCoursePayload, UpdateCoursePayload } from '@/types/curriculum';

interface CourseModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (payload: CreateCoursePayload | UpdateCoursePayload) => Promise<void>;
  initialCourse?: CurriculumCourse | null;
  defaultCohort: string;
  defaultMajor: string;
  isSubmitting: boolean;
}

const sanitizePrerequisites = (text: string): string => {
  return text
    .replace(/,\s*,+/g, ',')
    .replace(new RegExp(',\\s*ho\\u1EB7c\\s*,?', 'gi'), ' ho\u1EB7c ')
    .replace(/[\s,;]+$/g, '')
    .replace(new RegExp('\\s+(?:ho\\u1EB7c|or)$', 'i'), '')
    .replace(/[\s,;]+$/g, '')
    .trim();
};

// CourseModal renders an accessible dialog to create or update curriculum course details.
export const CourseModal: React.FC<CourseModalProps> = ({
  isOpen,
  onClose,
  onSubmit,
  initialCourse,
  defaultCohort,
  defaultMajor,
  isSubmitting,
}) => {
  const { t } = useTranslation();

  const [cohort, setCohort] = useState(initialCourse?.cohort || defaultCohort || 'K36');
  const [majorName, setMajorName] = useState(initialCourse?.major_name || defaultMajor || '');
  const [courseCode, setCourseCode] = useState(initialCourse?.course_code || '');
  const [courseName, setCourseName] = useState(initialCourse?.course_name || '');
  const [credits, setCredits] = useState<number>(initialCourse?.credits ?? 3.0);
  const [weight, setWeight] = useState<number>(initialCourse?.weight ?? 1.0);
  const [hours, setHours] = useState(initialCourse?.hours || '');
  const [knowledgeBlock, setKnowledgeBlock] = useState(initialCourse?.knowledge_block || '');
  const [suggestedSemester, setSuggestedSemester] = useState<number>(initialCourse?.suggested_semester || 1);
  const [isElearning, setIsElearning] = useState<boolean>(initialCourse?.is_elearning ?? false);
  const [prerequisitesText, setPrerequisitesText] = useState(initialCourse?.prerequisites_text || '');
  const [errorMessage, setErrorMessage] = useState('');

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!initialCourse && (!cohort.trim() || !majorName.trim() || !courseCode.trim())) {
      setErrorMessage(t('curriculum.courses.save_error'));
      return;
    }
    if (!courseName.trim()) {
      setErrorMessage(t('curriculum.courses.save_error'));
      return;
    }

    try {
      if (initialCourse) {
        const payload: UpdateCoursePayload = {
          course_name: courseName.trim(),
          credits: Number(credits),
          weight: Number(weight),
          hours: hours.trim(),
          knowledge_block: knowledgeBlock.trim(),
          prerequisites_text: sanitizePrerequisites(prerequisitesText),
          suggested_semester: Number(suggestedSemester),
          is_elearning: isElearning,
        };
        await onSubmit(payload);
      } else {
        const payload: CreateCoursePayload = {
          cohort: cohort.trim().toUpperCase(),
          major_name: majorName.trim(),
          course_code: courseCode.trim().toUpperCase(),
          course_name: courseName.trim(),
          credits: Number(credits),
          weight: Number(weight),
          hours: hours.trim(),
          knowledge_block: knowledgeBlock.trim(),
          prerequisites_text: sanitizePrerequisites(prerequisitesText),
          suggested_semester: Number(suggestedSemester),
          is_elearning: isElearning,
        };
        await onSubmit(payload);
      }
      onClose();
    } catch {
      setErrorMessage(t('curriculum.courses.save_error'));
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={initialCourse ? t('curriculum.courses.edit_title') : t('curriculum.courses.create_title')}
      maxWidth="2xl"
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        {errorMessage && (
          <div className="rounded-lg bg-rose-500/10 p-3 text-sm text-rose-500 dark:bg-rose-950/30">
            {errorMessage}
          </div>
        )}

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {!initialCourse && (
            <>
              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  {t('curriculum.form.cohort')} *
                </label>
                <input
                  type="text"
                  required
                  value={cohort}
                  onChange={(e) => setCohort(e.target.value)}
                  placeholder={t('curriculum.form.placeholder_cohort')}
                  className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  {t('curriculum.form.major_name')} *
                </label>
                <input
                  type="text"
                  required
                  value={majorName}
                  onChange={(e) => setMajorName(e.target.value)}
                  placeholder={t('curriculum.form.placeholder_major_name')}
                  className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
                />
              </div>
            </>
          )}

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.form.code')} *
            </label>
            <input
              type="text"
              required
              disabled={Boolean(initialCourse)}
              value={courseCode}
              onChange={(e) => setCourseCode(e.target.value)}
              placeholder={t('curriculum.form.placeholder_code')}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 disabled:opacity-60 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.form.name')} *
            </label>
            <input
              type="text"
              required
              value={courseName}
              onChange={(e) => setCourseName(e.target.value)}
              placeholder={t('curriculum.form.placeholder_name')}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.form.credits')} *
            </label>
            <input
              type="number"
              step="0.5"
              min="0"
              required
              value={credits}
              onChange={(e) => setCredits(parseFloat(e.target.value) || 0)}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.form.weight')}
            </label>
            <input
              type="number"
              step="0.1"
              min="0"
              value={weight}
              onChange={(e) => setWeight(parseFloat(e.target.value) || 1.0)}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.form.hours')}
            </label>
            <input
              type="text"
              value={hours}
              onChange={(e) => setHours(e.target.value)}
              placeholder={t('curriculum.form.placeholder_hours')}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.form.semester')}
            </label>
            <input
              type="number"
              min="1"
              max="12"
              value={suggestedSemester}
              onChange={(e) => setSuggestedSemester(parseInt(e.target.value, 10) || 1)}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div className="sm:col-span-2">
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.form.knowledge_block')}
            </label>
            <input
              type="text"
              value={knowledgeBlock}
              onChange={(e) => setKnowledgeBlock(e.target.value)}
              placeholder={t('curriculum.form.placeholder_knowledge_block')}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          <div className="sm:col-span-2">
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {t('curriculum.form.prerequisites_text')}
            </label>
            <input
              type="text"
              value={prerequisitesText}
              onChange={(e) => setPrerequisitesText(e.target.value)}
              placeholder={t('curriculum.form.placeholder_prerequisites')}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-tlu-blue focus:ring-1 focus:ring-tlu-blue focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
            <p className="mt-1 text-xs text-slate-400 dark:text-slate-500">
              {t('curriculum.form.prerequisites_hint')}
            </p>
          </div>

          <div className="sm:col-span-2 rounded-lg border border-slate-200 bg-slate-50/50 p-3.5 dark:border-slate-800 dark:bg-slate-900/50">
            <label className="flex items-start space-x-3 cursor-pointer">
              <input
                type="checkbox"
                checked={isElearning}
                onChange={(e) => setIsElearning(e.target.checked)}
                className="mt-0.5 h-4 w-4 rounded border-slate-300 text-tlu-blue focus:ring-tlu-blue dark:border-slate-700 dark:bg-slate-800"
              />
              <div className="space-y-0.5">
                <span className="text-sm font-semibold text-slate-900 dark:text-slate-100">
                  {t('curriculum.form.is_elearning')}
                </span>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  {t('curriculum.form.is_elearning_hint')}
                </p>
              </div>
            </label>
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
