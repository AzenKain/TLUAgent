package request

type CreateInquiryRequest struct {
	ConversationID string `json:"conversation_id,omitempty"`
	StudentName    string `json:"student_name" validate:"required,min=2,max=100"`
	StudentCode    string `json:"student_code" validate:"required,min=2,max=30"`
	StudentClass   string `json:"student_class" validate:"required,min=2,max=50"`
	Question       string `json:"question" validate:"required,min=5,max=2000"`
	Context        string `json:"context,omitempty"`
}

type AnswerInquiryRequest struct {
	Reply string `json:"reply" validate:"required,min=10,max=5000"`
}

type ExpireInquiryRequest struct {
	SupersededByDocID string `json:"superseded_by_doc_id"`
	Reason            string `json:"reason" validate:"required,min=5,max=500"`
}

type ListStudentInquiriesDto struct {
	PaginationDto
}

type ListTeacherInquiriesDto struct {
	PaginationDto
	Status string `json:"status,omitempty" query:"status" validate:"omitempty,max=50"`
	Search string `json:"search,omitempty" query:"search" validate:"omitempty,max=200"`
}
