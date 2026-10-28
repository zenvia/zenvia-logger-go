package logger

import (
	"log/slog"
	"strings"
	"testing"
)

func env(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

func TestOptionsFromEnvDefaults(t *testing.T) {
	o, err := OptionsFromEnv(env(map[string]string{"HOSTNAME": "pod-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if o.Level != slog.LevelDebug || o.Host != "pod-1" || o.TraceHeader != "X-TraceId" || o.Environment != "" || o.Text || o.Application == "" {
		t.Fatalf("unexpected defaults: %+v", o)
	}
}

func TestOptionsFromEnvOverrides(t *testing.T) {
	o, err := OptionsFromEnv(env(map[string]string{
		"APP_NAME": "api", "NODE_ENV": "stg", "APP_ENV": "ignored", "HOST": "h1", "HOSTNAME": "h2",
		"LOGGING_LEVEL": "warn", "LOGGING_TRACE_HEADER": "X-Request-Id", "LOGGING_FORMATTER_DISABLED": "true",
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := Options{Level: slog.LevelWarn, Application: "api", Environment: "stg", Host: "h1", Text: true, TraceHeader: "X-Request-Id"}
	if o != want {
		t.Fatalf("want %+v, got %+v", want, o)
	}
}

func TestOptionsFromEnvInvalid(t *testing.T) {
	if _, err := OptionsFromEnv(env(map[string]string{"LOGGING_LEVEL": "loud"})); err == nil || !strings.Contains(err.Error(), "LOGGING_LEVEL") {
		t.Fatalf("unknown level must be reported: %v", err)
	}
	if _, err := OptionsFromEnv(env(map[string]string{"LOGGING_FORMATTER_DISABLED": "maybe"})); err == nil {
		t.Fatal("invalid boolean must be reported")
	}
}

func TestTextOutput(t *testing.T) {
	var b strings.Builder
	New(&b, Options{Level: slog.LevelDebug, Application: "api", Text: true}).Info("hello")
	if !strings.Contains(b.String(), "message=hello") || !strings.Contains(b.String(), "level=INFO") {
		t.Fatalf("text output: %q", b.String())
	}
}
