package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func setupFlushableDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "flush.db")
	t.Setenv("DATA_DIR", filepath.Dir(dbPath))
	t.Setenv("SQLITE_DB_PATH", dbPath)

	db, err := NewSQLiteDB()
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.ExecContext(context.Background(), `CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	for i := 0; i < 50; i++ {
		if _, err := db.ExecContext(context.Background(), `INSERT INTO items (name) VALUES (?)`, "row"); err != nil {
			t.Fatalf("failed to insert row: %v", err)
		}
	}
	return db, dbPath
}

func TestCheckpointWALAndCloseLeavesNoSidecarFiles(t *testing.T) {
	db, dbPath := setupFlushableDB(t)

	if _, err := os.Stat(dbPath + "-wal"); err != nil {
		t.Fatalf("expected a write-ahead log to exist while the database is open: %v", err)
	}

	if err := CheckpointWALAndClose(db); err != nil {
		t.Fatalf("checkpoint and close failed: %v", err)
	}

	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(dbPath + suffix); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed after shutdown, stat error: %v", suffix, err)
		}
	}

	reopened, err := NewSQLiteDB()
	if err != nil {
		t.Fatalf("failed to reopen database: %v", err)
	}
	defer reopened.Close()

	var count int
	if err := reopened.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM items`).Scan(&count); err != nil {
		t.Fatalf("failed to read flushed rows: %v", err)
	}
	if count != 50 {
		t.Fatalf("expected 50 flushed rows, got %d", count)
	}

	var journalMode string
	if err := reopened.QueryRowContext(context.Background(), `PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("failed to read journal mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("expected the next run to use WAL again, got %q", journalMode)
	}
}

func TestNewSQLiteDBDropsStaleSharedMemoryFile(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "stale.db")
	t.Setenv("DATA_DIR", filepath.Dir(dbPath))
	t.Setenv("SQLITE_DB_PATH", dbPath)

	shmPath := dbPath + "-shm"
	if err := os.WriteFile(shmPath, []byte("incompatible-shared-memory"), 0o600); err != nil {
		t.Fatalf("failed to seed stale shm file: %v", err)
	}

	db, err := NewSQLiteDB()
	if err != nil {
		t.Fatalf("failed to open database with a stale shm file: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(shmPath); !os.IsNotExist(err) {
		t.Fatalf("expected the stale shared-memory file to be dropped")
	}
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE t (id INTEGER)`); err != nil {
		t.Fatalf("database is not usable after dropping stale shm: %v", err)
	}
}
