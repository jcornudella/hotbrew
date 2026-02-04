// Package daemon provides background sync and digest generation.
package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/curation"
	"github.com/jcornudella/hotbrew/internal/sinks"
	"github.com/jcornudella/hotbrew/internal/store"
	hsync "github.com/jcornudella/hotbrew/internal/sync"
	"github.com/jcornudella/hotbrew/pkg/source"
)

// pidFile returns the path to the daemon PID file.
func pidFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hotbrew", "daemon.pid")
}

// Start launches the daemon in the foreground (intended to be backgrounded by the caller).
func Start(cfg *config.Config, registry *source.Registry) error {
	// Check if already running.
	if pid := readPID(); pid > 0 {
		if processExists(pid) {
			return fmt.Errorf("daemon already running (PID %d)", pid)
		}
	}

	// Write PID file.
	if err := writePID(os.Getpid()); err != nil {
		return fmt.Errorf("write pid: %w", err)
	}
	defer os.Remove(pidFile())

	st, err := store.Open(cfg.GetDBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	interval := cfg.GetSyncInterval()
	fmt.Printf("☕ Daemon started (PID %d, interval %s)\n", os.Getpid(), interval)

	// Handle graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// Run immediately on start, then on interval.
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	runCycle(ctx, st, cfg, registry)

	for {
		select {
		case <-ticker.C:
			runCycle(ctx, st, cfg, registry)
		case sig := <-sigCh:
			fmt.Printf("\n☕ Daemon stopping (%v)\n", sig)
			return nil
		}
	}
}

// runCycle performs one sync + digest cycle.
func runCycle(ctx context.Context, st *store.Store, cfg *config.Config, registry *source.Registry) {
	syncCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	results := hsync.SyncAll(syncCtx, st, registry)
	hsync.PrintResults(results)

	// Generate digest and write to stream log.
	engine := curation.NewEngine(st)
	digest, err := engine.GenerateDigest(cfg.GetDigestWindow(), cfg.GetDigestMax(), "Hotbrew Digest")
	if err != nil {
		fmt.Printf("  ⚠ Digest error: %v\n", err)
		return
	}

	// Save digest to store.
	st.SaveDigest(digest)

	// Write to stream log.
	logSink := &sinks.StreamLog{Path: cfg.GetStreamLogPath()}
	if err := logSink.Deliver(digest); err != nil {
		fmt.Printf("  ⚠ Stream log error: %v\n", err)
	}

	fmt.Printf("  ✓ Digest: %d items → %s\n", digest.ItemCount, cfg.GetStreamLogPath())
}

// Stop sends SIGTERM to a running daemon.
func Stop() error {
	pid := readPID()
	if pid <= 0 {
		return fmt.Errorf("no daemon running (no PID file)")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		os.Remove(pidFile())
		return fmt.Errorf("signal process %d: %w", pid, err)
	}

	os.Remove(pidFile())
	fmt.Printf("☕ Daemon stopped (PID %d)\n", pid)
	return nil
}

// Status prints daemon status.
func Status() {
	pid := readPID()
	if pid <= 0 {
		fmt.Println("☕ Daemon: not running")
		return
	}

	if processExists(pid) {
		fmt.Printf("☕ Daemon: running (PID %d)\n", pid)
	} else {
		fmt.Printf("☕ Daemon: stale PID file (PID %d not running)\n", pid)
		os.Remove(pidFile())
	}
}

func writePID(pid int) error {
	path := pidFile()
	os.MkdirAll(filepath.Dir(path), 0755)
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}

func readPID() int {
	data, err := os.ReadFile(pidFile())
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0
	}
	return pid
}

func processExists(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
