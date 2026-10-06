package tuition

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const PortalBaseURL = "https://hocphi.thanglong.edu.vn/pay/thanglong"

var (
	hotenRegex    = regexp.MustCompile(`(?i)id=["']hoten["'][^>]*value=["']([^"']*)["']`)
	khoaRegex     = regexp.MustCompile(`(?i)id=["']khoa["'][^>]*value=["']([^"']*)["']`)
	debtRegex     = regexp.MustCompile(`(?i)id=["']totalAmount["']>([^<]+)<`)
	studentRegex  = regexp.MustCompile(`^[A-Za-z0-9]{4,15}$`)
	debtRowRegex  = regexp.MustCompile(`(?s)<tr[^>]*>.*?data-title="Mã Học Phần:">([^<]+)<.*?data-title="Tên Học Phần:">([^<]+)<.*?data-title="Loại Học Phần">([^<]*)<.*?data-title="Giá tiền:">.*?([0-9\.,]+).*?</tr>`)
	invRowRegex   = regexp.MustCompile(`(?s)<tr[^>]*>.*?data-title="Mã hóa đơn:">([^<]+)<.*?data-title="Ngày tạo:">([^<]+)<.*?data-title="Ngày hết hạn:">([^<]+)<.*?data-title="Tổng:">([^<]+)<.*?data-title="Thanh toán:">([^<]+)<.*?</tr>`)
)

// UnpaidBill represents a tuition or fee item pending payment.
type UnpaidBill struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	FeeType string `json:"fee_type"`
	Amount  string `json:"amount"`
}

// InvoiceRecord represents a past invoice record on the tuition portal.
type InvoiceRecord struct {
	InvoiceCode string `json:"invoice_code"`
	CreatedDate string `json:"created_date"`
	ExpiryDate  string `json:"expiry_date"`
	TotalAmount string `json:"total_amount"`
	Status      string `json:"status"`
}

// TuitionInfo represents parsed student tuition information from the TLU tuition portal.
type TuitionInfo struct {
	StudentID      string          `json:"student_id"`
	StudentName    string          `json:"student_name"`
	Faculty        string          `json:"faculty"`
	TotalDebt      string          `json:"total_debt"`
	UnpaidBills    []UnpaidBill    `json:"unpaid_bills"`
	RecentInvoices []InvoiceRecord `json:"recent_invoices"`
	PaymentURL     string          `json:"payment_url"`
	Found          bool            `json:"found"`
}

// Client interacts with the TLU online tuition portal.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new tuition portal client.
func NewClient(timeout time.Duration, customBaseURL ...string) *Client {
	base := PortalBaseURL
	if len(customBaseURL) > 0 && strings.TrimSpace(customBaseURL[0]) != "" {
		base = strings.TrimSpace(customBaseURL[0])
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    base,
	}
}

// LookupTuition retrieves tuition and debt information for a student ID.
func (c *Client) LookupTuition(ctx context.Context, studentID string) (*TuitionInfo, error) {
	cleanID := strings.ToUpper(strings.TrimSpace(studentID))
	if !studentRegex.MatchString(cleanID) {
		return nil, fmt.Errorf("invalid student id: %s", studentID)
	}

	url := fmt.Sprintf("%s?customer=%s", c.baseURL, cleanID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) TLUAgent-Advisor/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tuition portal request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tuition portal returned http status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body failed: %w", err)
	}

	return ParseTuitionHTML(string(bodyBytes), cleanID, url), nil
}

// ParseTuitionHTML parses tuition HTML body into a structured TuitionInfo.
func ParseTuitionHTML(content string, studentID string, paymentURL string) *TuitionInfo {
	info := &TuitionInfo{
		StudentID:      studentID,
		PaymentURL:     paymentURL,
		UnpaidBills:    make([]UnpaidBill, 0),
		RecentInvoices: make([]InvoiceRecord, 0),
	}

	if m := hotenRegex.FindStringSubmatch(content); len(m) > 1 {
		info.StudentName = strings.TrimSpace(html.UnescapeString(m[1]))
	}

	if info.StudentName == "" {
		info.Found = false
		return info
	}
	info.Found = true

	if m := khoaRegex.FindStringSubmatch(content); len(m) > 1 {
		info.Faculty = strings.TrimSpace(html.UnescapeString(m[1]))
	}

	if m := debtRegex.FindStringSubmatch(content); len(m) > 1 {
		info.TotalDebt = strings.TrimSpace(m[1])
	} else {
		info.TotalDebt = "0"
	}

	debtMatches := debtRowRegex.FindAllStringSubmatch(content, -1)
	for _, match := range debtMatches {
		if len(match) > 4 {
			info.UnpaidBills = append(info.UnpaidBills, UnpaidBill{
				Code:    strings.TrimSpace(match[1]),
				Name:    strings.TrimSpace(html.UnescapeString(match[2])),
				FeeType: strings.TrimSpace(html.UnescapeString(match[3])),
				Amount:  strings.TrimSpace(match[4]),
			})
		}
	}

	invMatches := invRowRegex.FindAllStringSubmatch(content, 10)
	for _, match := range invMatches {
		if len(match) > 5 {
			info.RecentInvoices = append(info.RecentInvoices, InvoiceRecord{
				InvoiceCode: strings.TrimSpace(match[1]),
				CreatedDate: strings.TrimSpace(match[2]),
				ExpiryDate:  strings.TrimSpace(match[3]),
				TotalAmount: strings.TrimSpace(match[4]),
				Status:      strings.TrimSpace(html.UnescapeString(match[5])),
			})
		}
	}

	return info
}

// FormatPromptContext formats TuitionInfo into a clean markdown snippet for prompt injection.
func FormatPromptContext(info *TuitionInfo) string {
	if info == nil || !info.Found {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("[DỮ LIỆU TRA CỨU HỌC PHÍ THỜI GIAN THỰC TỪ CỔNG HOCPHI.THANGLONG.EDU.VN]\n")
	fmt.Fprintf(&sb, "- Mã sinh viên: %s\n", info.StudentID)
	fmt.Fprintf(&sb, "- Họ và tên: %s\n", info.StudentName)
	if info.Faculty != "" {
		fmt.Fprintf(&sb, "- Khoa/Ngành: %s\n", info.Faculty)
	}
	fmt.Fprintf(&sb, "- Tổng nợ học phí hiện tại: %s VNĐ\n", info.TotalDebt)
	fmt.Fprintf(&sb, "- Cổng thanh toán trực tuyến: %s\n", info.PaymentURL)

	if len(info.UnpaidBills) > 0 {
		sb.WriteString("- Danh sách khoản nợ chi tiết:\n")
		for _, b := range info.UnpaidBills {
			feeTypeStr := ""
			if b.FeeType != "" {
				feeTypeStr = fmt.Sprintf(" (%s)", b.FeeType)
			}
			fmt.Fprintf(&sb, "  + %s%s: %s VNĐ (Mã khoản: %s)\n", b.Name, feeTypeStr, b.Amount, b.Code)
		}
	} else {
		sb.WriteString("- Tình trạng nợ: Không có khoản nợ học phí nào đang chờ thanh toán.\n")
	}

	if len(info.RecentInvoices) > 0 {
		sb.WriteString("- Lịch sử hóa đơn gần đây:\n")
		for i, inv := range info.RecentInvoices {
			if i >= 3 {
				break
			}
			fmt.Fprintf(&sb, "  + Hóa đơn #%s: %s VNĐ - %s (Tạo ngày: %s)\n", inv.InvoiceCode, inv.TotalAmount, inv.Status, inv.CreatedDate)
		}
	}

	return sb.String()
}
