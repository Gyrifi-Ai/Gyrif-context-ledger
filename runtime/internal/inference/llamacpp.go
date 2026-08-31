package inference

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type LlamaCppProvider struct {
	endpoint string
	model    string
	client   *http.Client
	server   *LlamaServer
}

func NewLlamaCppProvider(endpoint, model string) *LlamaCppProvider {
	return &LlamaCppProvider{endpoint: strings.TrimRight(endpoint, "/"), model: model, client: &http.Client{Timeout: 2 * time.Minute}}
}
func (provider *LlamaCppProvider) Name() string { return "llamacpp" }
func (provider *LlamaCppProvider) Healthy() bool {
	return provider.server == nil || provider.server.Healthy()
}
func (provider *LlamaCppProvider) State() string {
	if provider.server == nil {
		return "ready"
	}
	return provider.server.State()
}
func (provider *LlamaCppProvider) Health(ctx context.Context) error {
	if !provider.Healthy() {
		return fmt.Errorf("llama-server state is %s", provider.State())
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.endpoint+"/health", nil)
	if err != nil {
		return err
	}
	response, err := provider.client.Do(request)
	if err != nil {
		return fmt.Errorf("llama-server health: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("llama-server health returned status %d", response.StatusCode)
	}
	return nil
}
func (provider *LlamaCppProvider) Evaluate(ctx context.Context, request EvaluationRequest) (EvaluationResult, error) {
	prompt := fmt.Sprintf("You are a context-governance evaluator. Evaluate the supplied proposed state against the criteria. Return only JSON with passed (boolean), summary (string), and findings (array of {severity,message,unit}). Proposal hash: %s\nCriteria: %s\nContext: %s", request.ProposalHash, request.Criteria, string(request.Context))
	payload := map[string]any{"model": provider.model, "messages": []map[string]string{{"role": "user", "content": prompt}}, "temperature": 0, "response_format": map[string]string{"type": "json_object"}}
	encoded, _ := json.Marshal(payload)
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.endpoint+"/v1/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return EvaluationResult{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := provider.client.Do(httpRequest)
	if err != nil {
		return EvaluationResult{}, fmt.Errorf("llama-server request: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return EvaluationResult{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return EvaluationResult{}, fmt.Errorf("llama-server returned status %d", response.StatusCode)
	}
	var envelope struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return EvaluationResult{}, fmt.Errorf("decode llama-server response: %w", err)
	}
	if len(envelope.Choices) == 0 {
		return EvaluationResult{}, fmt.Errorf("llama-server returned no choices")
	}
	var result EvaluationResult
	if err := json.Unmarshal([]byte(envelope.Choices[0].Message.Content), &result); err != nil {
		return EvaluationResult{}, fmt.Errorf("decode structured evaluation: %w", err)
	}
	result.Model = envelope.Model
	result.Evidence = map[string]string{"proposalHash": request.ProposalHash}
	return result, nil
}

type supervisorOptions struct {
	startupTimeout time.Duration
	pollInterval   time.Duration
	backoffBase    time.Duration
	backoffCap     time.Duration
	stopTimeout    time.Duration
	lineLimit      int
	stderrLines    int
}

var defaultSupervisorOptions = supervisorOptions{
	startupTimeout: 45 * time.Second,
	pollInterval:   250 * time.Millisecond,
	backoffBase:    time.Second,
	backoffCap:     time.Minute,
	stopTimeout:    5 * time.Second,
	lineLimit:      4096,
	stderrLines:    10,
}

type managedProcess struct {
	command *exec.Cmd
	exit    chan error
}

type LlamaServer struct {
	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	logger      *slog.Logger
	executable  string
	modelPath   string
	port        int
	maxRestarts int
	options     supervisorOptions
	state       string
	process     *managedProcess
	stderr      []string
	done        chan struct{}
	stopOnce    sync.Once
	Provider    *LlamaCppProvider
}

func StartLlamaServer(ctx context.Context, logger *slog.Logger, executable, modelPath string, port, maxRestarts int) (*LlamaServer, error) {
	return startLlamaServer(ctx, logger, executable, modelPath, port, maxRestarts, defaultSupervisorOptions)
}

func startLlamaServer(ctx context.Context, logger *slog.Logger, executable, modelPath string, port, maxRestarts int, options supervisorOptions) (*LlamaServer, error) {
	if logger == nil {
		logger = slog.Default()
	}
	serverCtx, cancel := context.WithCancel(ctx)
	server := &LlamaServer{ctx: serverCtx, cancel: cancel, logger: logger, executable: executable, modelPath: modelPath, port: port, maxRestarts: maxRestarts, options: options, state: "starting", done: make(chan struct{})}
	server.Provider = NewLlamaCppProvider(fmt.Sprintf("http://127.0.0.1:%d", port), modelPath)
	server.Provider.server = server
	process, err := server.startProcess()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("start llama-server: %w", err)
	}
	server.setProcess(process)
	if err := server.waitReady(process); err != nil {
		cancel()
		server.terminate(process)
		server.setState("failed")
		return nil, fmt.Errorf("llama-server did not become ready: %w%s", err, server.stderrSuffix())
	}
	server.setState("ready")
	go server.supervise(process)
	return server, nil
}

func (server *LlamaServer) startProcess() (*managedProcess, error) {
	command := exec.Command(server.executable, "--host", "127.0.0.1", "--port", strconv.Itoa(server.port), "--model", server.modelPath)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := command.Start(); err != nil {
		return nil, err
	}
	server.mu.Lock()
	server.stderr = nil
	server.mu.Unlock()
	var streams sync.WaitGroup
	streams.Add(2)
	go func() {
		defer streams.Done()
		server.drain(stdout, false)
	}()
	go func() {
		defer streams.Done()
		server.drain(stderr, true)
	}()
	process := &managedProcess{command: command, exit: make(chan error, 1)}
	go func() {
		streams.Wait()
		err := command.Wait()
		process.exit <- err
		close(process.exit)
	}()
	return process, nil
}

func (server *LlamaServer) drain(reader io.Reader, isStderr bool) {
	buffered := bufio.NewReaderSize(reader, server.options.lineLimit)
	var line strings.Builder
	truncated := false
	emit := func() {
		value := line.String()
		if truncated {
			value += " [truncated]"
		}
		stream := "stdout"
		if isStderr {
			stream = "stderr"
		}
		server.logger.Debug(value, "component", "llama-server", "stream", stream)
		if isStderr {
			server.rememberStderr(value)
		}
		line.Reset()
		truncated = false
	}
	for {
		fragment, prefix, err := buffered.ReadLine()
		if remaining := server.options.lineLimit - line.Len(); remaining > 0 {
			if len(fragment) > remaining {
				fragment = fragment[:remaining]
				truncated = true
			}
			_, _ = line.Write(fragment)
		} else if len(fragment) > 0 {
			truncated = true
		}
		if !prefix && (line.Len() > 0 || truncated) {
			emit()
		}
		if err != nil {
			if line.Len() > 0 || truncated {
				emit()
			}
			return
		}
	}
}

func (server *LlamaServer) rememberStderr(line string) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.stderr = append(server.stderr, line)
	if len(server.stderr) > server.options.stderrLines {
		server.stderr = append([]string(nil), server.stderr[len(server.stderr)-server.options.stderrLines:]...)
	}
}

func (server *LlamaServer) stderrSnapshot() []string {
	server.mu.Lock()
	defer server.mu.Unlock()
	return append([]string(nil), server.stderr...)
}

func (server *LlamaServer) stderrSuffix() string {
	lines := server.stderrSnapshot()
	if len(lines) == 0 {
		return ""
	}
	return "; stderr: " + strings.Join(lines, " | ")
}

func (server *LlamaServer) waitReady(process *managedProcess) error {
	deadline := time.NewTimer(server.options.startupTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(server.options.pollInterval)
	defer ticker.Stop()
	client := &http.Client{Timeout: time.Second}
	probe := func() bool {
		request, err := http.NewRequestWithContext(server.ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", server.port), nil)
		if err != nil {
			return false
		}
		response, err := client.Do(request)
		if err != nil {
			return false
		}
		response.Body.Close()
		return response.StatusCode >= 200 && response.StatusCode < 300
	}
	for {
		if probe() {
			return nil
		}
		select {
		case <-server.ctx.Done():
			return server.ctx.Err()
		case err := <-process.exit:
			if err == nil {
				return fmt.Errorf("process exited")
			}
			return fmt.Errorf("process exited: %w", err)
		case <-deadline.C:
			return fmt.Errorf("readiness timed out after %s", server.options.startupTimeout)
		case <-ticker.C:
		}
	}
}

func (server *LlamaServer) supervise(process *managedProcess) {
	defer close(server.done)
	restartFailures := 0
	for {
		select {
		case <-server.ctx.Done():
			server.terminate(process)
			server.setState("stopped")
			return
		case err := <-process.exit:
			if server.ctx.Err() != nil {
				server.setState("stopped")
				return
			}
			server.logger.Error("llama-server exited unexpectedly", "component", "llama-server", "exitCode", processExitCode(err), "error", err, "stderr", server.stderrSnapshot())
		}

		for {
			if restartFailures >= server.maxRestarts {
				server.setState("failed")
				server.logger.Error("llama-server restart limit reached", "component", "llama-server", "failures", restartFailures)
				return
			}
			server.setState("restarting")
			if !server.waitBackoff(server.restartDelay(restartFailures)) {
				server.setState("stopped")
				return
			}
			restartFailures++
			next, err := server.startProcess()
			if err != nil {
				server.logger.Error("restart llama-server", "component", "llama-server", "error", err, "attempt", restartFailures)
				continue
			}
			process = next
			server.setProcess(process)
			if err := server.waitReady(process); err != nil {
				if server.ctx.Err() != nil {
					server.terminate(process)
					server.setState("stopped")
					return
				}
				server.logger.Error("restarted llama-server did not become ready", "component", "llama-server", "error", err, "attempt", restartFailures, "stderr", server.stderrSnapshot())
				server.terminate(process)
				continue
			}
			server.setState("ready")
			restartFailures = 0
			break
		}
	}
}

func processExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	return -1
}

func (server *LlamaServer) restartDelay(attempt int) time.Duration {
	delay := server.options.backoffBase
	for index := 0; index < attempt && delay < server.options.backoffCap; index++ {
		delay *= 2
		if delay > server.options.backoffCap {
			delay = server.options.backoffCap
		}
	}
	return delay
}

func (server *LlamaServer) waitBackoff(delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-server.ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (server *LlamaServer) setProcess(process *managedProcess) {
	server.mu.Lock()
	server.process = process
	server.mu.Unlock()
}

func (server *LlamaServer) setState(state string) {
	server.mu.Lock()
	server.state = state
	server.mu.Unlock()
}

func (server *LlamaServer) Healthy() bool { return server.State() == "ready" }

func (server *LlamaServer) State() string {
	if server == nil {
		return "stopped"
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.state
}

func (server *LlamaServer) terminate(process *managedProcess) {
	if process == nil || process.command == nil || process.command.Process == nil {
		return
	}
	_ = process.command.Process.Signal(syscall.SIGTERM)
	timer := time.NewTimer(server.options.stopTimeout)
	defer timer.Stop()
	select {
	case <-process.exit:
		return
	case <-timer.C:
		_ = process.command.Process.Kill()
		<-process.exit
	}
}

func (server *LlamaServer) Stop() error {
	if server == nil {
		return nil
	}
	server.stopOnce.Do(func() {
		server.cancel()
		server.mu.Lock()
		process := server.process
		server.mu.Unlock()
		server.terminate(process)
		select {
		case <-server.done:
		case <-time.After(server.options.stopTimeout):
		}
		server.setState("stopped")
	})
	return nil
}
