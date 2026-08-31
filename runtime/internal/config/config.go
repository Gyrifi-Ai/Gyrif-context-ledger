package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddress        string
	MetricsAddress     string
	DrainDelay         time.Duration
	DataDirectory      string
	SQLitePath         string
	ObjectsPath        string
	QdrantURL          string
	QdrantCollection   string
	QdrantAPIKey       string
	EvaluationProvider string
	ModelPath          string
	LlamaServerPath    string
	LlamaServerPort    int
	InferenceRestarts  int
	LogLevel           string
}

func Load() (Config, error) {
	data := environment("GYRIFI_DATA_DIR", "/data")
	port, err := strconv.Atoi(environment("GYRIFI_LLAMA_SERVER_PORT", "8081"))
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("invalid GYRIFI_LLAMA_SERVER_PORT")
	}
	restarts, err := strconv.Atoi(environment("GYRIFI_INFERENCE_MAX_RESTARTS", "5"))
	if err != nil || restarts < 1 {
		return Config{}, fmt.Errorf("invalid GYRIFI_INFERENCE_MAX_RESTARTS")
	}
	drainDelay, err := time.ParseDuration(environment("GYRIFI_DRAIN_DELAY", "0s"))
	if err != nil || drainDelay < 0 {
		return Config{}, fmt.Errorf("invalid GYRIFI_DRAIN_DELAY")
	}
	metricsAddress := environment("GYRIFI_METRICS_ADDRESS", "127.0.0.1:9090")
	host, _, err := net.SplitHostPort(metricsAddress)
	if err != nil || (host != "localhost" && (net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback())) {
		return Config{}, fmt.Errorf("GYRIFI_METRICS_ADDRESS must bind to loopback")
	}
	config := Config{HTTPAddress: environment("GYRIFI_HTTP_ADDRESS", ":8080"), MetricsAddress: metricsAddress, DrainDelay: drainDelay, DataDirectory: data, SQLitePath: environment("GYRIFI_SQLITE_PATH", filepath.Join(data, "state.db")), ObjectsPath: environment("GYRIFI_OBJECTS_PATH", filepath.Join(data, "objects")), QdrantURL: environment("GYRIFI_QDRANT_URL", "http://127.0.0.1:6333"), QdrantCollection: environment("GYRIFI_QDRANT_COLLECTION", "gyrifi"), QdrantAPIKey: os.Getenv("GYRIFI_QDRANT_API_KEY"), EvaluationProvider: strings.ToLower(environment("GYRIFI_EVALUATION_PROVIDER", "disabled")), ModelPath: os.Getenv("GYRIFI_MODEL_PATH"), LlamaServerPath: environment("GYRIFI_LLAMA_SERVER_PATH", "llama-server"), LlamaServerPort: port, InferenceRestarts: restarts, LogLevel: strings.ToLower(environment("GYRIFI_LOG_LEVEL", "info"))}
	if config.EvaluationProvider != "disabled" && config.EvaluationProvider != "llamacpp" {
		return Config{}, fmt.Errorf("GYRIFI_EVALUATION_PROVIDER must be disabled or llamacpp")
	}
	if config.EvaluationProvider == "llamacpp" && config.ModelPath == "" {
		return Config{}, fmt.Errorf("GYRIFI_MODEL_PATH is required when local evaluation is enabled")
	}
	return config, nil
}
func environment(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
