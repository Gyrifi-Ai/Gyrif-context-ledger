package config

import (
	"testing"
	"time"
)

func clearConfigEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"GYRIFI_DATA_DIR", "GYRIFI_HTTP_ADDRESS", "GYRIFI_METRICS_ADDRESS", "GYRIFI_DRAIN_DELAY",
		"GYRIFI_SQLITE_PATH", "GYRIFI_OBJECTS_PATH", "GYRIFI_QDRANT_URL", "GYRIFI_QDRANT_COLLECTION",
		"GYRIFI_QDRANT_API_KEY", "GYRIFI_EVALUATION_PROVIDER", "GYRIFI_MODEL_PATH", "GYRIFI_LLAMA_SERVER_PATH",
		"GYRIFI_LLAMA_SERVER_PORT", "GYRIFI_LOG_LEVEL",
		"GYRIFI_INFERENCE_MAX_RESTARTS",
		"GYRIFI_PREPARE_BATCH_SIZE", "GYRIFI_PREPARE_LEASE",
	} {
		t.Setenv(name, "")
	}
}

func TestOperationalConfigurationDefaults(t *testing.T) {
	clearConfigEnvironment(t)
	value, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if value.MetricsAddress != "127.0.0.1:9090" || value.DrainDelay != 0 || value.InferenceRestarts != 5 || value.PrepareBatchSize != 25 || value.PrepareLease != 2*time.Minute {
		t.Fatalf("operational defaults = %q %s", value.MetricsAddress, value.DrainDelay)
	}
}

func TestOperationalConfigurationValidation(t *testing.T) {
	t.Run("metrics must be loopback", func(t *testing.T) {
		clearConfigEnvironment(t)
		t.Setenv("GYRIFI_METRICS_ADDRESS", ":9090")
		if _, err := Load(); err == nil {
			t.Fatal("expected wildcard metrics bind to fail")
		}
	})
	t.Run("drain delay is a nonnegative duration", func(t *testing.T) {
		clearConfigEnvironment(t)
		t.Setenv("GYRIFI_DRAIN_DELAY", "-1s")
		if _, err := Load(); err == nil {
			t.Fatal("expected negative drain delay to fail")
		}
		t.Setenv("GYRIFI_DRAIN_DELAY", "750ms")
		value, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if value.DrainDelay != 750*time.Millisecond {
			t.Fatalf("drain delay = %s", value.DrainDelay)
		}
	})
	t.Run("inference restarts is positive", func(t *testing.T) {
		clearConfigEnvironment(t)
		t.Setenv("GYRIFI_INFERENCE_MAX_RESTARTS", "0")
		if _, err := Load(); err == nil {
			t.Fatal("expected zero restart limit to fail")
		}
	})
	t.Run("preparation settings are bounded", func(t *testing.T) {
		for name, value := range map[string]string{"GYRIFI_PREPARE_BATCH_SIZE": "0", "GYRIFI_PREPARE_LEASE": "0s"} {
			t.Run(name, func(t *testing.T) {
				clearConfigEnvironment(t)
				t.Setenv(name, value)
				if _, err := Load(); err == nil {
					t.Fatalf("expected %s validation failure", name)
				}
			})
		}
	})
}
