package tuition

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleHTML = `<!DOCTYPE html>
<html>
<body>
<input type="text" id="masv" value="A44519" />
<input type="text" id="hoten" value="ĐỖ DUY KH&#193;NH" />
<input type="text" id="khoa" value="Khoa C&#244;ng nghệ th&#244;ng tin" />
<span id="totalAmount">9.450.000</span> vnđ
<table>
<tr>
  <td data-title="Mã hóa đơn:">4744643049</td>
  <td data-title="Ngày tạo:">08/03/2026 22:02:42</td>
  <td data-title="Ngày hết hạn:">09/03/2026 10:02:42</td>
  <td data-title="Tổng:">17.400.000</td>
  <td data-title="Thanh toán:">Đã thanh toán</td>
</tr>
</table>
<div id="debtsModal">
<table>
<tr>
  <td data-title="Mã Học Phần:">27905122</td>
  <td data-title="Tên Học Phần:">Học phí HK01 - 2026-2027</td>
  <td data-title="Loại Học Phần">[Hoc phi]</td>
  <td data-title="Giá tiền:">9.450.000</td>
</tr>
</table>
</div>
</body>
</html>`

func TestParseTuitionHTML(t *testing.T) {
	info := ParseTuitionHTML(sampleHTML, "A44519", "https://hocphi.thanglong.edu.vn/pay/thanglong?customer=A44519")

	require.True(t, info.Found)
	assert.Equal(t, "A44519", info.StudentID)
	assert.Equal(t, "ĐỖ DUY KHÁNH", info.StudentName)
	assert.Equal(t, "Khoa Công nghệ thông tin", info.Faculty)
	assert.Equal(t, "9.450.000", info.TotalDebt)

	require.Len(t, info.UnpaidBills, 1)
	assert.Equal(t, "27905122", info.UnpaidBills[0].Code)
	assert.Equal(t, "Học phí HK01 - 2026-2027", info.UnpaidBills[0].Name)
	assert.Equal(t, "[Hoc phi]", info.UnpaidBills[0].FeeType)

	require.Len(t, info.RecentInvoices, 1)
	assert.Equal(t, "4744643049", info.RecentInvoices[0].InvoiceCode)
	assert.Equal(t, "Đã thanh toán", info.RecentInvoices[0].Status)

	formatted := FormatPromptContext(info)
	assert.Contains(t, formatted, "ĐỖ DUY KHÁNH")
	assert.Contains(t, formatted, "9.450.000")
	assert.Contains(t, formatted, "Học phí HK01 - 2026-2027")
}

func TestLookupTuition_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pay/thanglong", r.URL.Path)
		customer := r.URL.Query().Get("customer")
		if customer == "A44519" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(sampleHTML))
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><body><input id="hoten" value=""/></body></html>`))
		}
	}))
	defer server.Close()

	client := NewClient(5*time.Second, server.URL+"/pay/thanglong")
	info, err := client.LookupTuition(context.Background(), "A44519")
	require.NoError(t, err)
	assert.True(t, info.Found)
	assert.Equal(t, "ĐỖ DUY KHÁNH", info.StudentName)

	notFoundInfo, err := client.LookupTuition(context.Background(), "A99999")
	require.NoError(t, err)
	assert.False(t, notFoundInfo.Found)
}

func TestLookupTuition_Live(t *testing.T) {
	client := NewClient(5 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := client.LookupTuition(ctx, "A44519")
	if err != nil {
		t.Skipf("Skipping live tuition test due to network: %v", err)
	}
	require.True(t, info.Found)
	assert.Equal(t, "ĐỖ DUY KHÁNH", info.StudentName)
	assert.Equal(t, "Khoa Công nghệ thông tin", info.Faculty)
	assert.NotEmpty(t, info.TotalDebt)
}
