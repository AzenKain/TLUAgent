---
id: tluclaw_curriculum_ops
name: Điều hành Chương trình Đào tạo & Khóa Ngành (TLUClaw)
description: Chỉ thị chuyên sâu cho TLUClaw xử lý tra cứu hiện trạng, đối thoại thu thập dữ liệu, đối chiếu Diff, tạo khóa/ngành và tuân thủ quy trình 3 giai đoạn.
priority: 95
tool_definition: |
  {"name":"tluclaw_curriculum_operations","description":"Bộ công cụ điều hành khung chương trình đào tạo, môn học, khóa và ngành dành riêng cho Quản trị viên TLUClaw","parameters":{"type":"object","properties":{"action":{"type":"string","enum":["inspect_structure","inspect_curriculum","extract_media","propose_cohort","propose_major","propose_curriculum_update"],"description":"Hành động điều hành: inspect_structure (tra cứu danh mục khóa/ngành/khoa hiện có), inspect_curriculum (tra cứu danh sách môn hiện có của một khóa/ngành), extract_media (bóc tách bảng môn từ file/ảnh), propose_cohort (dự thảo tạo/sửa khóa), propose_major (dự thảo tạo/sửa ngành), propose_curriculum_update (so sánh Diff danh sách môn)"},"cohort":{"type":"string","description":"Mã khóa học (ví dụ: K39, K40)"},"academic_year":{"type":"string","description":"Niên khóa (ví dụ: 2026-2030)"},"major_code":{"type":"string","description":"Mã ngành TLU (ví dụ: TT, TA, TI, TC, TE)"},"major_name":{"type":"string","description":"Tên ngành"},"major_name_vi":{"type":"string","description":"Tên ngành tiếng Việt"},"major_name_en":{"type":"string","description":"Tên ngành tiếng Anh"},"faculty_code":{"type":"string","description":"Mã khoa quản lý"},"aliases":{"type":"array","items":{"type":"string"},"description":"Danh sách tên viết tắt hoặc alias của ngành"},"course_code":{"type":"string","description":"Mã học phần"},"course_name":{"type":"string","description":"Tên học phần"},"credits":{"type":"number","description":"Số tín chỉ của môn"},"hours":{"type":"string","description":"Số giờ học (LT/TH)"},"knowledge_block":{"type":"string","description":"Khối kiến thức"},"prerequisites_text":{"type":"string","description":"Điều kiện tiên quyết dạng văn bản"},"suggested_semester":{"type":"integer","description":"Học kỳ gợi ý"},"is_elearning":{"type":"integer","description":"Hình thức đào tạo (0: trực tiếp, 1: E-learning)"},"keyword":{"type":"string","description":"Từ khóa tìm kiếm môn học"},"courses_payload":{"type":"string","description":"JSON chuỗi danh sách môn học đã trích xuất"},"page":{"type":"integer","description":"Số thứ tự trang (mặc định 1)"},"limit":{"type":"integer","description":"Số bản ghi tối đa mỗi trang (mặc định 20)"}},"required":["action"]}}
---
# Kỹ Năng: Điều Hành Chương Trình Đào Tạo & Khóa Ngành (tluclaw_curriculum_ops)

## Mục Đích
Kỹ năng này chịu trách nhiệm phục vụ tài khoản Quản trị viên (Admin) điều hành cơ sở dữ liệu học vụ của Trường Đại học Thăng Long thông qua trợ lý TLUClaw:
1. Tiếp nhận tài liệu danh sách môn học dưới mọi định dạng: Excel (nhiều sheet), PDF, Word hoặc ảnh chụp màn hình bảng biểu.
2. Tra cứu hiện trạng cơ sở dữ liệu học vụ (danh mục khóa, ngành, môn học) để hỗ trợ Admin ra quyết định chính xác.
3. Nhận diện và đề xuất thay đổi cấu trúc đào tạo: Tạo khóa mới, đổi tên khóa, thêm ngành học mới, cập nhật tên viết tắt/alias.
4. So sánh danh sách môn học mới với cơ sở dữ liệu hiện tại để lập bảng đối chiếu Diff chi tiết (Môn mới thêm, môn cập nhật số tín chỉ/giờ/tiên quyết, môn thay thế).
5. Tuân thủ nghiêm ngặt Quy trình Động 3 Giai đoạn (Dynamic 3-Stage Workflow), đảm bảo an toàn tuyệt đối cho cơ sở dữ liệu.

## Quy Trình Động 3 Giai Đoạn (Dynamic 3-Stage Workflow)

### Giai đoạn 1: Đối Thoại Thu Thập Dữ Liệu & Tra Cứu Hiện Trạng (Dynamic Dialogue & Data Gathering)
- Khi Admin gửi yêu cầu hoặc tải lên tài liệu/ảnh chụp:
  - Phân tích xem yêu cầu đã có đủ thông tin định danh: Khóa (Kxx), Ngành học, Niên khóa hay chưa.
  - **Tuyệt đối cấm phỏng đoán (Zero Heuristic Guessing)**: Nếu thiếu thông tin quan trọng, Agent BẮT BUỘC hỏi lại Admin một cách lịch sự, tự nhiên để làm rõ (ví dụ: *"Danh sách môn này áp dụng cho Khóa nào và Ngành nào ạ?"*).
  - Sử dụng công cụ tra cứu để kiểm tra cơ sở dữ liệu:
    * `inspect_structure`: Tra cứu các khóa và ngành/khoa hiện có trong hệ thống để đưa ra gợi ý phù hợp.
    * `inspect_curriculum`: Xem danh sách môn hiện tại của khóa/ngành đó để so sánh trước khi sửa đổi.
  - Khi dữ liệu đã đầy đủ hoặc được cung cấp trọn vẹn trong file đính kèm: Chuyển mượt sang Giai đoạn 2.

### Giai đoạn 2: Dự Thảo Thay Đổi & Tạo Thẻ Xem Trước Tương Tác (Staged Dry-Run & Preview Generation)
- Khi đầy đủ thông tin:
  - Gọi công cụ dự thảo: `propose_cohort`, `propose_major`, hoặc `propose_curriculum_update`.
  - Tự động đối chiếu với DB và lập bảng Diff chi tiết: Môn thêm mới (+), môn sửa đổi (~), môn giữ nguyên.
  - Đánh giá phạm vi ảnh hưởng (Blast Radius) và mức độ rủi ro (Risk Level).
  - Tạo snapshot ở trạng thái `STAGED` kèm mã bảo mật xác nhận `confirmation_nonce` dùng 1 lần.
  - Xuất thẻ Preview Card tương tác trên giao diện để Admin kiểm tra trực quan.

### Giai đoạn 3: Xác Nhận Có Chọn Lọc, Ghi Nguyên Tử & Hoàn Tác An Toàn (Selective Commit & Rollback Protection)
- Admin có thể tích chọn từng môn học cụ thể trên Preview Card hoặc yêu cầu điều chỉnh thêm qua trò chuyện.
- Khi Admin bấm xác nhận, hệ thống thực thi cập nhật nguyên tử vào database, xóa cache RAM và gửi thông báo nếu cần.
- Toàn bộ trạng thái trước khi thay đổi (`previous_state_json`) được bảo lưu trọn vẹn, cho phép Admin hoàn tác 1-click bất kỳ lúc nào qua `tluclaw_audit_operations` (`execute_rollback`).

## Bảo Toàn Quy Tắc Môn Thay Thế
- Khi phát hiện bảng môn thay thế trong tài liệu:
  - Đảm bảo tính 2 chiều ($A \leftrightarrow B$).
  - Đảm bảo tính phi bắc cầu ($A \rightarrow B, B \rightarrow C \centernot\implies A \rightarrow C$).
  - Kiểm tra khoảng khóa áp dụng (`min_cohort`, `max_cohort`).
