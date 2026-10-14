-- Migration: Add can_escalate column to chat_messages table
ALTER TABLE chat_messages ADD COLUMN can_escalate INTEGER NOT NULL DEFAULT 0;
