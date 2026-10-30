package constants

const (
	JobTypeCleanCorpus        = "clean_corpus"
	JobTypeCrawlNotices       = "crawl_notices"
	JobTypeCrawlNews          = "crawl_news"
	JobTypeCrawlStaticPages   = "crawl_static_pages"
	JobTypeSupersedeCheck     = "supersede_check"
	JobTypeReextractDocuments = "reextract_documents"

	JobStatusPending     = "pending"
	JobStatusRunning     = "running"
	JobStatusCompleted   = "completed"
	JobStatusFailed      = "failed"
	JobStatusInterrupted = "interrupted"

	SlugNotices = "thong-bao"
	SlugNews    = "tin-tuc"
)
