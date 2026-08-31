package inference

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLlamaServerHelperProcess(t *testing.T) {
	if os.Getenv("GYRIFI_LLAMA_HELPER") != "1" {
		return
	}
	countPath := os.Getenv("GYRIFI_LLAMA_COUNT")
	count := readHelperCount(countPath) + 1
	if err := os.WriteFile(countPath, []byte(strconv.Itoa(count)), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	mode := os.Getenv("GYRIFI_LLAMA_MODE")
	if mode == "stderr-exit" || (mode == "crash-loop-after-ready" && count > 1) {
		fmt.Fprintln(os.Stderr, "model worker failed diagnostically")
		os.Exit(2)
	}
	port := helperArgument("--port")
	listener, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/health" {
			http.NotFound(writer, request)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})}
	go func() { _ = server.Serve(listener) }()
	shouldCrash := (mode == "restart-once" && count == 1) || (mode == "reset-counter" && count <= 2) || (mode == "crash-loop-after-ready" && count == 1) || (mode == "cycles" && count <= 4)
	if shouldCrash {
		time.Sleep(300 * time.Millisecond)
		fmt.Fprintln(os.Stderr, "simulated unexpected exit")
		os.Exit(1)
	}
	select {}
}

func helperArgument(name string) string {
	for index := range os.Args {
		if os.Args[index] == name && index+1 < len(os.Args) {
			return os.Args[index+1]
		}
	}
	return ""
}

func readHelperCount(path string) int {
	value, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	count, _ := strconv.Atoi(strings.TrimSpace(string(value)))
	return count
}

func openFileDescriptors() int {
	for _, path := range []string{"/proc/self/fd", "/dev/fd"} {
		entries, err := os.ReadDir(path)
		if err == nil {
			return len(entries)
		}
	}
	return -1
}

func helperExecutable(t *testing.T, mode string) (string, string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	countPath := filepath.Join(directory, "starts")
	scriptPath := filepath.Join(directory, "llama-server-test")
	script := fmt.Sprintf("#!/bin/sh\nexec %q -test.run=TestLlamaServerHelperProcess -- \"$@\"\n", executable)
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GYRIFI_LLAMA_HELPER", "1")
	t.Setenv("GYRIFI_LLAMA_COUNT", countPath)
	t.Setenv("GYRIFI_LLAMA_MODE", mode)
	return scriptPath, countPath
}

func availablePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	return port
}

func testSupervisorOptions() supervisorOptions {
	return supervisorOptions{startupTimeout: 3 * time.Second, pollInterval: 5 * time.Millisecond, backoffBase: 30 * time.Millisecond, backoffCap: 60 * time.Millisecond, stopTimeout: time.Second, lineLimit: 128, stderrLines: 4}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool, description string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !condition() && time.Now().Before(deadline) {
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
	if !condition() {
		t.Fatalf("timed out waiting for %s", description)
	}
}

func TestSupervisorRestartsUnexpectedExit(t *testing.T) {
	executable, countPath := helperExecutable(t, "restart-once")
	server, err := startLlamaServer(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), executable, "test.gguf", availablePort(t), 3, testSupervisorOptions())
	if err != nil {
		t.Fatal(err)
	}
	defer server.Stop()
	waitFor(t, time.Second, func() bool { return server.State() == "restarting" }, "restarting state")
	waitFor(t, 2*time.Second, func() bool { return readHelperCount(countPath) >= 2 && server.Healthy() }, "healthy restart")
}

func TestSupervisorStopsAtRestartCap(t *testing.T) {
	executable, countPath := helperExecutable(t, "crash-loop-after-ready")
	server, err := startLlamaServer(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), executable, "test.gguf", availablePort(t), 2, testSupervisorOptions())
	if err != nil {
		t.Fatal(err)
	}
	defer server.Stop()
	waitFor(t, 2*time.Second, func() bool { return server.State() == "failed" }, "failed state")
	if count := readHelperCount(countPath); count != 3 {
		t.Fatalf("process starts = %d, want initial plus two restarts", count)
	}
}

func TestSuccessfulReadinessResetsFailureCounter(t *testing.T) {
	executable, countPath := helperExecutable(t, "reset-counter")
	server, err := startLlamaServer(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), executable, "test.gguf", availablePort(t), 1, testSupervisorOptions())
	if err != nil {
		t.Fatal(err)
	}
	defer server.Stop()
	waitFor(t, 3*time.Second, func() bool { return readHelperCount(countPath) >= 3 && server.Healthy() }, "third healthy process")
}

func TestContextCancellationDoesNotRestart(t *testing.T) {
	executable, countPath := helperExecutable(t, "stable")
	ctx, cancel := context.WithCancel(context.Background())
	server, err := startLlamaServer(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), executable, "test.gguf", availablePort(t), 3, testSupervisorOptions())
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	waitFor(t, time.Second, func() bool { return server.State() == "stopped" }, "stopped state")
	if count := readHelperCount(countPath); count != 1 {
		t.Fatalf("process restarted after cancellation: %d starts", count)
	}
}

func TestRepeatedRestartsSettleWithoutGoroutineLeak(t *testing.T) {
	before := runtime.NumGoroutine()
	beforeFDs := openFileDescriptors()
	executable, countPath := helperExecutable(t, "cycles")
	server, err := startLlamaServer(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), executable, "test.gguf", availablePort(t), 2, testSupervisorOptions())
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 4*time.Second, func() bool { return readHelperCount(countPath) >= 5 && server.Healthy() }, "repeated restarts")
	if err := server.Stop(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, time.Second, func() bool { return runtime.NumGoroutine() <= before+4 }, "goroutines to settle")
	if afterFDs := openFileDescriptors(); beforeFDs >= 0 && afterFDs > beforeFDs+2 {
		t.Fatalf("file descriptors grew from %d to %d", beforeFDs, afterFDs)
	}
}

func TestStartupFailureIncludesCapturedStderr(t *testing.T) {
	executable, _ := helperExecutable(t, "stderr-exit")
	_, err := startLlamaServer(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), executable, "test.gguf", availablePort(t), 2, testSupervisorOptions())
	if err == nil || !strings.Contains(err.Error(), "model worker failed diagnostically") {
		t.Fatalf("startup error = %v", err)
	}
}
