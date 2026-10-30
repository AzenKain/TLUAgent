package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	tluagentdb "tluagent-web/db"
	"tluagent-web/pkg/config"
)

func defaultMaxOpenConns() int {
	conns := runtime.NumCPU() * 2
	if conns < 4 {
		return 4
	}
	if conns > 16 {
		return 16
	}
	return conns
}

// sortMigrationFiles orders migrations by numeric prefix, since 100_* must run after 99_*.
func sortMigrationFiles(files []string) {
	sort.Slice(files, func(i, j int) bool {
		ni, oki := migrationNumber(files[i])
		nj, okj := migrationNumber(files[j])
		if oki && okj && ni != nj {
			return ni < nj
		}
		if oki != okj {
			return oki
		}
		return files[i] < files[j]
	})
}

func migrationNumber(name string) (int64, bool) {
	end := 0
	for end < len(name) && name[end] >= '0' && name[end] <= '9' {
		end++
	}
	if end == 0 || (end < len(name) && name[end] != '_' && name[end] != '.') {
		return 0, false
	}
	n, err := strconv.ParseInt(name[:end], 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func ApplySchema(db *sql.DB) error {
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(tluagentdb.SchemaFS, "schema")
	if err != nil {
		return err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, entry.Name())
	}
	sortMigrationFiles(files)

	for _, file := range files {
		var count int
		err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, file).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		path := "schema/" + file
		schema, err := fs.ReadFile(tluagentdb.SchemaFS, path)
		if err != nil {
			return err
		}

		if _, err := db.ExecContext(context.Background(), string(schema)); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				log.Printf("Warning: duplicate column in %s, marking as applied", file)
			} else if strings.Contains(err.Error(), "already exists") {
				log.Printf("Warning: object already exists in %s, marking as applied", file)
			} else {
				return fmt.Errorf("apply schema %s: %w", path, err)
			}
		}

		if _, err := db.ExecContext(context.Background(), `INSERT INTO schema_migrations (version) VALUES (?)`, file); err != nil {
			return fmt.Errorf("record migration %s: %w", file, err)
		}
	}
	return nil
}

func autoSQLiteCacheKB(maxOpen int) int {
	totalKB := config.GetIntConfigWithDefault("SQLITE_CACHE_SIZE_KB", 0)
	if totalKB <= 0 {
		totalKB = int(systemMemoryBytes() / 64 / 1024)
		if totalKB < 64*1024 {
			totalKB = 64 * 1024
		}
		if totalKB > 512*1024 {
			totalKB = 512 * 1024
		}
	}
	if maxOpen < 1 {
		maxOpen = 1
	}
	perConnection := totalKB / maxOpen
	if perConnection < 4*1024 {
		perConnection = 4 * 1024
	}
	return perConnection
}

func autoSQLiteMmapSize() int64 {
	if configured := config.GetIntConfigWithDefault("SQLITE_MMAP_SIZE_BYTES", 0); configured > 0 {
		return int64(configured)
	}
	mmap := systemMemoryBytes() / 8
	if mmap < 256<<20 {
		mmap = 256 << 20
	}
	if mmap > 2<<30 {
		mmap = 2 << 30
	}
	return mmap
}

// SQLiteDBPath resolves the database file location from the environment.
func SQLiteDBPath() string {
	dataDir := config.GetConfigWithDefault("DATA_DIR", "./data")
	return config.GetConfigWithDefault("SQLITE_DB_PATH", filepath.Join(dataDir, "tluagent.db"))
}

// dropStaleSHM removes a leftover shared-memory index before the database is opened.
// SQLite rebuilds it from the write-ahead log, and a copy produced by another operating
// system cannot be mapped, which would make the database unopenable after a move.
func dropStaleSHM(dbPath string) {
	shmPath := dbPath + "-shm"
	if _, err := os.Stat(shmPath); err != nil {
		return
	}
	if err := os.Remove(shmPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to remove stale shared-memory file %s: %v", shmPath, err)
		return
	}
	log.Printf("Removed stale SQLite shared-memory file %s", shmPath)
}

func NewSQLiteDB() (*sql.DB, error) {
	dbPath := SQLiteDBPath()
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, err
	}
	dropStaleSHM(dbPath)
	if _, err := os.Stat(dbPath); err == nil {
		_ = os.Chmod(dbPath, 0600)
	}

	maxOpen := config.GetIntConfigWithDefault("SQLITE_MAX_OPEN_CONNS", defaultMaxOpenConns())
	if maxOpen < 1 {
		maxOpen = defaultMaxOpenConns()
	}
	cacheKB := autoSQLiteCacheKB(maxOpen)
	mmapBytes := autoSQLiteMmapSize()
	db, err := sql.Open("sqlite", sqliteDSN(dbPath, cacheKB, mmapBytes))
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(config.GetIntConfigWithDefault("SQLITE_MAX_IDLE_CONNS", maxOpen))

	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func sqliteDSN(dbPath string, cacheKB int, mmapBytes int64) string {
	values := url.Values{}
	values.Add("_txlock", "immediate")
	values.Add("_pragma", "busy_timeout=10000")
	values.Add("_pragma", "foreign_keys(ON)")
	values.Add("_pragma", "trusted_schema(OFF)")
	values.Add("_pragma", "journal_mode(WAL)")
	values.Add("_pragma", "synchronous(NORMAL)")
	values.Add("_pragma", "temp_store(MEMORY)")
	values.Add("_pragma", fmt.Sprintf("cache_size(-%d)", cacheKB))
	values.Add("_pragma", fmt.Sprintf("mmap_size(%d)", mmapBytes))
	values.Add("_time_format", "datetime")
	values.Add("_timezone", "UTC")

	separator := "?"
	if strings.Contains(dbPath, "?") {
		separator = "&"
	}
	return dbPath + separator + values.Encode()
}

// CheckpointWALAndClose flushes every pending write into the database file, switches the
// journal back to DELETE and removes the -wal/-shm sidecars so the data directory stays a
// single portable file that survives a move between operating systems.
func CheckpointWALAndClose(db *sql.DB) error {
	if db == nil {
		return nil
	}
	dbPath := SQLiteDBPath()
	ctx := context.Background()

	db.SetMaxIdleConns(0)
	if conn, err := db.Conn(ctx); err == nil {
		if _, err := conn.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			log.Printf("Warning: WAL checkpoint failed: %v", err)
		}
		if _, err := conn.ExecContext(ctx, `PRAGMA journal_mode=DELETE`); err != nil {
			log.Printf("Warning: switching journal mode to DELETE failed: %v", err)
		}
		_ = conn.Close()
	}
	if err := db.Close(); err != nil {
		return err
	}
	return removeWALSidecars(dbPath)
}

func removeWALSidecars(dbPath string) error {
	walPath := dbPath + "-wal"
	if info, err := os.Stat(walPath); err == nil {
		if info.Size() > 0 {
			return fmt.Errorf("write-ahead log still holds %d bytes after checkpoint: %s", info.Size(), walPath)
		}
		if err := os.Remove(walPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove write-ahead log: %w", err)
		}
	}
	if err := os.Remove(dbPath + "-shm"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove shared-memory file: %w", err)
	}
	return nil
}
