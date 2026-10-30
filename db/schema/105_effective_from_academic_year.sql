UPDATE documents
   SET effective_from = substr(academic_year, 1, 4) || '-09-01'
 WHERE (effective_from IS NULL OR trim(effective_from) = '')
   AND academic_year GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9][0-9][0-9]';
