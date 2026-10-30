DELETE FROM curriculums
 WHERE NOT EXISTS (
    SELECT 1 FROM curriculum_courses c WHERE c.curriculum_id = curriculums.id
 );
