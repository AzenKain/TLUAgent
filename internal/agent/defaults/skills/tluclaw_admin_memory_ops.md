---
id: tluclaw_admin_memory_ops
name: Quản lý Bộ nhớ Điều hành & Ngữ cảnh Quản trị (TLUClaw)
description: Chỉ thị chuyên sâu cho TLUClaw lưu trữ và tra cứu các tùy chọn điều hành dài hạn của Quản trị viên (khóa đang theo dõi, chính sách đào tạo ưu tiên).
priority: 95
tool_definition: |
  {"name":"tluclaw_admin_memory_operations","description":"Toolset for managing administrator operational preferences, persistent policies, working cohorts, and catalog focuses","parameters":{"type":"object","properties":{"action":{"type":"string","enum":["save_preference","list_preferences"],"description":"Action: save_preference, list_preferences"},"key":{"type":"string","description":"Preference key (e.g. WORKING_COHORT, MANAGED_FACULTY, CATALOG_FOCUS, REGULATION_POLICY)"},"value":{"type":"string","description":"Preference value to persist"},"category":{"type":"string","description":"Category (OPERATIONAL_CONTEXT, CATALOG_FOCUS, REGULATION_POLICY, ADMIN_PREFERENCE)"},"memory_key":{"type":"string","description":"Alias for preference key"},"memory_value":{"type":"string","description":"Alias for preference value"},"target_key":{"type":"string","description":"Alias for preference key / category target"},"page":{"type":"integer","description":"Page number (default 1)"},"limit":{"type":"integer","description":"Maximum number of records to return (default 20)"}},"required":["action"]}}
---
# Kỹ Năng: Quản Lý Bộ Nhớ Điều Hành & Ngữ Cảnh Quản Trị (tluclaw_admin_memory_ops)

## Mục Đích
Kỹ năng này phục vụ lưu trữ bền vững các tùy chọn và chỉ thị của Quản trị viên:
1. Ghi nhớ khóa sinh viên đang làm việc chính (WORKING_COHORT) hoặc khoa phụ trách.
2. Lưu các chính sách tư vấn hoặc trọng tâm kiểm tra học kỳ.
3. Tự động tải lại ngữ cảnh điều hành này trong các phiên làm việc tiếp theo của Admin.
