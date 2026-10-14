-- Performance Indexes & Foreign Key Optimization

-- Users table optimization
CREATE INDEX IF NOT EXISTS idx_users_student_code ON users(student_code) WHERE student_code IS NOT NULL AND student_code != '';

-- Supersession audit log optimization
CREATE INDEX IF NOT EXISTS idx_supersession_active_doc ON supersession_audit_log(active_doc_id);
CREATE INDEX IF NOT EXISTS idx_supersession_superseded_doc ON supersession_audit_log(superseded_doc_id);

-- User notifications optimization
CREATE INDEX IF NOT EXISTS idx_notifications_inquiry_id ON user_notifications(inquiry_id) WHERE inquiry_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON user_notifications(user_id, created_at DESC);

-- Advisory inquiries optimization
CREATE INDEX IF NOT EXISTS idx_inquiries_conversation_id ON advisory_inquiries(conversation_id) WHERE conversation_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_inquiries_superseded_doc_id ON advisory_inquiries(superseded_by_doc_id) WHERE superseded_by_doc_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_inquiries_status_created ON advisory_inquiries(status, created_at DESC);

-- Student academic profiles optimization
CREATE INDEX IF NOT EXISTS idx_profiles_cohort_major ON student_academic_profiles(cohort, major);

-- Curriculum courses optimization
CREATE INDEX IF NOT EXISTS idx_curriculum_courses_curr_semester ON curriculum_courses(curriculum_id, suggested_semester);
CREATE INDEX IF NOT EXISTS idx_curriculum_courses_curr_block ON curriculum_courses(curriculum_id, knowledge_block);

-- Document chunks optimization
CREATE INDEX IF NOT EXISTS idx_chunks_doc_chunk ON document_chunks(document_id, chunk_index);
CREATE INDEX IF NOT EXISTS idx_chunks_rag_filter ON document_chunks(status, domain, applicable_cohort);

-- Knowledge graph optimization
CREATE INDEX IF NOT EXISTS idx_kg_nodes_type_status ON kg_nodes(node_type, status);

-- Documents URL and code optimization
CREATE INDEX IF NOT EXISTS idx_documents_file_path ON documents(file_path);
CREATE INDEX IF NOT EXISTS idx_documents_doc_code ON documents(doc_code);
