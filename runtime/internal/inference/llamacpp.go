package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type LlamaCppProvider struct {
	endpoint string
	model    string
	client   *http.Client
}

func NewLlamaCppProvider(endpoint, model string) *LlamaCppProvider {
	return &LlamaCppProvider{endpoint: strings.TrimRight(endpoint, "/"), model: model, client: &http.Client{Timeout: 2 * time.Minute}}
}
func (provider *LlamaCppProvider) Name() string { return "llamacpp" }
func (provider *LlamaCppProvider) Health(ctx context.Context) error {
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

type LlamaServer struct {
	command  *exec.Cmd
	Provider *LlamaCppProvider
}

func StartLlamaServer(ctx context.Context, executable, modelPath string, port int) (*LlamaServer, error) {
	command := exec.CommandContext(ctx, executable, "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--model", modelPath)
	command.Stdout = nil
	command.Stderr = nil
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start llama-server: %w", err)
	}
	server := &LlamaServer{command: command, Provider: NewLlamaCppProvider(fmt.Sprintf("http://127.0.0.1:%d", port), modelPath)}
	deadline := time.Now().Add(45 * time.Second)
	client := &http.Client{Timeout: time.Second}
	for time.Now().Before(deadline) {
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", port), nil)
		response, err := client.Do(request)
		if err == nil {
			response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				return server, nil
			}
		}
		select {
		case <-ctx.Done():
			server.Stop()
			return nil, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	server.Stop()
	return nil, fmt.Errorf("llama-server did not become ready")
}
func (server *LlamaServer) Stop() error {
	if server == nil || server.command == nil || server.command.Process == nil {
		return nil
	}
	_ = server.command.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- server.command.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		return server.command.Process.Kill()
	}
}
