package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	_ "modernc.org/sqlite"

	"tluagent-web/pkg/worker"
)

type BenchmarkResult struct {
	Name         string
	TotalJobs    int
	DurationSec  float64
	Throughput   float64
	LatencyP50Ms float64
	LatencyP95Ms float64
	LatencyP99Ms float64
	LatencyMaxMs float64
	ErrorsCount  int64
}

func readSecretFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func calculateLatencyStats(latencies []float64) (p50, p95, p99, max float64) {
	if len(latencies) == 0 {
		return 0, 0, 0, 0
	}
	sort.Float64s(latencies)
	n := len(latencies)
	p50 = latencies[int(float64(n)*0.50)]
	p95 = latencies[int(float64(n)*0.95)]
	p99 = latencies[int(float64(n)*0.99)]
	max = latencies[n-1]
	return p50, p95, p99, max
}

func resolvePath(relPath string) string {
	if _, err := os.Stat(relPath); err == nil {
		return relPath
	}
	altPath := "../" + relPath
	if _, err := os.Stat(altPath); err == nil {
		return altPath
	}
	if _, err := os.Stat("../benchmark_data"); err == nil {
		return altPath
	}
	return relPath
}

func benchmarkSQLite(totalJobs int, concurrency int) BenchmarkResult {
	dbPath := resolvePath("benchmark_data/sqlite/benchmark.db")
	_ = os.MkdirAll(filepath.Dir(dbPath), 0755)
	_ = os.Remove(dbPath)
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(10000)&_pragma=foreign_keys(ON)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS outbox_jobs (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			payload TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		panic(err)
	}

	stmtInsert, err := db.Prepare("INSERT INTO outbox_jobs (id, type, status, payload) VALUES (?, ?, ?, ?)")
	if err != nil {
		panic(err)
	}
	defer stmtInsert.Close()

	stmtUpdate, err := db.Prepare("UPDATE outbox_jobs SET status = ? WHERE id = ?")
	if err != nil {
		panic(err)
	}
	defer stmtUpdate.Close()

	q := worker.NewQueue(10, 10000)
	var processedCount int64
	var errCount int64

	var dbMu sync.Mutex
	q.RegisterHandler("academic.inquiry", func(ctx context.Context, jobID string, payload string) error {
		dbMu.Lock()
		_, err := stmtUpdate.Exec("completed", jobID)
		dbMu.Unlock()
		if err != nil {
			atomic.AddInt64(&errCount, 1)
			return err
		}
		atomic.AddInt64(&processedCount, 1)
		return nil
	})

	q.Start()
	defer q.Stop()

	latencies := make([]float64, totalJobs)
	var wg sync.WaitGroup
	jobsPerWorker := totalJobs / concurrency

	start := time.Now()
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ctx := context.Background()
			for i := 0; i < jobsPerWorker; i++ {
				idx := workerID*jobsPerWorker + i
				jobID := uuid.NewString()

				t0 := time.Now()
				dbMu.Lock()
				_, err := stmtInsert.Exec(jobID, "academic.inquiry", "pending", "payload-data")
				dbMu.Unlock()

				if err != nil {
					atomic.AddInt64(&errCount, 1)
					continue
				}

				_ = q.Enqueue(ctx, worker.Job{
					ID:      jobID,
					Type:    "academic.inquiry",
					Payload: "payload-data",
				})

				latencies[idx] = float64(time.Since(t0).Microseconds()) / 1000.0
			}
		}(w)
	}

	wg.Wait()

	deadline := time.Now().Add(10 * time.Second)
	for atomic.LoadInt64(&processedCount) < int64(totalJobs-int(errCount)) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	duration := time.Since(start).Seconds()
	p50, p95, p99, max := calculateLatencyStats(latencies)

	return BenchmarkResult{
		Name:         "SQLite (WAL Mode) + Go Worker Queue",
		TotalJobs:    totalJobs,
		DurationSec:  duration,
		Throughput:   float64(totalJobs) / duration,
		LatencyP50Ms: p50,
		LatencyP95Ms: p95,
		LatencyP99Ms: p99,
		LatencyMaxMs: max,
		ErrorsCount:  errCount,
	}
}

func benchmarkPostgreSQL(totalJobs int, concurrency int, pgPass string) BenchmarkResult {
	connStr := fmt.Sprintf("host=localhost port=5433 user=bench_user password=%s dbname=bench_db sslmode=disable", pgPass)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(concurrency)
	db.SetMaxIdleConns(concurrency)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS outbox_jobs (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			payload TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		TRUNCATE TABLE outbox_jobs;
	`)
	if err != nil {
		panic(err)
	}

	stmtInsert, err := db.Prepare("INSERT INTO outbox_jobs (id, type, status, payload) VALUES ($1, $2, $3, $4)")
	if err != nil {
		panic(err)
	}
	defer stmtInsert.Close()

	stmtUpdate, err := db.Prepare("UPDATE outbox_jobs SET status = $1 WHERE id = $2")
	if err != nil {
		panic(err)
	}
	defer stmtUpdate.Close()

	q := worker.NewQueue(10, 10000)
	var processedCount int64
	var errCount int64

	q.RegisterHandler("academic.inquiry", func(ctx context.Context, jobID string, payload string) error {
		_, err := stmtUpdate.Exec("completed", jobID)
		if err != nil {
			atomic.AddInt64(&errCount, 1)
			return err
		}
		atomic.AddInt64(&processedCount, 1)
		return nil
	})

	q.Start()
	defer q.Stop()

	latencies := make([]float64, totalJobs)
	var wg sync.WaitGroup
	jobsPerWorker := totalJobs / concurrency

	start := time.Now()
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ctx := context.Background()
			for i := 0; i < jobsPerWorker; i++ {
				idx := workerID*jobsPerWorker + i
				jobID := uuid.NewString()

				t0 := time.Now()
				_, err := stmtInsert.Exec(jobID, "academic.inquiry", "pending", "payload-data")
				if err != nil {
					atomic.AddInt64(&errCount, 1)
					continue
				}

				_ = q.Enqueue(ctx, worker.Job{
					ID:      jobID,
					Type:    "academic.inquiry",
					Payload: "payload-data",
				})

				latencies[idx] = float64(time.Since(t0).Microseconds()) / 1000.0
			}
		}(w)
	}

	wg.Wait()

	deadline := time.Now().Add(10 * time.Second)
	for atomic.LoadInt64(&processedCount) < int64(totalJobs-int(errCount)) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	duration := time.Since(start).Seconds()
	p50, p95, p99, max := calculateLatencyStats(latencies)

	return BenchmarkResult{
		Name:         "PostgreSQL 18 (Alpine Docker) + Go Worker Queue",
		TotalJobs:    totalJobs,
		DurationSec:  duration,
		Throughput:   float64(totalJobs) / duration,
		LatencyP50Ms: p50,
		LatencyP95Ms: p95,
		LatencyP99Ms: p99,
		LatencyMaxMs: max,
		ErrorsCount:  errCount,
	}
}

func benchmarkRedisStreams(totalJobs int, concurrency int, appendOnly string, fsyncMode string, label string) BenchmarkResult {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6380",
	})
	defer rdb.Close()

	ctx := context.Background()
	_ = rdb.ConfigSet(ctx, "appendonly", appendOnly).Err()
	if appendOnly == "yes" {
		_ = rdb.ConfigSet(ctx, "appendfsync", fsyncMode).Err()
	}

	_ = rdb.Del(ctx, "bench_stream").Err()
	_ = rdb.XGroupCreateMkStream(ctx, "bench_stream", "bench_group", "$").Err()

	var processedCount int64
	var errCount int64
	stopConsumer := make(chan struct{})

	for c := 0; c < 10; c++ {
		go func(consumerID int) {
			consumerName := fmt.Sprintf("consumer-%d", consumerID)
			for {
				select {
				case <-stopConsumer:
					return
				default:
					entries, err := rdb.XReadGroup(context.Background(), &redis.XReadGroupArgs{
						Group:    "bench_group",
						Consumer: consumerName,
						Streams:  []string{"bench_stream", ">"},
						Count:    10,
						Block:    50 * time.Millisecond,
					}).Result()

					if err != nil || len(entries) == 0 {
						continue
					}

					for _, stream := range entries {
						for _, msg := range stream.Messages {
							_ = rdb.XAck(context.Background(), "bench_stream", "bench_group", msg.ID)
							atomic.AddInt64(&processedCount, 1)
						}
					}
				}
			}
		}(c)
	}

	latencies := make([]float64, totalJobs)
	var wg sync.WaitGroup
	jobsPerWorker := totalJobs / concurrency

	start := time.Now()
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			cctx := context.Background()
			for i := 0; i < jobsPerWorker; i++ {
				idx := workerID*jobsPerWorker + i
				jobID := uuid.NewString()

				t0 := time.Now()
				err := rdb.XAdd(cctx, &redis.XAddArgs{
					Stream: "bench_stream",
					Values: map[string]interface{}{
						"id":      jobID,
						"type":    "academic.inquiry",
						"payload": "payload-data",
					},
				}).Err()

				if err != nil {
					atomic.AddInt64(&errCount, 1)
					continue
				}

				latencies[idx] = float64(time.Since(t0).Microseconds()) / 1000.0
			}
		}(w)
	}

	wg.Wait()

	deadline := time.Now().Add(10 * time.Second)
	for atomic.LoadInt64(&processedCount) < int64(totalJobs-int(errCount)) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	close(stopConsumer)

	duration := time.Since(start).Seconds()
	p50, p95, p99, max := calculateLatencyStats(latencies)

	return BenchmarkResult{
		Name:         label,
		TotalJobs:    totalJobs,
		DurationSec:  duration,
		Throughput:   float64(totalJobs) / duration,
		LatencyP50Ms: p50,
		LatencyP95Ms: p95,
		LatencyP99Ms: p99,
		LatencyMaxMs: max,
		ErrorsCount:  errCount,
	}
}

func main() {
	pgPass := readSecretFile(resolvePath("benchmark_data/.pg_pass"))
	if pgPass == "" {
		pgPass = os.Getenv("PG_PASS")
	}

	totalJobs := 5000
	concurrency := 50

	fmt.Println("==========================================================================================")
	fmt.Printf(" REALISTIC WORKER QUEUE BENCHMARK COMPARISON (%d jobs, %d concurrency)\n", totalJobs, concurrency)
	fmt.Println("==========================================================================================")

	fmt.Println("\n[1/5] SQLite (WAL Mode) + Go Worker Queue (Outbox Pattern)...")
	resSQLite := benchmarkSQLite(totalJobs, concurrency)
	fmt.Printf("      Done! Throughput: %.1f jobs/sec, p50: %.3fms, p99: %.3fms\n", resSQLite.Throughput, resSQLite.LatencyP50Ms, resSQLite.LatencyP99Ms)

	fmt.Println("\n[2/5] PostgreSQL 18 (Alpine Docker) + Go Worker Queue...")
	resPG := benchmarkPostgreSQL(totalJobs, concurrency, pgPass)
	fmt.Printf("      Done! Throughput: %.1f jobs/sec, p50: %.3fms, p99: %.3fms\n", resPG.Throughput, resPG.LatencyP50Ms, resPG.LatencyP99Ms)

	fmt.Println("\n[3/5] Redis 8 Streams (Default / In-Memory - No Disk Sync)...")
	resRedisDefault := benchmarkRedisStreams(totalJobs, concurrency, "no", "", "Redis 8 (In-Memory / No AOF)")
	fmt.Printf("      Done! Throughput: %.1f jobs/sec, p50: %.3fms, p99: %.3fms\n", resRedisDefault.Throughput, resRedisDefault.LatencyP50Ms, resRedisDefault.LatencyP99Ms)

	fmt.Println("\n[4/5] Redis 8 Streams (AOF everysec - Fsync Every Second)...")
	resRedisEverysec := benchmarkRedisStreams(totalJobs, concurrency, "yes", "everysec", "Redis 8 (AOF everysec)")
	fmt.Printf("      Done! Throughput: %.1f jobs/sec, p50: %.3fms, p99: %.3fms\n", resRedisEverysec.Throughput, resRedisEverysec.LatencyP50Ms, resRedisEverysec.LatencyP99Ms)

	fmt.Println("\n[5/5] Redis 8 Streams (AOF always - Zero Task Loss / Strict Fsync)...")
	resRedisAlways := benchmarkRedisStreams(totalJobs, concurrency, "yes", "always", "Redis 8 (AOF always - Zero Loss)")
	fmt.Printf("      Done! Throughput: %.1f jobs/sec, p50: %.3fms, p99: %.3fms\n", resRedisAlways.Throughput, resRedisAlways.LatencyP50Ms, resRedisAlways.LatencyP99Ms)

	fmt.Println("\n==========================================================================================")
	fmt.Println(" FINAL DURABILITY & PERSISTENCE BENCHMARK REPORT")
	fmt.Println("==========================================================================================")
	results := []BenchmarkResult{resSQLite, resPG, resRedisDefault, resRedisEverysec, resRedisAlways}
	fmt.Printf("%-38s | %-12s | %-9s | %-9s | %-6s\n", "Architecture & Durability", "Throughput", "p50 (ms)", "p99 (ms)", "Errors")
	fmt.Println(strings.Repeat("-", 88))
	for _, r := range results {
		fmt.Printf("%-38s | %8.1f j/s | %6.3f ms | %6.3f ms | %d\n",
			r.Name, r.Throughput, r.LatencyP50Ms, r.LatencyP99Ms, r.ErrorsCount)
	}
	fmt.Println(strings.Repeat("=", 88))
}
