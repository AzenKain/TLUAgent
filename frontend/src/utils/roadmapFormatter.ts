import type { DegreeDowngradeRiskWarning } from '@/types/curriculum';

type TranslationFn = (key: string, options?: Record<string, unknown>) => string;

/**
 * Formats a degree downgrade warning object into a localized message.
 */
export function formatDowngradeWarning(
  downgrade: DegreeDowngradeRiskWarning | undefined,
  t: TranslationFn
): string {
  if (!downgrade) return '';
  if (downgrade.total_retake_credits !== undefined && downgrade.max_allowed_credits !== undefined) {
    return t('student_roadmap.downgrade_risk_warning', {
      total: downgrade.total_retake_credits.toFixed(1),
      max: downgrade.max_allowed_credits.toFixed(1),
    });
  }
  return downgrade.warning_message || '';
}

/**
 * Translates academic roadmap recommendation strings using i18n keys.
 */
export function formatRoadmapRecommendation(rec: string, t: TranslationFn): string {
  if (!rec) return '';

  const critMatch = rec.match(/^Critical warning:\s*Accumulated\s+([0-9.]+)\s+retake credits/i);
  if (critMatch) {
    return t('student_roadmap.rec_critical_warning', { total: critMatch[1] });
  }

  const retakeMatch = rec.match(
    /^Retake credit monitoring:\s*Currently accumulated\s+([0-9.]+)\s*\/\s*([0-9.]+)\s+safe retake credits/i
  );
  if (retakeMatch) {
    return t('student_roadmap.rec_retake_monitoring', {
      total: retakeMatch[1],
      max: retakeMatch[2],
    });
  }

  const critPathMatch = rec.match(
    /^High priority critical path courses:\s*(.*?)\.\s*Failing these will delay graduation timeline\.?$/i
  );
  if (critPathMatch) {
    return t('student_roadmap.rec_critical_path', { courses: critPathMatch[1] });
  }

  const freeNeedMatch = rec.match(
    /^Free electives:\s*Need\s+([0-9.]+)\s+additional free elective credits/i
  );
  if (freeNeedMatch) {
    return t('student_roadmap.rec_free_electives_needed', { credits: freeNeedMatch[1] });
  }

  if (/^Free electives:\s*Completed all\s+5\.0\s+required credits/i.test(rec)) {
    return t('student_roadmap.rec_free_electives_completed');
  }

  return rec;
}
