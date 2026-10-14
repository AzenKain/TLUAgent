package constants

const (
	JobTypeCleanCorpus        = "clean_corpus"
	JobTypeCrawlNotices       = "crawl_notices"
	JobTypeCrawlNews          = "crawl_news"
	JobTypeCrawlStaticPages   = "crawl_static_pages"
	JobTypeSupersedeCheck     = "supersede_check"

	JobStatusPending     = "pending"
	JobStatusRunning     = "running"
	JobStatusCompleted   = "completed"
	JobStatusFailed      = "failed"
	JobStatusInterrupted = "interrupted"

	SlugNotices = "thong-bao"
	SlugNews    = "tin-tuc"

	DefaultUniversityBaseURL = "https://thanglong.edu.vn"
)
