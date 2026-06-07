package parser

import (
	"testing"
)

func TestParseISO(t *testing.T) {
	ll := Parse("2024-01-01T14:22:11Z INFO worker ready shard=1")
	if ll.Timestamp == "" {
		t.Error("expected timestamp")
	}
	if ll.Level != "INFO" {
		t.Errorf("expected INFO, got %q", ll.Level)
	}
	if ll.Message != "worker ready shard=1" {
		t.Errorf("unexpected message: %q", ll.Message)
	}
}

func TestParseSpaceTimestamp(t *testing.T) {
	ll := Parse("2024-01-01 14:22:11 [INFO] worker ready shard=1")
	if ll.Timestamp == "" {
		t.Error("expected timestamp")
	}
	if ll.Level != "INFO" {
		t.Errorf("expected INFO, got %q", ll.Level)
	}
}

func TestParseSlashTimestamp(t *testing.T) {
	ll := Parse("2024/01/01 14:22:11 INFO worker ready")
	if ll.Timestamp == "" {
		t.Error("expected timestamp")
	}
}

func TestParseJSON(t *testing.T) {
	ll := Parse(`{"time":"2024-01-01T14:22:11Z","level":"info","msg":"worker ready"}`)
	if ll.Timestamp == "" {
		t.Error("expected timestamp from JSON")
	}
	if ll.Level != "info" {
		t.Errorf("expected 'info', got %q", ll.Level)
	}
	if ll.Message != "worker ready" {
		t.Errorf("unexpected message: %q", ll.Message)
	}
}

func TestParseJSONAlt(t *testing.T) {
	ll := Parse(`{"timestamp":"2024-01-01T14:22:11Z","severity":"INFO","message":"worker ready"}`)
	if ll.Message != "worker ready" {
		t.Errorf("unexpected message: %q", ll.Message)
	}
	if ll.Level != "INFO" {
		t.Errorf("unexpected level: %q", ll.Level)
	}
}

func TestParseLevelOnly(t *testing.T) {
	ll := Parse("INFO: worker ready shard=1")
	if ll.Level != "INFO" {
		t.Errorf("expected INFO, got %q", ll.Level)
	}
	if ll.Message != "worker ready shard=1" {
		t.Errorf("unexpected message: %q", ll.Message)
	}
}

func TestParseBracketLevel(t *testing.T) {
	ll := Parse("[ERROR] pool acquire 240ms")
	if ll.Level != "ERROR" {
		t.Errorf("expected ERROR, got %q", ll.Level)
	}
}

func TestParseNoFormat(t *testing.T) {
	ll := Parse("worker ready shard=1")
	if ll.Message != "worker ready shard=1" {
		t.Errorf("unexpected message: %q", ll.Message)
	}
}

func TestParseAll(t *testing.T) {
	raw := []string{
		"2024-01-01T14:22:11Z INFO worker ready",
		"",
		"ERROR: something failed",
	}
	out := ParseAll(raw)
	if len(out) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(out))
	}
}
