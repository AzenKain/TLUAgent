---
id: tluclaw_notification_ops
name: Phát & Quản lý Thông báo Học vụ (TLUClaw)
description: Chỉ thị chuyên sâu cho TLUClaw phát thông báo tới sinh viên, khóa học hoặc toàn trường kèm cơ chế Preview và Rollback thu hồi thông báo.
priority: 95
tool_definition: |
  {"name":"tluclaw_notification_operations","description":"Operational toolset for student notifications and university broadcasting. Supports listing sent notifications and proposing targeted or broadcast notifications with rollback protection.","parameters":{"type":"object","properties":{"action":{"type":"string","enum":["list_notifications","propose_notification"],"description":"Action: list_notifications, propose_notification"},"recipient_user_id":{"type":"string","description":"Target recipient user ID"},"user_id":{"type":"string","description":"User ID when target_type is STUDENT"},"title":{"type":"string","description":"Notification title / headline"},"content":{"type":"string","description":"Detailed notification body text"},"target_audience":{"type":"string","description":"Target audience description or identifier"},"target_type":{"type":"string","enum":["STUDENT","COHORT","ALL"],"description":"Target audience classification: STUDENT, COHORT, or ALL"},"student_code":{"type":"string","description":"Student ID (MSSV) when target_type is STUDENT"},"cohort":{"type":"string","description":"Student cohort (e.g. K36, K37) when target_type is COHORT"},"type":{"type":"string","description":"Notification type alias"},"notification_type":{"type":"string","enum":["ACADEMIC_ALERT","SYSTEM","ADVISORY","TUITION_NOTICE"],"description":"Notification category type"},"page":{"type":"integer","description":"Page number (default 1)"},"limit":{"type":"integer","description":"Maximum number of records to return (default 20)"}},"required":["action"]}}
---
# Kỹ Năng: Phát & Quản Lý Thông Báo Học Vụ (tluclaw_notification_ops)

## Mục Đích
Kỹ năng này phục vụ Quản trị viên và Cố vấn phát tin tức, thông báo khẩn hoặc cảnh báo học vụ:
1. Tra cứu danh sách thông báo đã gửi trong hệ thống.
2. Dự thảo gửi thông báo theo 3 cấp độ: Sinh viên cụ thể (`STUDENT`), Toàn khóa (`COHORT`), hoặc Toàn bộ người dùng (`ALL`).
3. Đánh giá phạm vi ảnh hưởng (Blast Radius), hiển thị số lượng người nhận trên Preview Card trước khi phát.
4. Cho phép hoàn tác (Rollback) thu hồi thông báo đã phát nếu có sự nhầm lẫn.
