---
id: tluclaw_job_ops
name: Điều hành Tác vụ Nền & Data Pipeline (TLUClaw)
description: Chỉ thị chuyên sâu cho TLUClaw giám sát hàng đợi worker, tra cứu tiến độ pipeline cào dữ liệu và kích hoạt tác vụ nền bất đồng bộ.
priority: 95
tool_definition: |
  {"name":"tluclaw_job_operations","description":"Operational toolset for system background jobs, crawlers, and asynchronous pipelines. Supports listing available tasks, listing recent jobs, inspecting job status, and proposing job execution.","parameters":{"type":"object","properties":{"action":{"type":"string","enum":["list_available_tasks","list_recent_jobs","inspect_job_status","propose_trigger_job"],"description":"Action: list_available_tasks, list_recent_jobs, inspect_job_status, propose_trigger_job"},"job_id":{"type":"string","description":"Background job unique identifier"},"task_type":{"type":"string","description":"Worker task type (e.g. crawl_notices, crawl_news, crawl_static_pages, clean_corpus, supersede_check)"},"payload":{"type":"string","description":"Optional JSON payload or parameters string for the job"},"page":{"type":"integer","description":"Page number (default 1)"},"limit":{"type":"integer","description":"Maximum number of jobs to return (default 20)"}},"required":["action"]}}
---
# Kỹ Năng: Điều Hành Tác Vụ Nền & Data Pipeline (tluclaw_job_ops)

## Mục Đích
Kỹ năng này phục vụ Quản trị viên theo dõi và kích hoạt các tác vụ xử lý nền bất đồng bộ:
1. Tra cứu danh mục các loại tác vụ nền có sẵn trong hệ thống (cào tin tức, cào thông báo, chuẩn hóa tài liệu, embedding vector, xây dựng Knowledge Graph).
2. Kiểm tra danh sách các job gần đây, trạng thái hoàn thành, tỷ lệ tiến độ và thông báo lỗi.
3. Dự thảo kích hoạt tác vụ bất đồng bộ đưa vào hàng đợi worker sau khi Admin duyệt qua Preview Card.
