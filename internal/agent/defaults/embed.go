package defaults

import (
	"embed"
	"fmt"
)

//go:embed soul.md rule.md skills/*.md
var DefaultsFS embed.FS

const (
	PromptSoulID = "soul"
	PromptRuleID = "rule"

	SkillSearchRegulationsID = "search_regulations"
	SkillAcademicStandingID  = "academic_standing"
	SkillTuitionCalcID       = "tuition_calculation"
	SkillCourseRegID         = "course_registration"
	SkillInquiryEscalationID = "inquiry_escalation"
)

// DefaultPromptMeta holds metadata for embedded default prompts.
type DefaultPromptMeta struct {
	ID    string
	Title string
	Path  string
}

// DefaultSkillMeta holds metadata for embedded default skills.
type DefaultSkillMeta struct {
	ID          string
	Name        string
	Description string
	Path        string
	ToolDef     string
	Priority    int
}

// GetDefaultPromptsMetadata returns all default prompt definitions.
func GetDefaultPromptsMetadata() []DefaultPromptMeta {
	return []DefaultPromptMeta{
		{
			ID:    PromptSoulID,
			Title: "Căn Tính & Sứ Mệnh Cố Vấn Học Vụ TLU",
			Path:  "soul.md",
		},
		{
			ID:    PromptRuleID,
			Title: "Quy Tắc Hoạt Động & Ranh Giới Pháp Lý",
			Path:  "rule.md",
		},
	}
}

// GetDefaultSkillsMetadata returns all default skill definitions.
func GetDefaultSkillsMetadata() []DefaultSkillMeta {
	return []DefaultSkillMeta{
		{
			ID:          SkillSearchRegulationsID,
			Name:        "Tra cứu Quy chế & Văn bản Học vụ",
			Description: "Tra cứu, diễn giải điều khoản quy chế đào tạo, chuẩn đầu ra TOEIC và thông báo học vụ của TLU.",
			Path:        "skills/search_regulations.md",
			Priority:    100,
		},
		{
			ID:          SkillAcademicStandingID,
			Name:        "Đánh giá Học lực & Cảnh báo Học vụ",
			Description: "Phân tích điểm số CPA/GPA, đánh giá nguy cơ cảnh báo học tập (Điều 18, Điều 20) và điều kiện tốt nghiệp.",
			Path:        "skills/academic_standing.md",
			Priority:    80,
		},
		{
			ID:          SkillTuitionCalcID,
			Name:        "Tính toán Học phí & Hạn nộp",
			Description: "Tính toán học phí theo tín chỉ, tra cứu trực tiếp tại cổng https://hocphi.thanglong.edu.vn/pay/thanglong và hướng dẫn xử lý nợ học phí.",
			Path:        "skills/tuition_calculation.md",
			ToolDef:     `{"name":"lookup_tuition_fee","description":"Tra cứu học phí thực tế, nợ học phí và tình trạng thanh toán của sinh viên TLU từ cổng hocphi.thanglong.edu.vn theo Mã sinh viên","parameters":{"type":"object","properties":{"student_id":{"type":"string","description":"Mã sinh viên TLU (ví dụ: A44519)"}},"required":["student_id"]}}`,
			Priority:    60,
		},
		{
			ID:          SkillCourseRegID,
			Name:        "Hướng dẫn Đăng ký Tín chỉ",
			Description: "Hướng dẫn quy trình đăng ký môn học, kiểm tra điều kiện môn tiên quyết, học trước, hủy môn và rút bớt học phần.",
			Path:        "skills/course_registration.md",
			Priority:    40,
		},
		{
			ID:          SkillInquiryEscalationID,
			Name:        "Chuyển tiếp Thắc mắc & Khiếu nại",
			Description: "Nhận diện và đóng gói các tình huống học vụ phức tạp, khiếu nại điểm để chuyển tiếp đến Giảng viên / Cố vấn chuyên trách.",
			Path:        "skills/inquiry_escalation.md",
			ToolDef:     `{"name":"escalate_inquiry","description":"Escalate unresolved or sensitive inquiry to human advisors"}`,
			Priority:    20,
		},
	}
}

// ReadDefaultFile reads content of an embedded default file.
func ReadDefaultFile(path string) (string, error) {
	data, err := DefaultsFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded default file %s: %w", path, err)
	}
	return string(data), nil
}
