---
id: tluclaw_document_supersession_ops
name: Rà soát Văn bản, Phân tích Bãi bỏ Quy chế & Đồ thị Tri thức (TLUClaw)
description: Chỉ thị chuyên sâu cho TLUClaw rà soát tác động văn bản mới, đối thoại làm rõ căn cứ bãi bỏ, phát hiện điều khoản bãi bỏ văn bản cũ (SUPERSEDED), cập nhật Knowledge Graph, tra cứu FTS và sinh cảnh báo Blast Radius theo quy trình 3 giai đoạn.
priority: 95
tool_definition: |
  {"name":"tluclaw_document_operations","description":"Administrative toolset for academic regulations, RAG hybrid search, knowledge graph inspection, supersession impact, and document lifecycle management.","parameters":{"type":"object","properties":{"action":{"type":"string","enum":["list_active_regulations","search_knowledge","inspect_kg_entity","propose_document_status","propose_ingestion","analyze_supersession_impact"],"description":"Action: list_active_regulations, search_knowledge, inspect_kg_entity, propose_document_status, propose_ingestion, analyze_supersession_impact"},"query":{"type":"string","description":"Query string for RAG hybrid knowledge search"},"node_id":{"type":"string","description":"Knowledge Graph entity node identifier"},"document_id":{"type":"string","description":"Unique regulation document identifier"},"status":{"type":"string","enum":["ACTIVE","SUPERSEDED","ARCHIVED","DRAFT"],"description":"Target regulation document status"},"document_title":{"type":"string","description":"Official regulation or decision title"},"document_type":{"type":"string","description":"Type classification (QUY_CHE, THONG_BAO, HUONG_DAN)"},"domain":{"type":"string","description":"Academic domain (DAO_TAO, HOC_PHI, CO_VAN_HOC_TAP)"},"academic_year":{"type":"string","description":"Academic school year applicable"},"applicable_cohort":{"type":"string","description":"Target student cohort (e.g. K36, K37, K40)"},"document_content":{"type":"string","description":"Full text or excerpt of the regulation document"},"superseded_doc_id":{"type":"string","description":"Explicit identifier of old document being replaced"},"reason":{"type":"string","description":"Rationale for supersession or modification"},"confirmation_nonce":{"type":"string","description":"Security nonce from preview card"},"page":{"type":"integer","description":"Page number (default 1)"},"limit":{"type":"integer","description":"Maximum number of records to return (default 20)"}},"required":["action"]}}
---
# Kỹ Năng: Rà Soát Văn Bản, Phân Tích Bãi Bỏ Quy Chế & Đồ Thị Tri Thức (tluclaw_document_supersession_ops)

## Mục Đích
Kỹ năng này chịu trách nhiệm phục vụ tài khoản Quản trị viên (Admin) thông qua trợ lý TLUClaw:
1. Tiếp nhận các văn bản quy chế, quyết định đào tạo, sổ tay sinh viên hoặc thông báo học vụ mới.
2. Tìm kiếm ngữ nghĩa và FTS các đoạn văn bản quy chế, tra cứu thực thể và mối quan hệ trên Knowledge Graph.
3. Thay đổi trạng thái vòng đời văn bản (ACTIVE, SUPERSEDED, ARCHIVED, DRAFT) kèm lý do rõ ràng.
4. Phân tích tác động bãi bỏ (Blast Radius), nạp văn bản mới và tạo liên kết SUPERSEDES trên Knowledge Graph.
5. Tuân thủ nghiêm ngặt Quy trình Động 3 Giai đoạn (Dynamic 3-Stage Workflow), hỗ trợ hoàn tác an toàn 1-click.
