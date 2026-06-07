package drain

import (
	"strings"
	"testing"
)

func lines(msgs ...string) []LogLine {
	out := make([]LogLine, len(msgs))
	for i, m := range msgs {
		out[i] = LogLine{Message: m}
	}
	return out
}

func TestBasicCompression(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BatchSize = 10

	input := lines(
		"worker ready shard=1",
		"worker ready shard=2",
		"worker ready shard=3",
		"worker ready shard=4",
		"worker ready shard=5",
	)

	r := Analyze(input, cfg)
	if r.OriginalCount != 5 {
		t.Fatalf("expected 5 original, got %d", r.OriginalCount)
	}
	if r.TemplateCount > 2 {
		t.Fatalf("expected ≤2 templates, got %d", r.TemplateCount)
	}

	found := false
	for _, g := range r.Groups {
		if strings.Contains(g.Template, "worker ready") && strings.Contains(g.Template, wildcard) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a 'worker ready shard=<*>' template, got: %+v", r.Groups)
	}
}

func TestDistinctMessages(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BatchSize = 10
	input := lines(
		"server started",
		"database connected",
		"cache warmed",
	)
	r := Analyze(input, cfg)
	if r.TemplateCount != 3 {
		t.Fatalf("expected 3 distinct templates, got %d", r.TemplateCount)
	}
}

func TestNumericSlotSummary(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BatchSize = 100
	var msgs []string
	for i := 10; i <= 100; i += 10 {
		msgs = append(msgs, "latency "+strings.Repeat("", 0)+strings.Replace("NNNms", "NNN", strings.TrimSpace(strings.Repeat("0", 0)+string(rune('0'+i/10))), 1))
	}
	// simpler: just use literal
	input := lines(
		"pool acquire 20ms",
		"pool acquire 40ms",
		"pool acquire 60ms",
		"pool acquire 80ms",
		"pool acquire 100ms",
		"pool acquire 200ms",
	)
	r := Analyze(input, cfg)
	if r.TemplateCount > 2 {
		t.Fatalf("expected ≤2 templates, got %d", r.TemplateCount)
	}
	for _, g := range r.Groups {
		if strings.Contains(g.Template, "pool acquire") {
			for _, s := range g.Slots {
				if s.Numeric {
					if s.Unit != "ms" {
						t.Errorf("expected unit 'ms', got %q", s.Unit)
					}
					return
				}
			}
		}
	}
}

func TestRender(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BatchSize = 100
	input := lines(
		"worker ready shard=1",
		"worker ready shard=2",
		"worker ready shard=48",
	)
	r := Analyze(input, cfg)
	out := Render(r)
	if !strings.Contains(out, "Compressed") {
		t.Error("render missing header")
	}
	if !strings.Contains(out, "[x") {
		t.Error("render missing group line")
	}
}

func TestFormatLargeNum(t *testing.T) {
	cases := []struct{ n int; want string }{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1200000, "1,200,000"},
	}
	for _, c := range cases {
		got := formatLargeNum(c.n)
		if got != c.want {
			t.Errorf("formatLargeNum(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestParseNumeric(t *testing.T) {
	cases := []struct {
		s    string
		val  float64
		unit string
		ok   bool
	}{
		{"240ms", 240, "ms", true},
		{"512MB", 512, "MB", true},
		{"3.14", 3.14, "", true},
		{"timeout", 0, "", false},
		{"45%", 45, "%", true},
	}
	for _, c := range cases {
		v, u, ok := parseNumeric(c.s)
		if ok != c.ok {
			t.Errorf("parseNumeric(%q) ok=%v want %v", c.s, ok, c.ok)
			continue
		}
		if ok && (v != c.val || u != c.unit) {
			t.Errorf("parseNumeric(%q) = (%v,%q), want (%v,%q)", c.s, v, u, c.val, c.unit)
		}
	}
}
