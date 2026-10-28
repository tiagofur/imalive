package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("writes JSON records with level and structured fields", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(&buf, slog.LevelInfo)

		logger.Info("request handled", slog.String("request_id", "abc123"), slog.Int("status", 200))

		var record map[string]any
		if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
			t.Fatalf("log line is not valid JSON: %v (%q)", err, buf.String())
		}
		if got := record["level"]; got != "INFO" {
			t.Errorf("level = %v, want INFO", got)
		}
		if got := record["msg"]; got != "request handled" {
			t.Errorf("msg = %v, want %q", got, "request handled")
		}
		if got := record["request_id"]; got != "abc123" {
			t.Errorf("request_id = %v, want %q", got, "abc123")
		}
		if _, ok := record["time"]; !ok {
			t.Error("record is missing the time field")
		}
	})

	t.Run("suppresses records below the configured level", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(&buf, slog.LevelWarn)

		logger.Debug("noisy detail")
		logger.Info("still below warn")
		logger.Warn("visible")

		var record map[string]any
		if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
			t.Fatalf("log line is not valid JSON: %v (%q)", err, buf.String())
		}
		if record["msg"] != "visible" {
			t.Errorf("only the warn record should be written, got %v", record["msg"])
		}
	})
}
