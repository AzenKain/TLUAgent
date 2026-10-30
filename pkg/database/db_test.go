package database

import "testing"

func TestSortMigrationFiles(t *testing.T) {
	files := []string{
		"100_audit_runs.sql",
		"99_chat_message_escalate.sql",
		"00_pragmas.sql",
		"102_document_validity_backfill.sql",
		"10_auth.sql",
		"101_kg_edge_status_backfill.sql",
		"9_legacy.sql",
	}
	sortMigrationFiles(files)

	want := []string{
		"00_pragmas.sql",
		"9_legacy.sql",
		"10_auth.sql",
		"99_chat_message_escalate.sql",
		"100_audit_runs.sql",
		"101_kg_edge_status_backfill.sql",
		"102_document_validity_backfill.sql",
	}
	for i := range want {
		if files[i] != want[i] {
			t.Fatalf("migration order mismatch at %d: got %q, want %q (order: %v)", i, files[i], want[i], files)
		}
	}
}
