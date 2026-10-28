package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"testing"
)

func logLine(t *testing.T, opts Options, f func(*slog.Logger)) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	f(New(&buf, opts))
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("not one JSON object: %q: %v", buf.String(), err)
	}
	return m
}

var opts = Options{Level: slog.LevelDebug, Application: "security-authz-service", Environment: "stg", Host: "pod-1"}

func TestLogstashShape(t *testing.T) {
	m := logLine(t, opts, func(l *slog.Logger) { l.Info("event applied", "org", "org-a", "panicFlag", true) })
	want := map[string]any{
		"@version": float64(1), "application": "security-authz-service", "environment": "stg", "host": "pod-1",
		"message": "event applied", "level": "INFO", "org": "org-a", "panicFlag": true,
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s: want %v, got %v", k, v, m[k])
		}
	}
	if ts, _ := m["@timestamp"].(string); !regexp.MustCompile(`^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{3}Z$`).MatchString(ts) {
		t.Errorf("@timestamp must be an ISO-8601 UTC instant with milliseconds, got %v", m["@timestamp"])
	}
	for _, k := range []string{"time", "msg", "traceID"} {
		if _, ok := m[k]; ok {
			t.Errorf("unexpected field %q", k)
		}
	}
}

func TestLevelsAreUpperCase(t *testing.T) {
	for name, f := range map[string]func(*slog.Logger){
		"DEBUG": func(l *slog.Logger) { l.Debug("x") },
		"INFO":  func(l *slog.Logger) { l.Info("x") },
		"WARN":  func(l *slog.Logger) { l.Warn("x") },
		"ERROR": func(l *slog.Logger) { l.Error("x") },
	} {
		if got := logLine(t, opts, f)["level"]; got != name {
			t.Errorf("want %s, got %v", name, got)
		}
	}
}

func TestTraceIDFromContext(t *testing.T) {
	ctx := WithTraceID(context.Background(), "abc-123")
	m := logLine(t, opts, func(l *slog.Logger) { l.InfoContext(ctx, "denied") })
	if m["traceID"] != "abc-123" {
		t.Fatalf("want traceID abc-123, got %v", m["traceID"])
	}
	m = logLine(t, opts, func(l *slog.Logger) { l.With("k", "v").InfoContext(ctx, "denied") })
	if m["traceID"] != "abc-123" || m["k"] != "v" {
		t.Fatalf("traceID must survive With: %v", m)
	}
}

func TestLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, Options{Level: slog.LevelWarn})
	l.Info("hidden")
	l.Warn("shown")
	if strings.Contains(buf.String(), "hidden") || !strings.Contains(buf.String(), "shown") {
		t.Fatalf("level filter: %q", buf.String())
	}
}

func TestParseLevel(t *testing.T) {
	for in, want := range map[string]slog.Level{
		"DEBUG": slog.LevelDebug, "verbose": slog.LevelDebug, "silly": slog.LevelDebug, "info": slog.LevelInfo,
		"WARN": slog.LevelWarn, "error": slog.LevelError, "fatal": slog.LevelError,
	} {
		if got, err := ParseLevel(in); err != nil || got != want {
			t.Errorf("%s: got %v %v", in, got, err)
		}
	}
	if _, err := ParseLevel("loud"); err == nil {
		t.Error("unknown level must be an error")
	}
}

func TestNewTraceID(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	a, b := NewTraceID(), NewTraceID()
	if !re.MatchString(a) || a == b {
		t.Fatalf("want distinct UUID v4 ids, got %s %s", a, b)
	}
}
