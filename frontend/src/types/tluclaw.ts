export interface TLUClawSession {
  id: string;
  admin_user_id: string;
  title: string;
  created_at: string;
  updated_at: string;
}

export interface TLUClawDiffItem {
  entity_type: string;
  action: 'CREATE' | 'UPDATE' | 'DELETE' | 'SUPERSEDE' | 'UPSERT' | string;
  key: string;
  name?: string;
  old_value?: string;
  new_value?: string;
  note?: string;
}

export interface TLUClawBlastRadius {
  affected_cohorts?: string[];
  affected_majors?: string[];
  superseded_doc_ids?: string[];
  superseded_doc_names?: string[];
  impact_summary: string;
  risk_level: 'LOW' | 'MEDIUM' | 'HIGH' | string;
}

export interface TLUClawPreviewCardData {
  snapshot_id: string;
  batch_id: string;
  action_type: string;
  summary: string;
  nonce: string;
  expires_at: string;
  diff_items: TLUClawDiffItem[];
  blast_radius: TLUClawBlastRadius;
}

export interface TLUClawMessage {
  id: string;
  session_id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  reasoning_content?: string;
  preview_card?: TLUClawPreviewCardData;
  did_compact?: boolean;
  created_at: string;
}

export interface TLUClawAttachment {
  file_name: string;
  file_type: string;
  data: string;
}

export interface TLUClawChatRequest {
  session_id?: string;
  message: string;
  images?: string[];
  attachments?: TLUClawAttachment[];
}

export interface TLUClawChatResponse {
  session_id: string;
  message_id: string;
  reply: string;
  reasoning_content?: string;
  preview_card?: TLUClawPreviewCardData;
  did_compact?: boolean;
  created_at: string;
}

export interface TLUClawConfirmRequest {
  snapshot_id: string;
  nonce: string;
  selected_keys?: string[];
}

export interface TLUClawCancelRequest {
  snapshot_id: string;
  nonce: string;
}

export interface TLUClawRollbackRequest {
  batch_id: string;
}

export interface TLUClawActionResponse {
  success: boolean;
  message: string;
  snapshot_id: string;
  batch_id?: string;
  status: 'COMMITTED' | 'CANCELLED' | 'ROLLED_BACK' | 'STAGED';
}

export interface TLUClawSessionDetailResponse {
  session: TLUClawSession;
  messages: TLUClawMessage[];
  snapshots?: TLUClawPreviewCardData[];
}

export interface TLUClawSnapshotItem {
  id: string;
  batch_id: string;
  session_id: string;
  operator_id: string;
  action_type: string;
  summary: string;
  diff_items: TLUClawDiffItem[];
  blast_radius: TLUClawBlastRadius;
  status: 'STAGED' | 'COMMITTED' | 'CANCELLED' | 'ROLLED_BACK';
  applied_at?: string;
  created_at: string;
}
