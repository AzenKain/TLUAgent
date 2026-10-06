package embedding

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLiveCatalogDownloadAndValidate(t *testing.T) {
	if os.Getenv("EMBEDDING_LIVE_TEST") != "1" {
		t.Skip("Skipping live ONNX test: EMBEDDING_LIVE_TEST is not set to 1")
	}

	root := t.TempDir()
	downloader := NewDownloader(root)
	catalog := BuiltinCatalog()

	runtimePresetID := "onnxruntime-" + runtime.GOOS + "-" + runtime.GOARCH
	var runtimePreset *CatalogRuntime
	for i := range catalog.Runtimes {
		if catalog.Runtimes[i].PresetID == runtimePresetID {
			runtimePreset = &catalog.Runtimes[i]
			break
		}
	}
	if runtimePreset == nil {
		t.Skipf("Skipping live ONNX test: no runtime preset for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if _, err := os.Stat(LibraryPathForTest(t, root, runtimePreset)); err == nil {
		t.Log("runtime library already present, skipping download")
	} else {
		downloadStart := time.Now()
		job, err := downloader.Start(runtimePresetID)
		if err != nil {
			t.Fatalf("runtime download failed to start: %v", err)
		}
		waitForJob(t, downloader, job.JobID, 10*time.Minute)
		t.Logf("runtime download finished in %s", time.Since(downloadStart).Round(time.Millisecond))
	}

	modelStart := time.Now()
	modelJob, err := downloader.Start("all-minilm-l6-v2")
	if err != nil {
		t.Fatalf("model download failed to start: %v", err)
	}
	waitForJob(t, downloader, modelJob.JobID, 30*time.Minute)
	t.Logf("model download finished in %s", time.Since(modelStart).Round(time.Millisecond))

	validation := ValidateModel(context.Background(), root, "all-minilm-l6-v2")
	if !validation.OK {
		t.Fatalf("live validation failed: %s", validation.Error)
	}
	if validation.Dim != 384 {
		t.Fatalf("expected dim 384, got %d", validation.Dim)
	}
	t.Logf("live validation ok: dim=%d latency_ms=%.2f rss_mb=%.1f", validation.Dim, validation.LatencyMs, processRSSMB())

	modelDir := root + string(os.PathSeparator) + "all-minilm-l6-v2"
	manifest, err := LoadManifest(modelDir)
	if err != nil {
		t.Fatalf("downloaded manifest invalid: %v", err)
	}
	modelPath, err := FindModelFile(modelDir)
	if err != nil {
		t.Fatalf("model file missing: %v", err)
	}
	libPath, err := CurrentRuntimeLibraryPath(root)
	if err != nil {
		t.Fatalf("runtime path resolution failed: %v", err)
	}
	embedder, err := NewOnnxEmbedder(modelDir, manifest, modelPath, libPath)
	if err != nil {
		t.Fatalf("embedder creation failed: %v", err)
	}
	defer embedder.Close()
	vecs, err := embedder.Embed(context.Background(), []string{"the quick brown fox", "the quick brown fox jumps"})
	if err != nil {
		t.Fatalf("embedder run failed: %v", err)
	}
	sameTextCosine := CosineSimilarity(vecs[0], vecs[0])
	crossCosine := CosineSimilarity(vecs[0], vecs[1])
	if sameTextCosine < 0.999 || crossCosine <= 0 || crossCosine >= 1 {
		t.Fatalf("unexpected cosine similarities: self=%.6f cross=%.6f", sameTextCosine, crossCosine)
	}
	t.Logf("embedder check: self_cosine=%.4f cross_cosine=%.4f rss_mb=%.1f", sameTextCosine, crossCosine, processRSSMB())

	record := LoadValidationRecord(modelDir)
	if record == nil || !record.OK {
		t.Fatalf("validation record must be persisted: %+v", record)
	}
}

func processRSSMB() float64 {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseFloat(fields[1], 64)
				if err == nil {
					return kb / 1024.0
				}
			}
		}
	}
	return 0
}

func LibraryPathForTest(t *testing.T, root string, preset *CatalogRuntime) string {
	t.Helper()
	path, err := RuntimeLibraryPath(root, preset.GOOS, preset.GOArch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return path
}

func waitForJob(t *testing.T, downloader *Downloader, jobID string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, job := range downloader.Jobs() {
			if job.JobID != jobID {
				continue
			}
			switch job.State {
			case JobStateDone:
				return
			case JobStateError:
				t.Fatalf("download job %s failed: %s", jobID, job.Error)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("download job %s timed out after %s", jobID, timeout)
}
