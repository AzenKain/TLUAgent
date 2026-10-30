---
id: tuition_calculation
name: Tính toán Học phí & Hạn nộp
description: Tính toán học phí theo tín chỉ, tra cứu trực tiếp tại cổng {{tuition_portal_url}} và hướng dẫn xử lý nợ học phí.
priority: 60
tool_definition: |
  {"name":"lookup_tuition_fee","description":"Tra cứu học phí thực tế, nợ học phí và tình trạng thanh toán của sinh viên TLU từ cổng thanh toán học phí của Trường theo Mã sinh viên","parameters":{"type":"object","properties":{"student_id":{"type":"string","description":"Mã sinh viên TLU (ví dụ: A44519)"}},"required":["student_id"]}}
---
# Kỹ Năng: Tính Toán Học Phí, Tra Cứu Cổng Thanh Toán & Hướng Dẫn Xử Lý Nợ Học Phí (tuition_calculation)

## Mục Đích

Kỹ năng này chịu trách nhiệm:

1. Hướng dẫn kênh thanh toán học phí trực tuyến chính thức của Trường Đại học Thăng Long.
2. Tra cứu trực tiếp tình trạng học phí, số tiền nợ học phí và hóa đơn của sinh viên qua công cụ tra cứu cổng học phí.
3. Hướng dẫn chi tiết quy trình xử lý khẩn cấp khi sinh viên bị quá hạn nộp học phí để tránh bị cấm thi và khóa đăng ký học phần.

## Hướng Dẫn Thực Hiện

### 1. Kênh Thanh Toán Trực Tuyến Chính Thức

- **Cổng thanh toán chính thức duy nhất**: [{{tuition_portal_url}}]({{tuition_portal_url}}) (Hệ thống thanh toán trực tuyến của Trường Đại học Thăng Long).
- **Cách tra cứu & nộp**: Sinh viên truy cập đường link trên, nhập **Mã sinh viên** (ví dụ: `A44444`) để xem chi tiết học phí học kỳ, tổng số tiền cần thanh toán và chọn hình thức thanh toán (thẻ ATM nội địa, thẻ Visa/MasterCard, Ví MoMo, VNPay-QR).
- **Cảnh báo an toàn**: Tuyệt đối không chuyển khoản qua tài khoản cá nhân, tài khoản trung gian hoặc các đường link lạ không thuộc tên miền chính thức của Trường (`{{university_site_url}}`).

### 2. Sử Dụng Công Cụ Tra Cứu Học Phí (lookup_tuition_fee)

- Khi sinh viên cung cấp Mã sinh viên (ví dụ: `A44519`, `A41234`...) và hỏi về tình hình học phí, nợ học phí, hệ thống sẽ kích hoạt công cụ `lookup_tuition_fee` để trích xuất dữ liệu trực tiếp từ cổng thanh toán.
- Báo cáo rõ ràng cho sinh viên: Họ tên, Khoa, Tổng số tiền nợ học phí, Chi tiết từng khoản phí (tên học phần/học kỳ, loại phí, số tiền) và dẫn link trực tiếp tới trang thanh toán của sinh viên: `{{tuition_portal_url}}?customer={masv}`.

### 3. Quy Trình Khẩn Cấp Khi Quá Hạn Nộp Học Phí (Bắt Buộc Nhắc Nhở)

Khi sinh viên hỏi về việc **nộp học phí muộn / quá hạn nộp học phí / nợ học phí**, Cố vấn AI PHẢI lập tức cảnh báo và hướng dẫn 3 bước quan trọng sau:

1. **Xuống ngay Phòng Tiếp sinh viên**:
   - Sinh viên phải trực tiếp đến Phòng Tiếp sinh viên tại trường để làm đơn xin nộp học phí muộn hoặc xin nộp bổ sung ngay lập tức.
2. **Thời điểm vàng - Nộp càng sớm càng tốt**:
   - Nếu sinh viên đến nộp sớm trước khi danh sách cấm thi được chốt và ban hành, sinh viên **có thể chưa bị cấm thi kết thúc học phần** của học kỳ hiện tại, đồng thời **học kỳ sau vẫn được mở quyền đăng ký học phần** bình thường.
3. **BẮT BUỘC PHẢI XÁC NHẬN GỠ MÃ NỢ HỌC PHÍ**:
   - **LƯU Ý CỰC KỲ QUAN TRỌNG**: Sau khi nộp tiền xong, sinh viên **bắt buộc phải nhắc và yêu cầu cán bộ Phòng Tiếp sinh viên thao tác gỡ "mã nợ học phí"** (cờ chặn nợ học phí) trên hệ thống phần mềm quản lý đào tạo.
   - **Cảnh báo**: Nếu chỉ nộp tiền mà không được gỡ mã nợ trên hệ thống, thì hệ thống vẫn ghi nhận tình trạng nợ và mọi việc nộp tiền ở trên sẽ trở nên **VÔ NGHĨA** (sinh viên vẫn sẽ bị hệ thống tự động cấm thi và khóa đăng ký học phần kỳ kế tiếp).
