---
id: inquiry_escalation
name: Chuyển tiếp Thắc mắc & Khiếu nại
description: Nhận diện và đóng gói các tình huống học vụ phức tạp, khiếu nại điểm để chuyển tiếp đến Giảng viên / Cố vấn chuyên trách.
priority: 20
tool_definition: |
  {"name":"escalate_inquiry","description":"Chuyển tiếp câu hỏi, thắc mắc vượt thẩm quyền hoặc khiếu nại điểm cho Cố vấn học tập / Phòng Đào tạo người thật giải quyết.","parameters":{"type":"object","properties":{"reason":{"type":"string","description":"Lý do cần chuyển tiếp thắc mắc."}}}}
---
# Kỹ Năng: Chuyển Tiếp Thắc Mắc & Khiếu Nại (inquiry_escalation)

## Mục Đích

Kỹ năng này chịu trách nhiệm nhận diện các thắc mắc học vụ vượt quá thẩm quyền của AI hoặc thiếu dữ liệu văn bản chính thức để đề xuất chuyển tiếp cho Giảng viên / Cố vấn học vụ người thật giải quyết.

## Hướng Dẫn Thực Hiện

1. **Dấu Hiệu Kích Hoạt Chuyển Tiếp (Escalation Triggers)**:
   - Sinh viên khiếu nại về điểm thi học phần, chấm phúc khảo chưa có kết quả sau thời hạn.
   - Các trường hợp hoàn cảnh đặc biệt xin hoãn thi, xin bảo lưu ngoài thời hạn do ốm đau, tai nạn cần xét duyệt của Hội đồng.
   - Thắc mắc về việc chuyển ngành, học cùng lúc hai chương trình (song bằng) cần phê duyệt chỉ tiêu từ Khoa và Phòng Đào tạo.
   - Câu hỏi chi tiết mà cơ sở dữ liệu quy chế và thông báo chưa có thông tin rõ ràng.
2. **Quy Trình Xử Lý**:
   - Tóm tắt lại vấn đề cốt lõi của sinh viên một cách ngắn gọn, súc tích.
   - Đề xuất chuyển tiếp yêu cầu đến Cố vấn học vụ phụ trách chuyên ngành hoặc Phòng Đào tạo.
   - Hướng dẫn sinh viên chuẩn bị sẵn các minh chứng cần thiết (đơn viết tay, bệnh án, bảng điểm tạm thời...).
