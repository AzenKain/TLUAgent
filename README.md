# TLUAgent Web Service - AI Cố Vấn Học Vụ Thông Minh (TLU)

Hệ thống AI Cố vấn Học vụ thế hệ mới dành cho sinh viên Đại học Thăng Long, tích hợp mô hình ngôn ngữ lớn (LLM), động cơ tìm kiếm kết hợp **RAG Hybrid** và **Đồ thị Tri thức (Knowledge Graph)** tự động loại bỏ quy chế cũ đã hết hiệu lực.

---

## 1. Mục Đích Dự Án
- **Giải quyết bài toán quy chế phân mảnh**: Khắc phục tình trạng văn bản đào tạo, học phí, tốt nghiệp ban hành rải rác qua nhiều năm khiến sinh viên khó tra cứu.
- **Triệt tiêu rủi ro áp dụng quy chế cũ**: Ngăn chặn tuyệt đối việc sinh viên hoặc AI tra cứu nhầm các quyết định/thông báo đã bị bãi bỏ hoặc sửa đổi bởi văn bản mới hơn.
- **Hỗ trợ sinh viên 24/7**: Cung cấp kênh giải đáp học vụ tức thì, có căn cứ văn bản rõ ràng theo đúng ngành và khóa học.
- **Giảm tải Ban Cố vấn**: Tự động hóa các câu hỏi thường gặp, đồng thời làm cầu nối chuyển tiếp các trường hợp phức tạp đến giảng viên.

---

## 2. Mục Tiêu Dự Án
- **Chính xác tuyệt đối (Zero Hallucination)**: 100% câu trả lời có nguồn trích dẫn quy chế còn hiệu lực; triệt tiêu hoàn toàn văn bản bị thay thế.
- **Cá nhân hóa theo khóa (Cohort-Aware)**: Tự động áp dụng đúng quy chế tương ứng với từng khóa sinh viên (*Lex Specialis*).
- **Minh bạch tư duy**: Trả lời dạng luồng thời gian thực kèm quá trình suy luận (Reasoning/Thinking) và điều khoản căn cứ.
- **Học tập chủ động (Human-in-the-Loop)**: Chuyển tiếp câu hỏi khó cho cố vấn giải đáp; tự động nạp tri thức mới vào hệ thống.
- **Bảo mật & Độc lập (Single-Binary)**: Đóng gói toàn bộ backend, frontend và CSDL vào một file thực thi duy nhất, bảo vệ quyền riêng tư người dùng.

---

## 3. Các Tính Năng Chính

### Phân Hệ Cố Vấn Sinh Viên
- **Chat Real-time (SSE Streaming)**: Phản hồi văn bản tức thì theo luồng, hỗ trợ nút dừng phản hồi sớm.
- **Minh bạch suy luận**: Hiển thị quá trình tư duy (Thinking/Reasoning trace) của mô hình (DeepSeek, Gemini).
- **Trích dẫn căn cứ**: Đi kèm tên văn bản, số hiệu, ngày ban hành và điều khoản chi tiết.
- **Phân tách Guest & User**: Khách chat ẩn danh (lưu tại trình duyệt, không lưu server); sinh viên đăng nhập lưu lịch sử bền vững.
- **Quản lý phiên & Đánh giá**: Tạo/đổi tên/xóa phiên chat; đánh giá Like/Dislike cho từng câu trả lời.
- **Chuyển tiếp thắc mắc (Escalation)**: Gửi câu hỏi khó lên Ban Cố vấn; nhận thông báo in-app khi có phản hồi.
- **Quyền riêng tư (Privacy)**: Tùy chọn xóa trắng thông tin cá nhân và xóa vĩnh viễn ký ức học vụ AI.

### Động Cơ RAG & Đồ Thị Tri Thức
- **RAG Hybrid Search**: Kết hợp Vector Cosine (60%) + SQLite FTS5 BM25 (30%) + Cấp bậc văn bản (10%).
- **Recursive CTE Conflict Resolver**: Thuật toán đệ quy trên đồ thị tri thức loại trừ 100% văn bản bị thay thế (`SUPERSEDES`) trước khi gửi vào LLM.
- **Context Compactor**: Quản lý cửa sổ ngữ cảnh token-aware, trượt và tóm tắt hội thoại dài tự động.
- **Bộ nhớ đa tầng (L1-L2-L3)**: Ngữ cảnh tức thời (L1), Hồ sơ tiến độ sinh viên (L2), Kho tri thức chuẩn đã kiểm duyệt (L3).

### Phân Hệ Quản Trị (Admin & Advisor)
- **Dashboard & Báo cáo**: Thống kê số lượng người dùng, cuộc trò chuyện, tài liệu và chỉ số hài lòng.
- **Quản lý Người dùng & RBAC**: Phân quyền 5 vai trò (`ADMIN`, `ADVISOR`, `STUDENT`, `GUEST`, `BANNED`), Atomic Setup Wizard chống chiếm quyền.
- **Quản lý Vòng đời Quy chế**: Theo dõi trạng thái (`ACTIVE`, `SUPERSEDED`, `ARCHIVED`), phân đoạn Chunks, lưu vết lịch sử thay thế và đồng bộ dữ liệu.
- **Đồ thị Tri thức Trực quan**: Canvas mạng SVG hiển thị quan hệ văn bản (`SUPERSEDES`, `REQUIRES`, `AMENDS`), thêm/xóa liên kết trực quan.
- **Quản lý Đa LLM & Fallback**: Hỗ trợ OpenAI-compatible và Google Gemini, chuỗi dự phòng tự động, mã hóa API Key AES-256-GCM, chỉnh thinking budget.
- **Nhân cách & Kỹ năng Agent**: Tùy biến Prompt persona, bật/tắt kỹ năng chuyên biệt (tra cứu học phí, điều kiện tốt nghiệp...).
- **Kiểm duyệt Chat & Thắc mắc**: Xem lại hội thoại, đo lường KPI, cố vấn trả lời thắc mắc sinh viên để tái nạp tri thức L3.
- **Hàng đợi Nền & Cron Jobs**: Quản lý Worker Pool (Pond v2) chạy cào dữ liệu, xử lý văn bản, lập lịch định kỳ.

### Nền Tảng Kỹ Thuật & Bảo Mật
- **Single-Binary Zero-Dependency**: Frontend React 19 nhúng trực tiếp vào Go binary, CSDL SQLite WAL tích hợp.
- **An ninh cấp doanh nghiệp**: 0 RCE, 0 SQL Injection (100% sqlc), chống SSRF/DNS Rebinding (`pkg/netx`), chống CSRF & Brute-force.
- **Đa ngôn ngữ 100%**: Hỗ trợ đầy đủ song ngữ Tiếng Việt và Tiếng Anh.

---

## 4. Khởi Chạy Nhanh

```bash
make dev       # Chạy môi trường phát triển (Go backend + nhúng frontend)
make test      # Chạy kiểm thử toàn bộ hệ thống
make build     # Biên dịch file nhị phân duy nhất (web/bin/server)
make check     # Chạy bộ 9 bước kiểm tra an ninh và chất lượng mã nguồn
```
