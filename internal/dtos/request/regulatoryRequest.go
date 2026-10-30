package request

// RegulatorySearchRegulationsFTSArgs specifies parameters for full-text search against active regulations.
type RegulatorySearchRegulationsFTSArgs struct {
	Query  string `json:"query"`
	Cohort string `json:"cohort,omitempty"`
	Domain string `json:"domain,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// RegulatoryInspectDocumentArticlesArgs specifies parameters for fetching document rules and chunks.
type RegulatoryInspectDocumentArticlesArgs struct {
	DocumentID    string `json:"document_id"`
	ArticleNumber string `json:"article_number,omitempty"`
}

// RegulatoryCompareArticleDiffArgs specifies parameters for semantic difference comparison.
type RegulatoryCompareArticleDiffArgs struct {
	OldDocumentID  string `json:"old_document_id"`
	ArticleNumber  string `json:"article_number"`
	OldArticleText string `json:"old_article_text"`
	NewArticleText string `json:"new_article_text"`
}

// RegulatorySearchKnowledgeGraphArgs specifies parameters for searching Knowledge Graph entities.
type RegulatorySearchKnowledgeGraphArgs struct {
	EntityName string `json:"entity_name"`
}

// RegulatoryInspectCohortApplicabilityArgs specifies parameters for cohort coverage analysis.
type RegulatoryInspectCohortApplicabilityArgs struct {
	DocID1 string `json:"doc_id_1"`
	DocID2 string `json:"doc_id_2"`
}

// RegulatoryProposeGranularSupersessionArgs specifies parameters for proposing article-level supersession.
type RegulatoryProposeGranularSupersessionArgs struct {
	ActionType               string   `json:"action_type"`
	TargetDocID              string   `json:"target_doc_id,omitempty"`
	SupersededArticleNumbers []string `json:"superseded_article_numbers,omitempty"`
	RetainedArticleNumbers   []string `json:"retained_article_numbers,omitempty"`
	AffectedCohorts          []string `json:"affected_cohorts,omitempty"`
	ImpactSummary            string   `json:"impact_summary"`
	RiskLevel                string   `json:"risk_level"`
	Reason                   string   `json:"reason,omitempty"`
	InquiryQuestion          string   `json:"inquiry_question,omitempty"`
}
