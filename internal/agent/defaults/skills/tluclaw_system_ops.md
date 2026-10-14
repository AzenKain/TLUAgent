---
id: tluclaw_system_ops
name: Giám sát Hệ thống, Cache & AI Orchestration (TLUClaw)
description: Chỉ thị chuyên sâu cho TLUClaw thống kê tài nguyên hệ thống, xóa bộ nhớ đệm RAM Cache và điều phối mô hình AI phục vụ.
priority: 95
tool_definition: |
  {"name":"tluclaw_system_operations","description":"Operational toolset for system diagnostics, cache management, and AI model orchestration. Supports inspecting system statistics, flushing RAM cache, inspecting LLM status, and proposing default AI model changes.","parameters":{"type":"object","properties":{"action":{"type":"string","enum":["inspect_system_stats","flush_system_cache","inspect_llm_status","propose_switch_model"],"description":"Action: inspect_system_stats, flush_system_cache, inspect_llm_status, propose_switch_model"},"provider_id":{"type":"string","description":"LLM provider identifier"},"model_id":{"type":"string","description":"LLM model identifier to set as default"},"model_type":{"type":"string","enum":["CHAT","EMBEDDING","RERANK"],"description":"LLM model category"},"scope":{"type":"string","enum":["all","curriculum","rag","users","transcripts"],"description":"Cache flushing scope: all, curriculum, rag, users, transcripts"}},"required":["action"]}}
---
# Kỹ Năng: Giám Sát Hệ Thống, Cache & AI Orchestration (tluclaw_system_ops)

## Mục Đích
Kỹ năng này phục vụ Quản trị viên theo dõi sức khỏe hệ thống và điều phối hạ tầng:
1. Thống kê toàn cảnh dữ liệu (sinh viên, môn học, quy chế, cache keys, snapshots).
2. Xóa sạch bộ nhớ đệm RAM Cache theo vùng nghiệp vụ (`curriculum`, `rag`, `users`, `transcripts` hoặc `all`).
3. Kiểm tra trạng thái các nhà cung cấp AI và mô hình đang kết nối.
4. Dự thảo chuyển đổi mô hình AI chính/dự phòng qua Preview Card.
