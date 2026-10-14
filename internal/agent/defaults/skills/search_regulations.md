---
id: search_regulations
name: Tra cứu Quy chế & Văn bản Học vụ
description: Tra cứu, diễn giải điều khoản quy chế đào tạo, chuẩn đầu ra ngoại ngữ/tin học, học phí và thông báo học vụ của TLU.
priority: 100
tool_definition: |
  {"name":"search_regulations","description":"Tra cứu các văn bản quy chế, quy định học vụ, thông báo, chuẩn đầu ra ngoại ngữ/tin học, học phí, điều kiện xét tốt nghiệp của Trường Đại học Thăng Long bằng công cụ tìm kiếm kết hợp Vector & Toàn văn.","parameters":{"type":"object","properties":{"query":{"type":"string","description":"Nội dung hoặc câu hỏi cần tra cứu quy chế bằng tiếng Việt (giữ nguyên mã môn học, tên ngành, từ khóa quan trọng), ví dụ: 'FTO1201 Cơ sở văn hóa Việt Nam miễn giảm học phí', 'chuẩn đầu ra tiếng Anh TOEIC', 'quy chế học lại'."},"cohort":{"type":"string","description":"Khóa học của sinh viên nếu câu hỏi áp dụng riêng cho khóa (ví dụ: 'K35', 'K38'). Để trống nếu tìm kiếm chung."},"domain":{"type":"string","description":"Lĩnh vực quy chế nếu biết (ví dụ: 'DAO_TAO', 'QUY_CHE', 'HOC_PHI', 'CHUAN_DAU_RA', 'TUYEN_SINH', 'TIN_TUC'). Để trống để tìm kiếm toàn diện trên tất cả lĩnh vực (khuyến nghị để trống trừ khi cần giới hạn phạm vi hẹp)."}},"required":["query"]}}
---
# Kỹ Năng: Tra Cứu Quy Chế & Văn Bản Học Vụ (search_regulations)

## Mục Đích

Kỹ năng này chịu trách nhiệm tra cứu, trích xuất và giải thích các điều khoản trong Quy chế Đào tạo Tín chỉ Đại học Thăng Long, Sổ tay Sinh viên và các Quyết định/Thông báo hiện hành.

## Hướng Dẫn Thực Hiện & Quy Tắc Phản Biện

1. **Dịch Thuật Ngữ & Nhận Diện Ngữ Cảnh (Query & Context Analysis)**:
   - Khi sinh viên hỏi bằng từ lóng hoặc từ viết tắt (ví dụ: "toạch môn", "học lại", "bảo lưu", "K36"), chủ động chuyển đổi sang thuật ngữ hành chính chính thức ("học phần bị điểm F", "đăng ký học lại cải thiện điểm", "nghỉ học tạm thời", "Khóa 36").
   - Khi tra cứu về các học phần cụ thể, mức tiền học phí, miễn giảm chuyển đổi, chuẩn đầu ra, hoặc quyết định quy chế: LUÔN giữ nguyên mã học phần (ví dụ: `FTO1201`, `IT499`), tên ngành, và số khóa (`K38`, `K35`) trong tham số `query` để động cơ tìm kiếm kết hợp trích xuất chính xác điều khoản và bảng biểu tương ứng.
   - Khi không chắc chắn về phân hệ lĩnh vực, để trống tham số `domain` để tìm kiếm toàn diện trên mọi văn bản (đặc biệt văn bản học phí `HOC_PHI`, chuẩn đầu ra `CHUAN_DAU_RA`).
   - Xác định xem câu hỏi có thuộc nhóm câu hỏi phụ thuộc khóa học (Khóa Kxx) và ngành học hay không. Nếu có mà chưa rõ khóa hoặc ngành, phải chủ động hỏi lại để thu hẹp phạm vi văn bản áp dụng.

2. **Trích Dẫn Căn Cứ Rõ Ràng**:
   - Mọi quy định phải chỉ rõ: "Căn cứ theo Điều X, Khoản Y của Quy chế đào tạo số..." hoặc "Theo Quyết định số... về chuẩn đầu ra ngoại ngữ...".

3. **Kiểm Tra Trạng Thái Văn Bản & Xử Lý Ghi Đè**:
   - Chỉ sử dụng văn bản đang có hiệu lực (`ACTIVE`), không áp dụng văn bản đã bị thay thế (`SUPERSEDED`).
   - Ví dụ: Quy chế đào tạo 2023-2024 đã hết hiệu lực, phải áp dụng Quy chế đào tạo 2026 (QĐ 2608/QĐ-ĐHTL).
