---
id: tluclaw_inquiry_ops
name: Xử lý & Phê duyệt Thắc mắc Học vụ (TLUClaw)
description: Chỉ thị chuyên sâu cho TLUClaw rà soát các câu hỏi tư vấn học vụ từ sinh viên và dự thảo câu trả lời chính thức căn cứ theo quy chế đào tạo.
priority: 95
tool_definition: |
  {"name":"tluclaw_inquiry_operations","description":"Operational toolset for student advisory inquiries. Supports listing pending inquiries, inspecting inquiry details and context, and proposing official answers with regulatory basis.","parameters":{"type":"object","properties":{"action":{"type":"string","enum":["list_pending_inquiries","inspect_inquiry_detail","propose_reply_inquiry"],"description":"Action: list_pending_inquiries, inspect_inquiry_detail, propose_reply_inquiry"},"inquiry_id":{"type":"string","description":"Unique identifier of advisory inquiry"},"search":{"type":"string","description":"Keyword to search question or student details"},"reply":{"type":"string","description":"Official advisory reply to student"},"answer":{"type":"string","description":"Proposed answer to the student inquiry"},"status":{"type":"string","enum":["PENDING","ANSWERED","EXPIRED","ALL"],"description":"Filter by inquiry status (default PENDING)"},"chunk_id":{"type":"string","description":"Optional regulatory chunk ID supporting the reply"},"page":{"type":"integer","description":"Page number (default 1)"},"limit":{"type":"integer","description":"Maximum number of records to return (default 20)"}},"required":["action"]}}
---
# Kỹ Năng: Xử Lý & Phê Duyệt Thắc Mắc Học Vụ (tluclaw_inquiry_ops)

## Mục Đích
Kỹ năng này phục vụ Quản trị viên và Cố vấn giải quyết các thắc mắc học vụ sinh viên gửi lên:
1. Liệt kê các câu hỏi đang chờ giải quyết (`PENDING`) kèm phân loại chủ đề.
2. Đọc chi tiết câu hỏi, ngữ cảnh hội thoại trước đó của sinh viên.
3. Dự thảo câu trả lời chính xác, trích dẫn căn cứ điều khoản quy chế, tạo Preview Card để Cố vấn/Admin xác nhận trước khi gửi về cho sinh viên.
