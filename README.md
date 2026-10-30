# TLUAgent - Hệ thống AI cố vấn học vụ thông minh và điều hành học vụ TLU

Hệ thống AI cố vấn học vụ và nền tảng hỗ trợ điều hành học vụ dành cho sinh viên và ban quản lý học vụ Trường Đại học Thăng Long. Hệ thống tích hợp mô hình ngôn ngữ lớn, tìm kiếm **RAG Hybrid**, **Đồ thị tri thức (Knowledge Graph)** tự động loại trừ văn bản hết hiệu lực, bộ rà soát bãi bỏ quy chế **Regulatory Dissection Engine**, phân hệ **TLUClaw** quản trị học vụ bằng ngôn ngữ tự nhiên, pipeline **Crawler & Job Scheduler** tự động hóa dữ liệu từ cổng thông tin trường, và **Cổng lộ trình học tập cá nhân hóa**.

---

## 1. Mục đích và phạm vi đồ án

- **Chuẩn hóa tri thức học vụ phân mảnh**: Tích hợp dữ liệu từ 16 khóa đào tạo (K24 - K39), hàng nghìn lượt môn học, hàng trăm quyết định, thông báo, quy chế đào tạo tín chỉ và sổ tay sinh viên vào một cơ sở dữ liệu đồng nhất.
- **Triệt tiêu nguy cơ áp dụng quy chế cũ (Zero Hallucination & Lex Posterior)**: Tự động phát hiện và loại bỏ các văn bản đã bị sửa đổi hoặc thay thế, đảm bảo sinh viên và cố vấn luôn tiếp cận thông tin có hiệu lực pháp lý cao nhất theo thứ tự thời gian ban hành.
- **Tự động hóa thu thập dữ liệu học vụ**: Pipeline crawler tự động thu thập thông báo, tin tức và sổ tay sinh viên từ cổng thông tin đào tạo của nhà trường, tự bóc tách tệp PDF đính kèm/Google Drive thuần trong bộ nhớ RAM và vector hóa tức thì.
- **Cá nhân hóa lộ trình học tập theo thời gian thực**: Phân tích bảng điểm cá nhân, tính toán đồ thị môn tiên quyết (DAG) và gợi ý phân bổ môn học từng kỳ nhằm tối ưu thời gian tốt nghiệp (3.5 - 4.5 năm).
- **Trợ lý điều hành học vụ TLUClaw**: Hỗ trợ quản trị viên bóc tách tài liệu chương trình đào tạo đa định dạng (Excel, Word, PDF, CSV), so sánh sai khác (Diff), đánh giá phạm vi ảnh hưởng (Blast Radius), thực hiện cập nhật theo quy trình 2 pha an toàn có thể hoàn tác (Rollback).
- **Đóng gói độc lập (Single-Binary)**: Đóng gói toàn bộ backend Go, frontend React 19 và CSDL SQLite trong một tệp thực thi duy nhất, không phụ thuộc môi trường ngoài.

---

## 2. Hệ thống tính năng

### 2.1. Phân hệ cố vấn học vụ và sinh viên

- **Hội thoại thời gian thực và suy luận minh bạch**:
  - Phản hồi văn bản tức thì theo luồng sự kiện (SSE), hỗ trợ ngắt luồng sớm.
  - Tách bạch và hiển thị khối suy luận tư duy (Reasoning/Thinking trace) của các mô hình hỗ trợ.
  - Trả kèm nguồn trích dẫn là các văn bản quy chế đã dùng để trả lời, do trợ lý trình bày theo mẫu "Nguồn trích dẫn" (tên văn bản, số quyết định, điều khoản liên quan).
- **Giao thức phản biện Socratic (làm rõ khóa và ngành)**:
  - Trợ lý không phỏng đoán khóa học (Kxx) hay ngành học khi tư vấn chương trình đào tạo, môn thay thế, chuẩn đầu ra hay học phí.
  - Tự động đối chiếu nhiều tầng ngữ cảnh: hồ sơ sinh viên → ký ức dài hạn → lịch sử hội thoại → kho văn bản. Nếu thiếu thông tin, trợ lý chủ động hỏi lại sinh viên trước khi trả lời.
- **Bóc tách bảng điểm qua ảnh (Multimodal LLM & Transcript Diff)**:
  - Tải lên nhiều ảnh chụp màn hình bảng điểm từ cổng thông tin đào tạo, xử lý trực tiếp qua LLM đa phương thức (không cần engine OCR ngoài).
  - Trích xuất điểm tự động, tính GPA/CPA hệ 10 và hệ 4.
  - **Thẻ xác nhận tương tác**: đối chiếu điểm mới và điểm cũ, gắn nhãn `NEW` (môn mới), `UPGRADED` (cải thiện), `LOWER` (giảm điểm), `UNCHANGED` (giữ nguyên); lưu vào hồ sơ với 1 thao tác hoặc tiếp tục tư vấn.
- **Tư vấn quy chế học vụ đặc thù Thăng Long**:
  - *Chuỗi tiếng Anh bậc thang (`GE111` → `GE333`)*: bắt buộc học tuần tự, cảnh báo không được học song song các cấp độ trong cùng một kỳ.
  - *Cơ chế thác nước 5 tín chỉ tự do*: gom linh hoạt từ môn ngoại khoa và ngoại ngữ bổ sung ngoài cặp chuẩn.
  - *Cảnh báo hạ bậc bằng tốt nghiệp*: theo dõi ngưỡng an toàn 5% tổng số tín chỉ bị thi lại, chỉ áp dụng cho sinh viên có học lực Xuất sắc/Giỏi; các mức học lực khác nhận ghi chú giải thích thay vì cảnh báo.
  - *Quy chế thi nâng điểm*: áp dụng cho môn đạt 4.0 - 5.4 điểm (D/D+), giới hạn trong 02 học kỳ liên tiếp.
  - *Môn học thay thế*: tra cứu quan hệ thay thế tương đương theo khóa và ngành.
- **Chuyển tiếp thắc mắc học vụ**:
  - Chuyển câu hỏi phức tạp hoặc khiếu nại điểm tới cố vấn chuyên trách, theo dõi trạng thái xử lý và nhận thông báo khi có phản hồi.
- **Quyền riêng tư và quản lý phiên**:
  - Khách dùng thử không cần đăng nhập; sinh viên đăng nhập có lịch sử hội thoại lưu lâu dài.
  - Hỗ trợ xóa lịch sử trò chuyện và xóa vĩnh viễn hồ sơ, ký ức học vụ do AI ghi nhớ.

### 2.2. Cổng lộ trình học tập cá nhân hóa

Truy cập tại `/my-roadmap`.

- **Bảng Kanban tiến độ theo học kỳ**:
  - Trực quan hóa tiến trình học tập theo từng học kỳ của chương trình đào tạo.
  - Gắn nhãn trạng thái: `PASSED` (đã tích lũy), `FAILED_RETAKE` (cần học lại ngay), `UNLOCKED` (đủ tiên quyết, có thể đăng ký), `LOCKED` (chưa đủ tiên quyết).
  - Đánh dấu **đường găng (Critical Path)**: các môn nằm trên chuỗi giải phóng môn học phía sau.
- **Khối thẻ KPI học tập**: tín chỉ đã tích lũy, CPA hệ 10 và hệ 4, số môn đã qua, số môn cần học lại và số tín chỉ còn lại.
- **Tra cứu điều kiện tiên quyết**: xem số tín chỉ, khối kiến thức, học kỳ gợi ý, mô tả tiên quyết và danh sách môn tiên quyết còn thiếu của từng môn.

### 2.3. Phân hệ quản trị TLUClaw

Truy cập tại `/admin/tluclaw`.

- **Điều hành học vụ bằng ngôn ngữ tự nhiên**: quản trị viên yêu cầu trực tiếp (ví dụ *"Thêm môn học mới IT499 cho K39 CNTT"*) hoặc đính kèm tài liệu chương trình đào tạo mới; trợ lý thực hiện thay vì soạn thảo thủ công.
- **Bộ phân tích tài liệu đa định dạng**: bóc tách danh sách môn học từ Excel (nhiều sheet), Word, PDF và CSV.
- **Quy trình thực thi 2 pha an toàn**:
  - *Pha 1 (Dry-run)*: lập bảng đối chiếu sai khác (môn thêm mới, thay đổi tín chỉ/tên môn/tiên quyết) và phạm vi ảnh hưởng tới các khóa, ngành liên quan.
  - *Pha 2 (Commit)*: chỉ ghi dữ liệu sau khi quản trị viên xác nhận trên thẻ xem trước, kèm mã xác nhận dùng một lần để chống thao tác giả mạo hoặc lặp lại.
- **Lịch sử cập nhật và hoàn tác**: mỗi đợt cập nhật được lưu thành một bản ghi trạng thái (`STAGED`, `COMMITTED`, `CANCELLED`, `ROLLED_BACK`) và có thể hoàn tác để khôi phục cấu trúc môn học trước đó.

### 2.4. Quản trị khung chương trình đào tạo

Truy cập tại `/admin/curriculums`.

- **Danh mục môn học đa khóa**: tra cứu và quản lý môn học của 16 khóa (K24 - K39), lọc theo khóa, ngành, khối kiến thức, học kỳ gợi ý và từ khóa.
- **Sơ đồ lộ trình và mạng lưới môn tiên quyết**: tỷ trọng tín chỉ theo khối kiến thức, bảng môn học theo học kỳ dự kiến, sơ đồ liên kết môn học - tiên quyết.
- **Quản lý danh mục môn thay thế**: thiết lập và rà soát các cặp môn thay thế tương đương theo phạm vi khóa và ngành.

### 2.5. Hệ thống thu thập dữ liệu học vụ tự động

Truy cập tại `/admin/jobs`.

- **Thu thập dữ liệu từ cổng thông tin trường**: thông báo, tin tức và các trang tĩnh (sổ tay sinh viên, quy chế đào tạo, chương trình đào tạo, thông tin các khoa...).
- **Nạp dữ liệu theo trục thời gian**: tài liệu được xử lý theo thứ tự ban hành từ cũ đến mới, tạo tiền đề cho việc phát hiện văn bản bị thay thế; chế độ giới hạn số ngày giúp các lần chạy định kỳ chỉ xử lý phần mới phát sinh.
- **Chống trùng lặp**: các bài đã có trong hệ thống được nhận diện và bỏ qua ngay từ trang danh mục, không tải lại nội dung.
- **Bóc tách tài liệu trong bộ nhớ**: chuyển nội dung bài viết sang văn bản chuẩn hóa, tự động tải và trích xuất tệp PDF đính kèm (kể cả PDF trên Google Drive); với bản scan không có lớp văn bản, hệ thống dùng LLM thị giác để phiên âm.
- **Lập lịch tự động**: các tác vụ thu thập chạy định kỳ theo lịch cấu hình sẵn (thông báo, tin tức, trang tĩnh); tác vụ rà soát bãi bỏ quy chế được bật thủ công vì tiêu tốn nhiều lượt gọi LLM. Quản trị viên theo dõi tiến trình, nhật ký lỗi và kích hoạt lại từng tác vụ khi cần.
- **Cảnh báo vận hành**: khi một loại tác vụ thất bại hai lần liên tiếp, hệ thống gửi thông báo cho toàn bộ tài khoản quản trị để kiểm tra crawler hoặc nhà cung cấp LLM.
- **Đồ thị tri thức tự phục hồi**: mỗi lượt khởi động và trước mỗi lượt rà soát, hệ thống dựng lại node văn bản, node
  khóa, node chủ đề và các cạnh bãi bỏ còn thiếu từ dữ liệu đã ghi nhận, nên đồ thị không bị lệch khi một lần nạp
  dữ liệu bị gián đoạn.
- **Rà soát bãi bỏ quy chế chạy tiếp được**: mỗi lượt chạy xử lý một lô văn bản rồi lưu con trỏ tiến độ; lượt kế tiếp tự động tiếp tục từ chỗ dừng thay vì xóa toàn bộ trạng thái và làm lại từ đầu.

### 2.6. Động cơ RAG đồ thị và rà soát bãi bỏ quy chế

- **Tìm kiếm lai (Hybrid search)**: kết hợp tìm kiếm ngữ nghĩa bằng vector, tìm kiếm từ khóa trên chỉ mục toàn văn và cấp bậc hiệu lực của văn bản; kết quả được xếp lại bằng mô hình rerank trước khi trả về.
- **Rà soát bãi bỏ quy chế tự động**: phân tích từng điều khoản văn bản học vụ, xác định văn bản bị bãi bỏ hoặc thay thế, đánh dấu trạng thái `SUPERSEDED` / `PARTIALLY_AMENDED`, ghi nhật ký kiểm toán và cập nhật đồ thị tri thức.
- **Bộ nhớ kế thừa quan hệ bãi bỏ**: xử lý quan hệ bắc cầu (nếu A thay thế B và B thay thế C thì A cũng loại trừ C khỏi kết quả tư vấn).
- **Rà soát theo dòng thời gian ban hành**: duyệt văn bản theo ngày ban hành tăng dần để tuân thủ nguyên tắc văn bản sau thay thế văn bản trước.
- **Phân đoạn văn bản có ngữ cảnh chồng lấn**: đoạn văn dài được cắt theo ranh giới câu với phần chồng lấn giữa hai đoạn liền kề, giữ nguyên ngữ nghĩa khi truy xuất.
- **Cào dữ liệu qua CMS**: website trường render phía client nên crawler đọc trực tiếp headless CMS (địa chỉ cấu hình qua `CMS_API_BASE_URL`, địa chỉ website qua `CMS_SITE_URL`): thông báo từ `announcements`, tin/bài từ `posts` và `events`, trang tĩnh từ `pages`; nội dung rich-text được chuyển thành văn bản, thông báo đính kèm Google Drive được tải và trích xuất; các bản dịch `-en` của trang tĩnh bị bỏ qua để kho văn bản chỉ chứa tiếng Việt.
- **Phân cấp hiệu lực mặc định**: văn bản quy chế, quyết định, hướng dẫn, sổ tay, CTĐT được gán mức hiệu lực chính thức, thông báo gán mức hiệu lực theo học kỳ, tin tức và trang tĩnh gán mức hiệu lực hiện hành; mốc hiệu lực lấy theo ngày ban hành ngay từ lúc nạp dữ liệu, với tài liệu chỉ có niên khóa (bộ Hướng dẫn học tập) thì mốc hiệu lực là ngày đầu niên khóa.
- **Chống trùng lặp và loại dữ liệu rỗng khỏi truy xuất**: bản ghi trùng tiêu đề và ngày ban hành (kể cả khác hoa/thường tiếng Việt) được giữ bản đầy đủ nhất và đánh dấu `SUPERSEDED` kèm cạnh đồ thị; tài liệu chỉ còn tiêu đề được giữ lại cho mục đích quản trị nhưng bị loại khỏi kết quả tìm kiếm để không làm nhiễu câu trả lời.
- **Quản lý đa mô hình LLM** (`/admin/llm`): cấu hình nhiều nhà cung cấp (tương thích OpenAI hoặc Google Gemini), chuỗi dự phòng khi nhà cung cấp gặp sự cố, tự kiểm tra kết nối; khóa API được mã hóa trước khi lưu. Embedding có thể chạy cục bộ bằng ONNX Runtime hoặc qua API.
- **Trực quan hóa đồ thị tri thức** (`/admin/knowledge-graph`): xem trực quan quan hệ giữa văn bản, chủ đề và khóa đào tạo, bao gồm các cạnh thay thế/bãi bỏ và phạm vi áp dụng.

### 2.7. Kiến trúc và bảo mật

- **Phân lớp rõ ràng**: routes → controllers → services → repositories, giao diện React tách biệt hoàn toàn với API.
- **Truy vấn an toàn**: truy vấn CSDL được sinh tự động từ tệp SQL (sqlc), hạn chế lỗi SQL injection và lệch cấu trúc dữ liệu.
- **Đóng máy an toàn, dữ liệu di động được**: khi nhận tín hiệu tắt, hệ thống dừng HTTP server và hàng đợi tác vụ nền, đẩy toàn bộ write-ahead log vào tệp CSDL (`wal_checkpoint(TRUNCATE)`), chuyển journal mode về `DELETE` rồi xoá `-wal`/`-shm`; nhờ vậy thư mục `data/` chỉ còn duy nhất tệp `.db` và có thể copy sang Windows/macOS mà không lỗi. Khi khởi động, tệp `-shm` còn sót lại (do lần chạy trước bị kill) được dọn và SQLite tự dựng lại từ WAL.
- **Hiệu năng**: cache RAM dùng chung cho các truy vấn đọc nhiều, chống dồn tải khi nhiều yêu cầu trùng lặp.
- **Chống Prompt Injection**: dữ liệu người dùng, ngữ cảnh RAG và ký ức được bọc trong thẻ có mã ngẫu nhiên và khai báo là dữ liệu thụ động; bộ lọc đầu vào chuẩn hóa Unicode, phát hiện nội dung mã hóa và chặn các kỹ thuật jailbreak phổ biến.
- **Xác thực và phân quyền**: JWT, phân quyền theo vai trò (ADMIN, ADVISOR, STUDENT, GUEST, BANNED) và permission chi tiết, chống CSRF, giới hạn tần suất yêu cầu.
- **Song ngữ Việt - Anh**: toàn bộ giao diện hỗ trợ hai ngôn ngữ, nhận diện thương hiệu theo bộ màu của Trường Đại học Thăng Long.

### 2.8. Thông báo trong hệ thống

- *Thắc mắc học vụ*: sinh viên nhận thông báo khi cố vấn trả lời (`INQUIRY_ANSWERED`) và khi thắc mắc hết hiệu lực do văn bản quy chế mới thay thế (`INQUIRY_EXPIRED`), kèm lý do cụ thể.
- *Cập nhật chương trình đào tạo*: sau khi quản trị viên xác nhận một đợt cập nhật TLUClaw, từng sinh viên thuộc các khóa bị ảnh hưởng nhận thông báo riêng (`CURRICULUM_UPDATED`).
- *Cảnh báo quản trị*: tài khoản quản trị nhận thông báo `JOB_FAILURE_ALERT` khi tác vụ nền thất bại liên tiếp.

---

## 3. Cấu trúc mã nguồn

```text
web/
├── cmd/server/            # Điểm khởi chạy, dựng dependency và nhúng frontend vào binary
├── db/
│   ├── schema/            # Lược đồ CSDL SQLite (auth, LLM, jobs, chat, RAG/KG, CTĐT, bảng điểm...)
│   └── query/             # Truy vấn SQL nguồn cho sqlc
├── frontend/              # SPA React 19 + TypeScript + Vite + Tailwind CSS
│   ├── public/locales/    # Từ điển tiếng Việt và tiếng Anh
│   └── src/
│       ├── components/    # Thành phần dùng chung, layout, thẻ chat, thẻ xem trước TLUClaw
│       ├── pages/         # Trang sinh viên, chat cố vấn và các trang quản trị
│       ├── services/      # Lớp gọi API có kiểu dữ liệu
│       ├── stores/        # Quản lý trạng thái (Zustand)
│       └── types/         # Định nghĩa kiểu dữ liệu dùng chung
├── internal/
│   ├── agent/             # Prompt hệ thống, bộ kỹ năng nghiệp vụ của trợ lý
│   ├── controllers/       # Tiếp nhận HTTP request (bao gồm stream SSE)
│   ├── middlewares/       # Xác thực, phân quyền, bảo mật, giới hạn tần suất
│   ├── repositories/      # Truy cập dữ liệu và cache
│   ├── routes/            # Khai báo endpoint theo phân hệ
│   └── services/          # Nghiệp vụ: tư vấn, RAG, rà soát quy chế, lộ trình, TLUClaw, tác vụ nền
└── pkg/
    ├── crawler/           # Thu thập dữ liệu và xử lý tệp đính kèm
    ├── pipeline/          # Chuẩn hóa nội dung, trích xuất PDF
    ├── docparser/         # Đọc Excel, Word, PDF, CSV
    ├── embedding/         # Vector hóa văn bản (ONNX Runtime)
    ├── llm/               # Kết nối đa nhà cung cấp, dự phòng, rerank
    ├── guardrail/         # Lọc và chống Prompt Injection
    ├── supersede/         # Phân đoạn văn bản quy chế
    └── ...                # cache, config, crypto, database, logging, mailer, validator, worker
```

---

## 4. Hướng dẫn cài đặt và khởi chạy

**Yêu cầu**: Go 1.27.2 trở lên, Bun 1.x (hoặc Node 20+); sqlc nếu cần sửa tầng dữ liệu, Docker nếu triển khai bằng container.

```bash
cp .env.example .env                          # cổng, thư mục dữ liệu, CORS, token setup

go test -count=1 ./...                        # kiểm thử backend
(cd frontend && bun install && bun run check) # kiểm tra kiểu và lint frontend
(cd frontend && bun run build)                # đóng gói giao diện (bắt buộc trước bước dưới)
go build -trimpath -ldflags="-s -w" -o bin/server ./cmd/server
```

- Chạy khi phát triển: `go run ./cmd/server` (backend) và `cd frontend && bun run dev` (giao diện).
- Triển khai bằng Docker: `docker compose up -d` (compose đọc trực tiếp `.env`, cổng publish ra host lấy từ `APP_PORT`, container luôn lắng nghe ở `3434`).
- Lần chạy đầu trên thư mục dữ liệu trống sẽ tự tải ONNX Runtime và model embedding (`bge-m3`, khoảng 590 MB) về `data/onnx`; đặt `EMBEDDING_AUTO_DOWNLOAD=0` nếu muốn tự tải thủ công trong `/admin/embedding`.
- Sau khi sửa truy vấn SQL: `sqlc generate` và commit cả `internal/gen/sqlc`.

### 4.1. Biến môi trường chính

| Biến | Ý nghĩa |
| --- | --- |
| `HOST`, `PORT` | Địa chỉ và cổng lắng nghe của backend (`go run` dùng trực tiếp; trong container cổng luôn là `3434`). |
| `APP_PORT` | Cổng publish ra host khi chạy bằng docker compose. |
| `DATA_DIR`, `SQLITE_DB_PATH` | Thư mục dữ liệu (CSDL, khóa mã hóa, model) và đường dẫn tệp SQLite. |
| `CORS_ALLOWED_ORIGINS`, `SETUP_TOKEN` | Nguồn giao diện được phép gọi API và token khởi tạo quản trị viên. |
| `CMS_API_BASE_URL` | Địa chỉ headless CMS dùng cho crawler (`announcements`, `posts`, `events`, `pages`). Bắt buộc khi chạy tác vụ cào dữ liệu. |
| `CMS_SITE_URL` | Địa chỉ website trường, dùng để dựng URL chuẩn của văn bản và gốc tải tài liệu đính kèm. |
| `CMS_LOCALE` | Ngôn ngữ lấy nội dung từ CMS (mặc định `vi`). |
| `TUITION_PORTAL_URL` | Địa chỉ cổng thanh toán học phí dùng cho công cụ tra cứu học phí. |
| `EMBEDDING_AUTO_DOWNLOAD` | `0` để tắt tự động tải ONNX Runtime và model embedding khi khởi chạy lần đầu. |

---

## 5. Thông tin đồ án và bản quyền

- **Tên đồ án**: Nghiên cứu, xây dựng hệ thống AI Cố vấn học vụ và Điều hành học vụ thông minh cho Trường Đại học Thăng Long.
- **Tính chất dự án**: Đây là sản phẩm **Đồ án tốt nghiệp đại học**, phục vụ mục đích nghiên cứu, học thuật và thử nghiệm ứng dụng thực tiễn trong công tác hỗ trợ sinh viên và quản trị học vụ.
