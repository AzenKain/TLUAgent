package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"tluagent-web/pkg/config"
	"tluagent-web/pkg/logging"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-healthcheck", "--healthcheck":
			os.Exit(runHealthcheck())
		}
	}

	if err := config.LoadEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load .env: %v\n", err)
	}

	dataDir := config.GetConfigWithDefault("DATA_DIR", "./data")
	logWriter, err := logging.NewRotatingWriter(
		filepath.Join(dataDir, "logs", "tluagent.log"),
		int64(config.GetIntConfigWithDefault("LOG_MAX_SIZE_MB", 10))<<20,
		config.GetIntConfigWithDefault("LOG_MAX_FILES", 5),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize log writer: %v\n", err)
		os.Exit(1)
	}
	defer logWriter.Close()

	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	log.Logger = zerolog.New(zerolog.MultiLevelWriter(consoleWriter, logWriter)).With().Timestamp().Logger()

	host := config.GetConfigWithDefault("HOST", "0.0.0.0")
	port := config.GetConfigWithDefault("PORT", "8080")
	addr := fmt.Sprintf("%s:%s", host, port)

	server := NewServer(addr)

	go func() {
		if err := server.Start(); err != nil {
			log.Fatal().Err(err).Msg("Server halted unexpectedly")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	} else {
		log.Info().Msg("Server stopped successfully")
	}
}

// runHealthcheck calls the local health endpoint and converts the result into a process exit code.
func runHealthcheck() int {
	_ = config.LoadEnv()

	port := config.GetConfigWithDefault("PORT", "8080")
	url := fmt.Sprintf("http://127.0.0.1:%s/api/health", port)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck returned status %d\n", resp.StatusCode)
		return 1
	}
	return 0
}
