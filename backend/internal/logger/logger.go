package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

type cloudHandler struct {
	inner     slog.Handler
	projectID string
}

func (h *cloudHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *cloudHandler) Handle(ctx context.Context, r slog.Record) error {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.IsValid() && h.projectID != "" {
		r.AddAttrs(
			slog.String("logging.googleapis.com/trace",
				fmt.Sprintf("projects/%s/traces/%s", h.projectID, sc.TraceID())),
			slog.String("logging.googleapis.com/spanId", sc.SpanID().String()),
			slog.Bool("logging.googleapis.com/trace_sampled", sc.TraceFlags().IsSampled()),
		)
	}
	return h.inner.Handle(ctx, r)
}

func (h *cloudHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &cloudHandler{inner: h.inner.WithAttrs(attrs), projectID: h.projectID}
}

func (h *cloudHandler) WithGroup(name string) slog.Handler {
	return &cloudHandler{inner: h.inner.WithGroup(name), projectID: h.projectID}
}

// LevelCritical は slog にない Cloud Logging の CRITICAL 重大度に対応するカスタムレベル。
// 使用例: slog.Log(ctx, LevelCritical, "unrecoverable error")
const LevelCritical = slog.LevelError + 4

func levelToSeverity(l slog.Level) string {
	switch {
	case l >= LevelCritical:
		return "CRITICAL"
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARNING"
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}

func newJSONHandler(w io.Writer, level slog.Leveler, projectID string) slog.Handler {
	inner := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) != 0 {
				return a
			}
			switch a.Key {
			case slog.LevelKey:
				// slog guarantees LevelKey value is always slog.Level; assertion is safe.
				return slog.String("severity", levelToSeverity(a.Value.Any().(slog.Level))) //nolint:errcheck
			case slog.MessageKey:
				return slog.Attr{Key: "message", Value: a.Value}
			}
			return a
		},
	})
	return &cloudHandler{inner: inner, projectID: projectID}
}

func parseLevel(s string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Init initializes the global slog default logger with Cloud Logging compatible JSON output.
// Reads LOG_LEVEL (DEBUG/INFO/WARN/ERROR, default INFO) and GOOGLE_CLOUD_PROJECT env vars.
func Init(w io.Writer) {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	slog.SetDefault(slog.New(newJSONHandler(w, level, projectID)))
}

// New returns a logger writing to w, using the same Cloud Logging format as Init.
// Reads GOOGLE_CLOUD_PROJECT from the environment for trace field formatting.
// Intended for tests and non-global usage; for test isolation pass projectID via newJSONHandler directly.
func New(w io.Writer) *slog.Logger {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	return slog.New(newJSONHandler(w, slog.LevelDebug, projectID))
}
