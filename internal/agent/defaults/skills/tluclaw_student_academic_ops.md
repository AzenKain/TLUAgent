---
id: tluclaw_student_academic_ops
name: Quản lý Hồ sơ Học vụ, Điểm & Học phí Sinh viên (TLUClaw)
description: Chỉ thị chuyên sâu cho TLUClaw tra cứu hồ sơ học tập sinh viên, bảng điểm chi tiết, nợ học phí thời gian thực, lọc cảnh báo học vụ và ghi chú cố vấn.
priority: 95
tool_definition: |
  {"name":"tluclaw_student_academic_operations","description":"Operational toolset for student academic records, transcripts, academic standings, tuition debt status, and advisor notes. Supports inspecting student profiles, inspecting full transcripts, inspecting real-time tuition fee debt, listing students on academic warnings, and proposing advisor notes.","parameters":{"type":"object","properties":{"action":{"type":"string","enum":["inspect_student_profile","inspect_student_transcript","inspect_tuition_fee","list_academic_warnings","propose_advisor_note"],"description":"Action: inspect_student_profile, inspect_student_transcript, inspect_tuition_fee, list_academic_warnings, propose_advisor_note"},"student_id":{"type":"string","description":"Unique student identifier"},"student_code":{"type":"string","description":"Student identification code (MSSV, e.g. A44519, A36999)"},"user_id":{"type":"string","description":"Unique system user identifier"},"email":{"type":"string","description":"Student email address"},"cohort":{"type":"string","description":"Student cohort (e.g. K35, K36, K37)"},"major":{"type":"string","description":"Major code or name"},"major_name":{"type":"string","description":"Major name"},"academic_standing":{"type":"string","enum":["CBHT","BTH","NORMAL","ALL"],"description":"Academic standing filter (CBHT: Cảnh báo học tập, BTH: Buộc thôi học, NORMAL: Bình thường)"},"current_semester":{"type":"integer","description":"Current semester (1-8)"},"target_graduation_term":{"type":"integer","description":"Target graduation term / semester"},"advisor_note":{"type":"string","description":"Advisor guidance note or academic warning details to append/record"},"page":{"type":"integer","description":"Page number (default 1)"},"limit":{"type":"integer","description":"Maximum number of records to return (default 20)"}},"required":["action"]}}
---
# Kỹ Năng: Quản Lý Hồ Sơ Học Vụ, Điểm & Học Phí Sinh Viên (tluclaw_student_academic_ops)

## Mục Đích
Kỹ năng này phục vụ Quản trị viên (Admin) và Cố vấn học tập (Advisor) điều hành dữ liệu người học:
1. Tra cứu toàn diện hồ sơ học vụ sinh viên (MSSV, GPA tích lũy, số tín chỉ hoàn thành, tình trạng nợ học phí trực tiếp từ cổng thanh toán học phí TLU).
2. Kiểm tra chi tiết bảng điểm từng học kỳ, danh sách môn đã qua, môn nợ cần học lại.
3. Lọc danh sách sinh viên diện cảnh báo học tập (CBHT) hoặc buộc thôi học (BTH) theo Khóa và Ngành.
4. Ghi chú ý kiến tư vấn học vụ vào hồ sơ sinh viên qua Preview Card và lưu vết vĩnh viễn.
