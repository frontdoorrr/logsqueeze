package parser

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/frontdoorrr/logsqueeze/drain"
)

var (
	// 2024-01-01T14:22:11Z or 2024-01-01 14:22:11 or 2024/01/01 14:22:11
	reTimestamp = regexp.MustCompile(`^(\d{4}[-/]\d{2}[-/]\d{2}[T ]\d{2}:\d{2}:\d{2}(?:[Z,][\d+:-]*)?)\s*`)
	reLevel     = regexp.MustCompile(`(?i)^(?:\[?(TRACE|DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|CRITICAL)\]?[:\s]+)`)
	reLevelOnly = regexp.MustCompile(`(?i)^(?:\[?(TRACE|DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|CRITICAL)\]?[:\s]+)`)
)

// Parse converts a raw log line into a LogLine with Message, Level, and Timestamp.
func Parse(raw string) drain.LogLine {
	raw = strings.TrimRight(raw, "\r\n")
	if raw == "" {
		return drain.LogLine{}
	}

	// Try JSON first
	if raw[0] == '{' {
		if ll, ok := parseJSON(raw); ok {
			return ll
		}
	}

	return parseText(raw)
}

// ParseAll parses a slice of raw strings.
func ParseAll(lines []string) []drain.LogLine {
	out := make([]drain.LogLine, 0, len(lines))
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		out = append(out, Parse(l))
	}
	return out
}

func parseJSON(raw string) (drain.LogLine, bool) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return drain.LogLine{}, false
	}

	ll := drain.LogLine{}
	ll.Timestamp = jsonStr(m, "time", "timestamp", "ts", "@timestamp")
	ll.Level = jsonStr(m, "level", "severity", "lvl", "log.level")
	ll.Message = jsonStr(m, "msg", "message", "log", "text")

	if ll.Message == "" {
		// fallback: use whole JSON as message
		ll.Message = raw
	}
	return ll, true
}

func jsonStr(m map[string]json.RawMessage, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			var s string
			if err := json.Unmarshal(v, &s); err == nil {
				return s
			}
		}
	}
	return ""
}

func parseText(raw string) drain.LogLine {
	ll := drain.LogLine{}
	rest := raw

	// extract timestamp
	if loc := reTimestamp.FindStringIndex(rest); loc != nil {
		ll.Timestamp = strings.TrimSpace(rest[loc[0]:loc[1]])
		rest = rest[loc[1]:]
	}

	// extract level
	if m := reLevel.FindStringSubmatch(rest); m != nil {
		ll.Level = strings.ToUpper(m[1])
		rest = rest[len(m[0]):]
	} else if ll.Timestamp == "" {
		// no timestamp, try level at start
		if m2 := reLevelOnly.FindStringSubmatch(rest); m2 != nil {
			ll.Level = strings.ToUpper(m2[1])
			rest = rest[len(m2[0]):]
		}
	}

	ll.Message = strings.TrimSpace(rest)
	if ll.Message == "" {
		ll.Message = raw
	}
	return ll
}
