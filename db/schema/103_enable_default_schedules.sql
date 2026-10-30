UPDATE job_schedules SET enabled = 1
 WHERE task_type IN ('crawl_notices', 'crawl_news', 'crawl_static_pages') AND enabled = 0;

UPDATE job_schedules SET enabled = 0
 WHERE task_type = 'supersede_check';
