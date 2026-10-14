---
id: tluclaw_user_ops
name: Quản lý Người dùng, Vai trò & Phân quyền (TLUClaw)
description: Chỉ thị chuyên sâu cho TLUClaw quản lý tài khoản người dùng, khóa/mở khóa tài khoản, gán/hạ quyền vai trò và tra cứu hệ thống phân quyền RBAC.
priority: 95
tool_definition: |
  {"name":"tluclaw_user_operations","description":"Operational toolset for user accounts, status, system roles, and permission access control. Supports searching users, inspecting user permissions and assigned roles, listing all system roles, listing all system permissions, proposing account status updates (ACTIVE/BANNED), and proposing role assignments/removals.","parameters":{"type":"object","properties":{"action":{"type":"string","enum":["search_users","inspect_user_detail","list_system_roles","list_system_permissions","propose_user_status","propose_user_role"],"description":"Action: search_users, inspect_user_detail, list_system_roles, list_system_permissions, propose_user_status, propose_user_role"},"search_text":{"type":"string","description":"Keyword to search across email, full name, or student code"},"keyword":{"type":"string","description":"Search keyword for filtering users"},"user_id":{"type":"string","description":"Target user ID"},"student_code":{"type":"string","description":"Target student code (MSSV)"},"email":{"type":"string","description":"Target user email"},"role_id":{"type":"string","description":"Target role ID"},"role_name":{"type":"string","enum":["ADMIN","ADVISOR","STUDENT","GUEST","BANNED"],"description":"System role name to assign or remove"},"role_action":{"type":"string","enum":["ASSIGN","REMOVE"],"description":"Whether to ASSIGN or REMOVE the role"},"status":{"type":"string","enum":["ACTIVE","BANNED"],"description":"Target account status (ACTIVE or BANNED)"},"reason":{"type":"string","description":"Administrative rationale for banning or activating user account"},"page":{"type":"integer","description":"Page number (default 1)"},"limit":{"type":"integer","description":"Maximum number of records to return (default 20)"}},"required":["action"]}}
---
# Kỹ Năng: Quản Lý Người Dùng, Vai Trò & Phân Quyền (tluclaw_user_ops)

## Mục Đích
Kỹ năng này chịu trách nhiệm phục vụ tài khoản Quản trị viên điều hành nhân sự và quyền hạn:
1. Tìm kiếm và soi chiếu thông tin người dùng (Sinh viên, Giảng viên, Quản trị viên).
2. Tra cứu toàn bộ các vai trò và quyền hạn chi tiết trong hệ thống RBAC.
3. Dự thảo khóa/kích hoạt tài khoản hoặc gán/hạ vai trò với cơ chế bảo vệ Preview Card và thu hồi phiên làm việc (Revoke Sessions).
