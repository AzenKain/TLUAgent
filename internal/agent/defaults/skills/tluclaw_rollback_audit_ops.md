---
id: tluclaw_rollback_audit_ops
name: Quản lý Lịch sử Thao tác & Hoàn tác Trạng thái (TLUClaw)
description: Chỉ thị chuyên sâu cho TLUClaw hoàn tác phiên thao tác (Rollback Batch), kiểm tra lịch sử snapshot và khôi phục trạng thái an toàn theo yêu cầu Admin.
priority: 90
tool_definition: |
  {"name":"tluclaw_audit_operations","description":"Bộ công cụ tra cứu lịch sử thay đổi và thực thi hoàn tác trạng thái cơ sở dữ liệu học vụ","parameters":{"type":"object","properties":{"action":{"type":"string","enum":["list_recent_batches","inspect_batch_snapshot","execute_rollback"],"description":"Hành động: list_recent_batches (xem các phiên thay đổi gần nhất), inspect_batch_snapshot (soi chi tiết snapshot trước/sau của một batch), execute_rollback (thực thi hoàn tác quay lại trạng thái cũ)"},"snapshot_id":{"type":"string","description":"Mã định danh snapshot học vụ"},"batch_id":{"type":"string","description":"Mã định danh phiên thao tác cần hoàn tác"},"status":{"type":"string","description":"Trạng thái snapshot (STAGED, COMMITTED, ROLLED_BACK)"},"page":{"type":"integer","description":"Số thứ tự trang (mặc định 1)"},"limit":{"type":"integer","description":"Số lượng phiên cần liệt kê (mặc định 20)"}},"required":["action"]}}
---
# Kỹ Năng: Quản Lý Lịch Sử Thao Tác & Hoàn Tác Trạng Thái (tluclaw_rollback_audit_ops)

## Mục Đích
Kỹ năng này phục vụ Quản trị viên (Admin) thông qua trợ lý TLUClaw:
1. Tra cứu lịch sử các phiên thay đổi dữ liệu học vụ do Admin thực hiện (ai làm, vào lúc nào, thay đổi những gì).
2. Khi Admin gửi yêu cầu: *"Hãy hoàn tác phiên vừa rồi"* hoặc *"Hủy bỏ thay đổi của batch xyz"*:
   - Tự động tìm snapshot tương ứng trong cơ sở dữ liệu (`previous_state_json`).
   - Tạo bản Preview hoàn tác: Chỉ rõ các bản ghi nào sẽ được khôi phục về trạng thái cũ.
   - Khi Admin xác nhận, thực thi hoàn tác nguyên tử và chuyển trạng thái snapshot sang `ROLLED_BACK`.

## Quy Tắc Hoạt Động Bắt Buộc

1. **Kiểm Tra Tính Hợp Lệ Của Phiên Hoàn Tác**:
   - Chỉ cho phép hoàn tác các phiên có trạng thái `COMMITTED`.
   - Nếu phiên đã được `ROLLED_BACK` trước đó, thông báo rõ ràng cho Admin và từ chối hoàn tác lần 2.

2. **Quy Trình 2 Pha Cho Tác Vụ Hoàn Tác**:
   - Trước khi thực sự rollback, hiển thị cho Admin xem danh sách các thực thể (môn học, tài liệu, liên kết đồ thị) sẽ bị đảo ngược trạng thái.
   - Chỉ thực hiện khôi phục dữ liệu khi Admin xác nhận.
