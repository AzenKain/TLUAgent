package memory

import (
	"fmt"
	"strings"
	"time"
)

type AcademicStanding string

const (
	StandingNormal    AcademicStanding = "NORMAL"
	StandingWarning   AcademicStanding = "ACADEMIC_WARNING"
	StandingProbation AcademicStanding = "PROBATION"
)

type StudentAcademicProfile struct {
	UserID           string            `json:"user_id"`
	StudentID        string            `json:"student_id"`
	FullName         string            `json:"full_name"`
	Major            string            `json:"major"`
	CohortYear       int               `json:"cohort_year"`
	AcademicStanding AcademicStanding  `json:"academic_standing"`
	CompletedCredits int               `json:"completed_credits"`
	CumulativeGPA    float64           `json:"cumulative_gpa"`
	Preferences      map[string]string `json:"preferences,omitempty"`
}

type LearnedKnowledgeItem struct {
	ID             string    `json:"id"`
	Topic          string    `json:"topic"`
	Question       string    `json:"question"`
	VerifiedAnswer string    `json:"verified_answer"`
	AnsweredBy     string    `json:"answered_by"`
	Department     string    `json:"department"`
	CreatedAt      time.Time `json:"created_at"`
	UsageCount     int       `json:"usage_count"`
}

// FormatStudentPromptContext transforms student profile into a system prompt injection snippet.
func FormatStudentPromptContext(p *StudentAcademicProfile) string {
	if p == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("[HỒ SƠ HỌC VỤ SINH VIÊN - NGƯỜI DÙNG ĐÃ XÁC THỰC]\n")
	if p.CohortYear > 0 && p.Major != "" {
		sb.WriteString("- Trạng thái: Sinh viên đã xác thực. Khóa học và Ngành đã được ghi nhận chính thức.\n")
	} else {
		sb.WriteString("- Trạng thái: Sinh viên đã xác thực. LƯU Ý: Khóa học hoặc Ngành chưa rõ trong hồ sơ.\n")
		sb.WriteString("- YÊU CẦU BẮT BUỘC: Khi sinh viên hỏi về chương trình đào tạo, môn thay thế, chuẩn đầu ra hoặc học phí, bạn PHẢI chủ động hỏi sinh viên Khóa (K mấy?) và/hoặc Ngành trước khi tư vấn chi tiết.\n")
	}
	if p.StudentID != "" {
		fmt.Fprintf(&sb, "- Mã sinh viên / Student ID: %s\n", p.StudentID)
	}
	if p.FullName != "" {
		fmt.Fprintf(&sb, "- Họ và tên: %s\n", p.FullName)
	}
	if p.Major != "" {
		fmt.Fprintf(&sb, "- Ngành học: %s\n", p.Major)
	} else {
		sb.WriteString("- Ngành học: CHƯA RÕ (Bắt buộc hỏi sinh viên khi cần thiết)\n")
	}
	if p.CohortYear > 0 {
		fmt.Fprintf(&sb, "- Khóa học: K%d\n", p.CohortYear)
	} else {
		sb.WriteString("- Khóa học: CHƯA RÕ (Bắt buộc hỏi sinh viên khi cần thiết)\n")
	}
	if p.CompletedCredits > 0 {
		fmt.Fprintf(&sb, "- Tín chỉ đã tích lũy: %d\n", p.CompletedCredits)
	}
	if p.CumulativeGPA > 0 {
		fmt.Fprintf(&sb, "- Điểm trung bình tích lũy (CPA): %.2f\n", p.CumulativeGPA)
	}
	if p.AcademicStanding != "" {
		fmt.Fprintf(&sb, "- Tình trạng học vụ: %s\n", p.AcademicStanding)
	}
	if len(p.Preferences) > 0 {
		for k, v := range p.Preferences {
			fmt.Fprintf(&sb, "- Thiết lập cá nhân [%s]: %s\n", k, v)
		}
	}
	return sb.String()
}

// FormatLearnedKnowledgePromptContext builds system prompt context from teacher-verified QAs.
func FormatLearnedKnowledgePromptContext(items []LearnedKnowledgeItem) string {
	if len(items) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("[CƠ SỞ TRI THỨC HỌC VỤ ĐÃ ĐƯỢC GIẢNG VIÊN / CỐ VẤN PHÊ DUYỆT]\n")
	for i, item := range items {
		fmt.Fprintf(&sb, "[%d] Chủ đề: %s (%s)\n", i+1, item.Topic, item.Department)
		fmt.Fprintf(&sb, "    Hỏi: %s\n", item.Question)
		fmt.Fprintf(&sb, "    Đáp: %s\n", item.VerifiedAnswer)
	}
	return sb.String()
}
