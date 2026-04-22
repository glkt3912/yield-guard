package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func logToMap(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse log JSON: %v\nraw: %s", err, buf.String())
	}
	return m
}

func newTestLogger(buf *bytes.Buffer, projectID string) *slog.Logger {
	return slog.New(newJSONHandler(buf, slog.LevelDebug, projectID))
}

func TestSeverityField_Warn(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf, "")
	l.Warn("test warn")
	m := logToMap(t, &buf)
	if got := m["severity"]; got != "WARNING" {
		t.Errorf("severity = %q, want %q", got, "WARNING")
	}
}

func TestSeverityField_Error(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf, "")
	l.Error("test error")
	m := logToMap(t, &buf)
	if got := m["severity"]; got != "ERROR" {
		t.Errorf("severity = %q, want %q", got, "ERROR")
	}
}

func TestSeverityField_Critical(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf, "")
	l.Log(context.Background(), LevelCritical, "test critical")
	m := logToMap(t, &buf)
	if got := m["severity"]; got != "CRITICAL" {
		t.Errorf("severity = %q, want %q", got, "CRITICAL")
	}
}

func TestSeverityField_Info(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf, "")
	l.Info("test info")
	m := logToMap(t, &buf)
	if got := m["severity"]; got != "INFO" {
		t.Errorf("severity = %q, want %q", got, "INFO")
	}
}

func TestMessageField(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf, "")
	l.Info("hello world")
	m := logToMap(t, &buf)
	if _, ok := m["msg"]; ok {
		t.Error("unexpected 'msg' field; want 'message'")
	}
	if got := m["message"]; got != "hello world" {
		t.Errorf("message = %q, want %q", got, "hello world")
	}
}

func TestTraceInjection_WithValidSpan(t *testing.T) {
	var traceID trace.TraceID
	var spanID trace.SpanID
	traceID[0] = 0xab
	spanID[0] = 0xcd

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	var buf bytes.Buffer
	l := newTestLogger(&buf, "my-project")
	l.InfoContext(ctx, "traced request")
	m := logToMap(t, &buf)

	traceField, ok := m["logging.googleapis.com/trace"]
	if !ok {
		t.Fatal("missing 'logging.googleapis.com/trace' field")
	}
	want := "projects/my-project/traces/" + traceID.String()
	if traceField != want {
		t.Errorf("trace = %q, want %q", traceField, want)
	}

	if _, ok := m["logging.googleapis.com/spanId"]; !ok {
		t.Error("missing 'logging.googleapis.com/spanId' field")
	}
	if sampled, _ := m["logging.googleapis.com/trace_sampled"].(bool); !sampled {
		t.Error("trace_sampled should be true")
	}
}

func TestTraceInjection_WithoutSpan(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf, "my-project")
	l.InfoContext(context.Background(), "untraced request")
	m := logToMap(t, &buf)

	if _, ok := m["logging.googleapis.com/trace"]; ok {
		t.Error("unexpected trace field for context without span")
	}
}

func TestTraceInjection_EmptyProjectID(t *testing.T) {
	var traceID trace.TraceID
	var spanID trace.SpanID
	traceID[0] = 0x01
	spanID[0] = 0x02

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	var buf bytes.Buffer
	l := newTestLogger(&buf, "") // projectID 空
	l.InfoContext(ctx, "no project id")
	m := logToMap(t, &buf)

	if _, ok := m["logging.googleapis.com/trace"]; ok {
		t.Error("unexpected trace field when projectID is empty")
	}
}

func TestParseLevel(t *testing.T) {
	cases := []struct {
		input string
		want  slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"", slog.LevelInfo},
		{"invalid", slog.LevelInfo},
	}
	for _, tc := range cases {
		if got := parseLevel(tc.input); got != tc.want {
			t.Errorf("parseLevel(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
