# TLUAgent - Hệ thống AI cố vấn học vụ thông minh và điều hành học vụ TLU

> **Đồ án tốt nghiệp** ngành Công nghệ thông tin — Trường Đại học Thăng Long (TLU).

Hệ thống AI cố vấn học vụ và nền tảng hỗ trợ điều hành học vụ dành cho sinh viên và ban quản lý học vụ Trường Đại học Thăng Long. Hệ thống tích hợp mô hình ngôn ngữ lớn (Multimodal LLM), động cơ tìm kiếm **RAG Hybrid**, **Đồ thị tri thức (Knowledge Graph)** tự động loại trừ văn bản hết hiệu lực, bộ rà soát bãi bỏ quy chế **Regulatory Dissection Engine**, phân hệ **TLUClaw** quản trị học vụ bằng ngôn ngữ tự nhiên, pipeline **Crawler & Job Scheduler** tự động hóa dữ liệu từ cổng thông tin trường, và **Cổng lộ trình học tập cá nhân hóa**.

---

## 1. Mục đích và phạm vi đồ án

- **Chuẩn hóa tri thức học vụ phân mảnh**: Tích hợp dữ liệu từ 16 khóa đào tạo (K24 - K39), hàng trăm quyết định, thông báo, quy chế đào tạo tín chỉ và sổ tay sinh viên vào một cơ sở dữ liệu đồng nhất.
- **Triệt tiêu nguy cơ áp dụng quy chế cũ (Zero Hallucination & Lex Posterior)**: Tự động phát hiện và loại bỏ các văn bản đã bị sửa đổi hoặc thay thế (`SUPERSEDED`), đảm bảo sinh viên và cố vấn luôn tiếp cận thông tin có hiệu lực pháp lý cao nhất theo thứ tự thời gian ban hành.
- **Tự động hóa thu thập dữ liệu học vụ**: Pipeline crawler tự động thu thập thông báo, tin tức và sổ tay sinh viên từ cổng thông tin đào tạo của nhà trường, tự bóc tách tệp PDF đính kèm/Google Drive thuần trong bộ nhớ RAM và vector hóa tức thì.
- **Cá nhân hóa lộ trình học tập theo thời gian thực**: Phân tích bảng điểm cá nhân, tính toán đồ thị môn tiên quyết (DAG) và gợi ý phân bổ môn học từng kỳ nhằm tối ưu thời gian tốt nghiệp (3.5 - 4.5 năm).
- **Trợ lý điều hành học vụ TLUClaw**: Hỗ trợ quản trị viên bóc tách tài liệu chương trình đào tạo đa định dạng (Excel 25+ sheet, Word, PDF, CSV), so sánh sai khác (Diff), tính toán phạm vi ảnh hưởng (Blast Radius), thực hiện cập nhật theo quy trình 2 pha an toàn có thể hoàn tác (Rollback).
- **Đóng gói độc lập (Single-Binary)**: Đóng gói toàn bộ backend Go, frontend React 19 và CSDL SQLite WAL trong một tệp thực thi duy nhất, không phụ thuộc môi trường ngoài.

---

## 2. Hệ thống tính năng

### 2.1. Phân hệ cố vấn học vụ và sinh viên
- **Hội thoại thời gian thực và suy luận minh bạch (SSE streaming)**:
  - Phản hồi văn bản tức thì theo luồng sự kiện (SSE), hỗ trợ ngắt luồng sớm.
  - Tách bạch và hiển thị khối suy luận tư duy (Reasoning/Thinking trace) của các mô hình tiên tiến (DeepSeek-R1, Qwen, Gemini).
  - Trích dẫn chính xác số hiệu văn bản, ngày ban hành, điều khoản và trích đoạn quy chế gốc.
- **Giao thức phản biện Socratic (làm rõ khóa và ngành)**:
  - Tuyệt đối không phỏng đoán khóa học (Kxx) hay ngành học khi tư vấn chương trình đào tạo, môn thay thế, chuẩn đầu ra hay học phí.
  - Tự động đối chiếu 3 tầng bộ nhớ: Hồ sơ sinh viên → Ký ức dài hạn → Lịch sử hội thoại. Nếu thiếu thông tin, AI chủ động hỏi lại sinh viên trước khi đưa ra câu trả lời.
- **Bóc tách bảng điểm qua ảnh (Multimodal LLM & Transcript Diff)**:
  - Cho phép tải lên đồng thời nhiều ảnh chụp màn hình bảng điểm từ cổng thông tin đào tạo (xử lý trực tiếp qua LLM đa phương thức, không cần engine OCR ngoài).
  - Trích xuất điểm tự động, tính GPA/CPA hệ 10 và hệ 4.
  - **Thẻ xác nhận tương tác (Interactive Confirmation Card)**: Đối chiếu điểm mới và điểm cũ, gắn nhãn `NEW` (môn mới), `UPGRADED` (cải thiện), `UNCHANGED` (giữ nguyên); hỗ trợ lưu vào hồ sơ với 1 thao tác hoặc tiếp tục tư vấn.
- **Tư vấn quy chế học vụ đặc thù Thăng Long**:
  - *Chuỗi tiếng Anh bậc thang (`GE111` → `GE333`)*: Bắt buộc học tuần tự, cảnh báo không được học song song các cấp độ trong cùng một kỳ.
  - *Cơ chế thác nước 5 tín chỉ tự do*: Tự động gom linh hoạt từ môn ngoại khoa, ngoại ngữ bổ sung, hoặc tín chỉ chuyên ngành học vượt định mức.
  - *Cảnh báo hạ bậc bằng tốt nghiệp*: Theo dõi ngưỡng an toàn 5% tín chỉ thi lại (~6.5 tín chỉ đối với chương trình 130 tín chỉ) đối với sinh viên có học lực khá, giỏi, xuất sắc.
  - *Quy chế thi nâng điểm*: Kiểm tra điều kiện môn điểm 4.0 - 5.4 (D/D+) và giới hạn 02 học kỳ liên tiếp.
  - *Môn học thay thế*: Tra cứu quan hệ thay thế 2 chiều (`A ↔ B`), phi bắc cầu theo khóa (≤ K35 phân theo chuyên ngành; K36–K37 áp dụng toàn khoa CNTT).
- **Chuyển tiếp thắc mắc học vụ (Inquiry escalation)**:
  - Chuyển tiếp các câu hỏi học vụ phức tạp hoặc khiếu nại điểm lên giảng viên / ban cố vấn chuyên trách.
  - Hệ thống thông báo in-app cập nhật tiến độ giải quyết cho sinh viên.
- **Quyền riêng tư và quản lý phiên**:
  - Tách biệt khách (lưu tại trình duyệt) và sinh viên đăng nhập (lưu cơ sở dữ liệu bền vững).
  - Tùy chọn xóa lịch sử trò chuyện và xóa vĩnh viễn ký ức học vụ AI.

---

### 2.2. Cổng lộ trình học tập cá nhân hóa
- **Truy cập tại**: `/student/roadmap`
- **Bảng Kanban tiến độ 8 học kỳ**:
  - Trực quan hóa toàn bộ tiến trình học tập từ kỳ 1 đến kỳ 8.
  - Gắn nhãn trạng thái trực quan:
    - `PASSED` (Xanh lục): Môn đã hoàn thành và tích lũy.
    - `FAILED_RETAKE` (Đỏ): Môn trượt F cần ưu tiên học lại ngay ở kỳ kế tiếp.
    - `UNLOCKED` (Vàng): Môn đã đủ điều kiện tiên quyết, sẵn sàng đăng ký.
    - `LOCKED` (Xám): Môn chưa hoàn thành các điều kiện tiên quyết.
    - `CRITICAL PATH` (Biểu tượng ngọn lửa): Môn nằm trên đường găng giải phóng chuỗi môn học sau.
- **Khối thẻ KPI học tập**:
  - Thanh tiến độ tích lũy tín chỉ trực quan (số tín chỉ đạt / số tín chỉ chuẩn).
  - Điểm trung bình tích lũy CPA hệ 10 và hệ 4.
  - Tổng số môn nợ cần trả và số môn đã mở khóa.
- **Tra cứu điều kiện tiên quyết**:
  - Xem thông tin chi tiết từng môn học: số tín chỉ, hệ số, số giờ lý thuyết/thực hành, danh sách môn tiên quyết bắt buộc và danh sách các môn tiếp theo sẽ được mở khóa.

---

### 2.3. Phân hệ quản trị TLUClaw
- **Truy cập tại**: `/admin/tluclaw`
- **Điều hành học vụ bằng ngôn ngữ tự nhiên**:
  - Quản trị viên tương tác trực tiếp: *"Thêm môn học mới IT499 cho K39 CNTT"* hoặc đính kèm tài liệu chương trình đào tạo mới.
- **Bộ phân tích tài liệu đa định dạng (`pkg/docparser`)**:
  - Bóc tách danh sách môn học từ tệp Excel (hỗ trợ nhiều sheet, ví dụ 14 sheet CTDT_K39), Word (.docx), PDF và CSV.
- **Quy trình thực thi 2 pha an toàn (Two-Phase execution)**:
  - **Pha 1 (Dry-run & Preview)**: Trích xuất dữ liệu, lập bảng đối chiếu Diff (môn thêm mới, sửa tín chỉ/tiên quyết, xóa môn) và đo lường phạm vi tác động (**Blast Radius**: khóa học, ngành học bị ảnh hưởng).
  - **Pha 2 (Commit with cryptographic nonce)**: Chỉ khi quản trị viên xác nhận trên thẻ Preview, giao dịch nguyên tử mới được ghi vào cơ sở dữ liệu.
- **Tự động cắt đoạn và tạo vector embedding**:
  - Khi thay thế quy chế, hệ thống tự động đánh dấu văn bản cũ là `SUPERSEDED`, ghi nhận nhật ký `supersession_audit_log`.
  - Văn bản mới được tự động cắt đoạn (kích thước ~1200 ký tự, trùng lặp 150 ký tự), sinh vector embedding và cập nhật đồ thị tri thức (`SUPERSEDES`, `APPLIES_TO_COHORT`).
- **Phát sóng thông báo tự động**:
  - Tự động tạo bản ghi thông báo gửi đến sinh viên thuộc các khóa/ngành nằm trong phạm vi ảnh hưởng (Blast Radius).
- **Lịch sử cập nhật và hoàn tác 1-click (True rollback)**:
  - Lưu trữ lịch sử các đợt cập nhật (Snapshot) kèm trạng thái (`PENDING_CONFIRMATION`, `COMMITTED`, `CANCELLED`, `ROLLED_BACK`).
  - Cho phép quản trị viên hoàn tác 1-click về trạng thái ban đầu, tự động khôi phục cấu trúc môn học và xóa sạch bộ nhớ cache RAM.

---

### 2.4. Quản trị khung chương trình đào tạo
- **Truy cập tại**: `/admin/curriculum`
- **Danh mục môn học đa khóa**:
  - Tra cứu và quản lý hơn 16.000 lượt môn học thuộc 16 khóa (K24 - K39).
  - Lọc đa chiều theo khóa, ngành, khối kiến thức (đại cương, cơ sở ngành, chuyên ngành), học kỳ và từ khóa.
- **Sơ đồ lộ trình và mạng lưới môn tiên quyết**:
  - Phân bổ tỷ trọng tín chỉ theo các khối kiến thức chuẩn kiểm định.
  - Bảng Kanban môn học theo học kỳ dự kiến.
  - Sơ đồ DAG đồ thị liên kết trực quan giữa các môn học.
- **Quản lý danh mục môn thay thế**:
  - Thiết lập và rà soát các cặp môn thay thế tương đương theo phạm vi khóa và ngành.

---

### 2.5. Hệ thống thu thập dữ liệu học vụ tự động
- **Truy cập tại**: `/admin/jobs`
- **Bộ thu thập dữ liệu tự động (`pkg/crawler`)**:
  - Thu thập trực tiếp từ cổng thông tin đào tạo của nhà trường cho 3 phân hệ:
    - **Thông báo (`/thong-bao`)**: 378 trang với 2,268 thông báo học vụ (6 bài/trang).
    - **Tin tức (`/tin-tuc`)**: 119 trang với 474 bài tin tức (4 bài/trang).
    - **Trang tĩnh (`TLUStaticRoutes`)**: 25 trang tĩnh sổ tay sinh viên, hướng dẫn học vụ, chuẩn đầu ra ngoại ngữ/tin học.
- **Cơ chế cào đảo chiều theo thời gian (`reverse = true`)**:
  - Nạp tài liệu theo thứ tự thời gian từ cũ nhất đến mới nhất (`oldest` → `newest`).
  - Bảo đảm tính nhân quả thời gian, làm tiền đề cho bộ phát hiện bãi bỏ quy chế hoạt động chính xác.
- **Cắt mốc thời gian tăng dần (`max_time_days`)**:
  - Tự động phát hiện bài mốc trong quá khứ (mốc thời gian `t < now - max_time_days`) và cào tiến dần về hiện tại, dừng sớm (early exit) giúp các lần chạy định kỳ hoàn thành trong vài giây.
- **Bộ lọc định danh RAM siêu tốc (O(1) Deduplication)**:
  - Tải sẵn toàn bộ định danh văn bản trong cơ sở dữ liệu vào RAM khi bắt đầu chạy.
  - So khớp ngay tại trang danh sách phân trang theo ID, Node ID, URL và tiêu đề; bỏ qua tức thì các bài đã có mà không gửi yêu cầu mạng hay truy vấn cơ sở dữ liệu.
- **Trích xuất thuần trong RAM và tải PDF tự động (`pkg/pipeline`)**:
  - Chuyển đổi nội dung HTML sang định dạng Markdown chuẩn hóa.
  - Tự động nhận diện và tải trực tiếp các tệp PDF đính kèm từ máy chủ trường hoặc Google Drive.
  - Trích xuất văn bản PDF bằng thư viện Pure Go (`pipeline.ExtractPDFBytesPure`), không phụ thuộc công cụ bên ngoài.
- **Hệ thống lập lịch định kỳ (`job_schedules`)**:
  - Quản trị viên theo dõi trạng thái, tiến trình (`progress`/`total`), xem nhật ký lỗi và kích hoạt thủ công.
  - Lịch trình tự động chạy ngầm:
    - `sched-crawl-notices`: Tự động cào thông báo mới mỗi 2 giờ.
    - `sched-crawl-news`: Tự động cào tin tức mới mỗi 4 giờ.
    - `sched-crawl-static`: Cập nhật trang tĩnh mỗi 24 giờ.
    - `sched-supersede-check`: Tự động rà soát bãi bỏ mỗi 3 giờ.

---

### 2.6. Động cơ RAG đồ thị và rà soát bãi bỏ quy chế
- **Tìm kiếm lai (Hybrid search)**:
  - Kết hợp đồng thời Vector Search Cosine (60%) + SQLite FTS5 BM25 (30%) + Cấp bậc văn bản (10%).
  - Lọc bỏ văn bản đã bị thay thế thông qua giải thuật đệ quy trên đồ thị tri thức.
- **Bộ rà soát bãi bỏ quy chế (Regulatory Dissection Loop)**:
  - Phân tích chi tiết từng điều khoản văn bản học vụ, trích xuất căn cứ pháp lý ("Bãi bỏ Quyết định số...", "Thay thế Quy chế...").
  - Tìm kiếm văn bản ứng viên bị bãi bỏ bằng FTS5 BM25 kết hợp Vector Search và so khớp tiêu đề.
  - Đánh dấu trạng thái `SUPERSEDED`, ghi nhận liên kết `SUPERSEDES` trên đồ thị tri thức và lưu nhật ký `supersession_audit_log`.
- **Bộ nhớ kế thừa quan hệ bãi bỏ (Supersession lineage)**:
  - Xử lý quan hệ bắc cầu bãi bỏ: Nếu A → B và B → C thì A tự động thay thế cả C.
- **Bộ nén ngữ cảnh tự động (`pkg/compactor`)**:
  - Tự động nén và tóm tắt lịch sử hội thoại nhiều bước của Agent, bảo đảm không bị tràn cửa sổ ngữ cảnh (Context Window) khi rà soát kho văn bản lớn.
- **Rà soát theo dòng thời gian ban hành (`AuditCorpusChronological`)**:
  - Duyệt toàn bộ văn bản theo `publish_date ASC` để tuân thủ nguyên tắc pháp lý Lex Posterior (văn bản ban hành sau có giá trị thay thế văn bản trước).
- **Quản lý đa mô hình LLM (`/admin/llm`)**:
  - Hỗ trợ các nhà cung cấp OpenAI, Google Gemini, Ollama, vLLM, DeepSeek.
  - Hỗ trợ mô hình embedding cục bộ qua **ONNX Runtime** (`bge-m3`) kết hợp Cloud API với cơ chế tự động chuyển vùng dự phòng (Fallback).
  - Mã hóa khóa API an toàn tại chỗ bằng chuẩn AES-256-GCM.
- **Trực quan hóa đồ thị tri thức (`/admin/knowledge-graph`)**:
  - Giao diện Canvas SVG hiển thị các nút (Khóa học, Chủ đề, Tài liệu, Môn học) và các cạnh liên kết (`SUPERSEDES`, `APPLIES_TO_COHORT`, `REQUIRES`, `ACCOMPANIES`).

---

### 2.7. Bảo mật và kiến trúc phần mềm
- **Cách ly thẻ bằng nonce động (Delimiter isolation)**:
  - Dữ liệu người dùng, trích xuất RAG và ký ức được bọc trong các thẻ XML có mã nonce ngẫu nhiên (`<user_query nonce="...">`, `<rag_context nonce="...">`, `<student_memory nonce="...">`).
  - Quy định nội dung bên trong thẻ là dữ liệu thụ động (`untrusted passive data`), ngăn chặn kỹ thuật tấn công Indirect Prompt Injection và Delimiter Spoofing.
- **Phòng thủ Red Team và chuẩn hóa văn bản (Guardrail)**:
  - Chuẩn hóa Unicode NFKD, loại bỏ ký tự zero-width, chuyển đổi ký tự Cyrillic giả mạo (Homoglyphs) và Leetspeak.
  - Đệ quy giải mã đa tầng (Base64, Hex, URL unescape) để phát hiện payload ẩn.
  - Phân loại ngữ nghĩa ngăn chặn bẻ khóa (DAN, Developer Mode, Roleplay).
  - Bộ đệm cửa sổ trượt (Sliding Window Buffer) bảo toàn ranh giới UTF-8 rune và ngăn chặn rò rỉ thẻ hệ thống trong luồng stream SSE.
- **Kiến trúc repository chuẩn mực**:
  - 100% truy vấn CSDL biên dịch qua `sqlc`, tuyệt đối không có SQL inline dạng chuỗi.
  - Sử dụng `singleflight.Group` triệt tiêu hiện tượng Thundering Herd khi truy vấn đồng thời cao.
  - Tích hợp RAM cache (`theine-go`) theo mô hình **Cache IDs Pattern**.
- **Đa ngôn ngữ song ngữ (i18n)**:
  - Giao diện hỗ trợ song ngữ tiếng Việt và tiếng Anh (`public/locales/vi.json` và `en.json`).
  - Tuyệt đối không để chuỗi giao diện bị hardcode trong mã nguồn frontend.
- **Thiết kế theo nhận diện thương hiệu TLU**:
  - Màu sắc chuẩn thương hiệu: Navy (`#050C56`), Navy Light (`#0D1B7A`), Blue (`#1D53E6`), Red (`#EA2212`), Cyan (`#3AC2D8`).
  - Giao diện hiện đại, rõ ràng, phù hợp môi trường học thuật.

---

## 3. Cấu trúc mã nguồn

```text
web/
├── cmd/
│   ├── server/             # Điểm khởi chạy máy chủ chính & nhúng file tĩnh Single-Binary
│   ├── stress_test/        # Bộ kiểm thử tải đồng thời (50 - 100 workers)
│   ├── ingest_curriculum/  # Công cụ nạp dữ liệu CTĐT K24 - K39
│   └── test_edge_cases/    # Kịch bản kiểm thử các trường hợp biên học vụ
├── db/
│   ├── schema/             # Các tệp lược đồ DDL SQLite (documents, jobs, curriculum...)
│   └── query/              # Các tệp truy vấn SQL dùng cho sqlc generate
├── frontend/               # Ứng dụng SPA React 19 + TypeScript + Tailwind CSS v4
│   ├── src/
│   │   ├── components/     # Các thành phần UI dùng chung, Layout, Thẻ TLUClaw, Thẻ Bảng điểm
│   │   ├── pages/
│   │   │   ├── admin/      # Quản trị: TLUClaw, Khung CTĐT, Jobs & Scheduler, RAG Documents, LLM, KG...
│   │   │   └── student/    # Sinh viên: Lộ trình cá nhân hóa (Roadmap), Chat cố vấn
│   │   ├── services/       # Lớp gọi API backend chuẩn Type-safe
│   │   ├── stores/         # Quản lý trạng thái ứng dụng (Zustand)
│   │   └── types/          # Định nghĩa kiểu dữ liệu TypeScript
│   └── public/locales/     # Bộ từ điển song ngữ vi.json và en.json (100% Parity)
├── internal/
│   ├── controllers/        # Bộ điều hướng HTTP RESTful API & SSE (Job, Advisory, Auth, TLUClaw...)
│   ├── services/           # Nghiệp vụ Cố vấn, TLUClaw, Jobs, Lập lịch, Lộ trình DAG, Dissection Engine
│   ├── repositories/       # Truy vấn dữ liệu qua sqlc + singleflight + RAM cache (Cache IDs Pattern)
│   ├── middlewares/        # JWT, RBAC, CSRF, Rate Limiting, Guardrail, Safe CORS
│   ├── agent/              # Hạt nhân Agent, Quy tắc học vụ (rule.md, soul.md), Bộ kỹ năng
│   └── gen/sqlc/           # Mã nguồn Go được tự động sinh bởi sqlc (Zero raw SQL)
└── pkg/
    ├── cache/              # Bộ nhớ đệm RAM tốc độ cao (theine-go)
    ├── compactor/          # Bộ nén và tóm tắt ngữ cảnh Agent đa bước
    ├── crawler/            # Bộ cào dữ liệu thông báo, tin tức, trang tĩnh đảo chiều theo thời gian
    ├── docparser/          # Bộ bóc tách đa định dạng (Excel 25+ sheets, Word, PDF, CSV)
    ├── embedding/          # Động cơ vector embedding (ONNX Runtime On-Premise & Cloud API)
    ├── guardrail/          # Bộ lọc an toàn, chuẩn hóa Unicode, giải mã đa tầng, chống Jailbreak
    ├── llm/                # Trình quản lý kết nối đa mô hình LLM & cơ chế tự phục hồi
    ├── netx/               # Lớp mạng an toàn chống SSRF và DNS Rebinding
    ├── pipeline/           # Chuyển đổi HTML sang Markdown & Trích xuất PDF Pure-Go
    └── supersede/          # Bộ phân tích văn bản bãi bỏ và đồ thị quan hệ kế thừa
```

---

## 4. Hướng dẫn cài đặt và khởi chạy

### Yêu cầu môi trường
- **Go**: Phiên bản 1.23 trở lên
- **Node.js**: Phiên bản 20+ hoặc **Bun**
- **Sqlc**: Dùng để biên dịch các câu truy vấn SQL

### 4.1. Thiết lập môi trường
Sao chép tệp cấu hình mẫu và chỉnh sửa các biến môi trường:
```bash
cp .env.example .env
```

### 4.2. Các lệnh thực thi chính

| Lệnh | Mục đích |
| :--- | :--- |
| `make dev` | Khởi chạy máy chủ phát triển (Backend Go + Frontend Vite) |
| `make test` | Chạy toàn bộ các bộ kiểm thử tự động (Unit, Integration, Security, Red Team) |
| `make build` | Biên dịch toàn bộ hệ thống thành tệp nhị phân duy nhất (`bin/server`) |
| `make check` | Chạy bộ kiểm tra chất lượng: sqlc, typecheck, oxlint, kiểm tra secrets |
| `make run-stress` | Chạy kịch bản kiểm thử tải áp lực 50–100 workers đồng thời |

### 4.3. Kiểm thử trực tiếp

```bash
# 1. Kiểm tra biên dịch truy vấn sqlc
sqlc generate

# 2. Chạy kiểm thử toàn diện Backend
go test -count=1 ./...

# 3. Kiểm tra cú pháp và linter Frontend
cd frontend && bun run check

# 4. Đóng gói sản phẩm Frontend
cd frontend && bun run build

# 5. Biên dịch Single-Binary Server
go build -o ../bin/server ./cmd/server
```

---

## 5. Thông tin đồ án và bản quyền

* **Tên đồ án**: Nghiên cứu, xây dựng hệ thống AI Cố vấn học vụ và Điều hành học vụ thông minh cho Trường Đại học Thăng Long.
* **Tính chất dự án**: Đây là sản phẩm **Đồ án tốt nghiệp đại học**, phục vụ mục đích nghiên cứu, học thuật và thử nghiệm ứng dụng thực tiễn trong công tác hỗ trợ sinh viên và quản trị học vụ.
