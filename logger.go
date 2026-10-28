// Package logger writes logs in the format of @zenvia/logger (Logstash JSON), the same one the Node services
// send to Datadog: @timestamp, @version, application, message, upper-case level, environment, host and traceID.
package logger

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

type Options struct {
	Level       slog.Level
	Application string
	Environment string
	Host        string
	// Text writes human-readable key=value lines instead of JSON (LOGGING_FORMATTER_DISABLED=true), for development.
	Text bool
	// TraceHeader is the request header that carries the trace id (LOGGING_TRACE_HEADER, default X-TraceId).
	// New does not use it: servers and middlewares read it to call WithTraceID.
	TraceHeader string
}

// ParseLevel accepts the @zenvia/logger level names. "fatal" maps to error (slog has no fatal) and
// "verbose" and "silly" behave as debug, as in the Node logger.
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug", "verbose", "silly":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error", "fatal":
		return slog.LevelError, nil
	}
	return 0, fmt.Errorf("unknown log level %q", s)
}

// New returns a logger that writes Logstash JSON to w. Records logged with a context that carries a trace id
// (see WithTraceID) get a traceID field.
func New(w io.Writer, opts Options) *slog.Logger {
	ho := &slog.HandlerOptions{Level: opts.Level, ReplaceAttr: replace}
	var h slog.Handler = slog.NewJSONHandler(w, ho)
	if opts.Text {
		h = slog.NewTextHandler(w, ho)
	}
	base := h.WithAttrs([]slog.Attr{
		slog.Int("@version", 1),
		slog.String("application", opts.Application),
		slog.String("environment", opts.Environment),
		slog.String("host", opts.Host),
	})
	return slog.New(traceHandler{base})
}

func replace(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.TimeKey:
		return slog.String("@timestamp", a.Value.Time().UTC().Format("2006-01-02T15:04:05.000Z07:00"))
	case slog.MessageKey:
		return slog.String("message", a.Value.String())
	case slog.LevelKey:
		return slog.String("level", levelName(a))
	}
	return a
}

// levelName returns the base level name without the "+N" offset slog appends to custom levels.
func levelName(a slog.Attr) string {
	l := a.Value.Any().(slog.Level)
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARN"
	case l >= slog.LevelInfo:
		return "INFO"
	}
	return "DEBUG"
}

type traceKey struct{}

// WithTraceID returns a context whose log records carry traceID.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceKey{}, id)
}

func TraceID(ctx context.Context) string {
	id, _ := ctx.Value(traceKey{}).(string)
	return id
}

// NewTraceID generates a random UUID v4, as @zenvia/logger does when the request carries no trace header.
func NewTraceID() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

type traceHandler struct{ slog.Handler }

func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := TraceID(ctx); id != "" {
		r.AddAttrs(slog.String("traceID", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceHandler{h.Handler.WithAttrs(attrs)}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{h.Handler.WithGroup(name)}
}
