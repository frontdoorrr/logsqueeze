package drain

import (
	"math"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const wildcard = "<*>"

type LogLine struct {
	Message   string
	Level     string
	Timestamp string
}

type SlotSummary struct {
	Numeric  bool
	Min, Max float64
	Median   float64
	Unit     string
	Distinct int
	Samples  []string
}

type TemplateGroup struct {
	ID        int
	Template  string
	Count     int
	Samples   []string
	Slots     []SlotSummary
	FirstSeen string
}

type Result struct {
	Groups          []TemplateGroup
	OriginalCount   int
	TemplateCount   int
	LineCompression float64
}

type Config struct {
	Depth         int
	SimTh         float64
	MaxChildren   int
	BatchSize     int
	SampleCap     int
	SlotSampleCap int
}

func DefaultConfig() Config {
	return Config{
		Depth:         4,
		SimTh:         0.4,
		MaxChildren:   100,
		BatchSize:     2000,
		SampleCap:     3,
		SlotSampleCap: 6,
	}
}

type prefixNode struct {
	children map[string]*prefixNode
	groups   []*group
}

type group struct {
	id        int
	tokens    []string
	count     int
	samples   []string
	slots     [][]string   // distinct string samples per token position
	numVals   [][]float64  // all numeric values per token position (for stats)
	numUnits  []map[string]int
	firstSeen string
}

type xdrain struct {
	cfg      Config
	trees    []*prefixNode // one tree per rotation
	nextID   int
	groups   map[int]*group
}

const rotations = 2

func newXDrain(cfg Config) *xdrain {
	trees := make([]*prefixNode, rotations)
	for i := range trees {
		trees[i] = &prefixNode{children: make(map[string]*prefixNode)}
	}
	return &xdrain{cfg: cfg, trees: trees, groups: make(map[int]*group)}
}

func tokenize(msg string) []string { return strings.Fields(msg) }

// rotate returns tokens starting at position start (circular).
func rotate(tokens []string, start int) []string {
	if start == 0 {
		return tokens
	}
	n := len(tokens)
	out := make([]string, n)
	for i, t := range tokens {
		out[(i-start+n)%n] = t
	}
	return out
}

func (x *xdrain) addLine(line LogLine) {
	tokens := tokenize(line.Message)
	if len(tokens) == 0 {
		return
	}

	// Collect votes from each rotation.
	votes := make(map[int]int)
	for r := 0; r < rotations; r++ {
		gid := x.findMatch(x.trees[r], rotate(tokens, r))
		if gid >= 0 {
			votes[gid]++
		}
	}

	// Pick the group with the most votes (or create one via rotation 0).
	winner := -1
	bestVotes := 0
	for gid, v := range votes {
		if v > bestVotes {
			bestVotes = v
			winner = gid
		}
	}

	if winner >= 0 {
		x.mergeGroup(x.groups[winner], tokens, line)
	} else {
		// No match in any rotation: create a new group via rotation-0 tree.
		x.insertNew(x.trees[0], tokens, line)
	}
}

// findMatch walks the tree for the given (possibly rotated) tokens and returns
// the best matching group ID, or -1 if no match exceeds SimTh.
func (x *xdrain) findMatch(root *prefixNode, tokens []string) int {
	node := x.routeToLeaf(root, tokens, false)
	if node == nil {
		return -1
	}
	best, bestSim := -1, -1.0
	for _, g := range node.groups {
		if len(g.tokens) != len(tokens) {
			continue
		}
		sim := similarity(g.tokens, tokens)
		if sim > bestSim {
			bestSim = sim
			best = g.id
		}
	}
	if bestSim >= x.cfg.SimTh {
		return best
	}
	return -1
}

// insertNew creates a new group and inserts it into the rotation-0 tree.
func (x *xdrain) insertNew(root *prefixNode, tokens []string, line LogLine) {
	node := x.routeToLeaf(root, tokens, true)
	id := x.nextID
	x.nextID++

	n := len(tokens)
	slots := make([][]string, n)
	numVals := make([][]float64, n)
	numUnits := make([]map[string]int, n)
	for i := range slots {
		slots[i] = []string{}
		numUnits[i] = map[string]int{}
	}
	g := &group{
		id:        id,
		tokens:    append([]string(nil), tokens...),
		count:     1,
		slots:     slots,
		numVals:   numVals,
		numUnits:  numUnits,
		firstSeen: line.Timestamp,
		samples:   appendSample(nil, line.Message, x.cfg.SampleCap),
	}
	x.groups[id] = g
	node.groups = append(node.groups, g)
}

// routeToLeaf navigates (and optionally creates) prefix nodes.
// Depth is capped at len(tokens)-1 so the last token stays at the leaf for comparison.
func (x *xdrain) routeToLeaf(root *prefixNode, tokens []string, create bool) *prefixNode {
	lenKey := strconv.Itoa(len(tokens))
	lenNode := root.children[lenKey]
	if lenNode == nil {
		if !create {
			return nil
		}
		lenNode = &prefixNode{children: make(map[string]*prefixNode)}
		root.children[lenKey] = lenNode
	}

	node := lenNode
	// Route on at most depth-1 tokens (leave ≥1 token for leaf comparison).
	maxDepth := x.cfg.Depth - 1
	if maxDepth < 1 {
		maxDepth = 1
	}
	depth := min(maxDepth, len(tokens)-1)

	for i := 0; i < depth; i++ {
		tok := tokens[i]
		child := node.children[tok]
		if child == nil {
			// Route to wildcard if at capacity, or if wildcard already exists.
			if _, hasWild := node.children[wildcard]; hasWild || len(node.children) >= x.cfg.MaxChildren {
				tok = wildcard
				child = node.children[tok]
			}
		}
		if child == nil {
			if !create {
				// Try wildcard fallback before giving up.
				if wc := node.children[wildcard]; wc != nil {
					node = wc
					continue
				}
				return nil
			}
			child = &prefixNode{children: make(map[string]*prefixNode)}
			node.children[tok] = child
		}
		node = child
	}
	return node
}

func (x *xdrain) mergeGroup(g *group, tokens []string, line LogLine) {
	g.count++
	g.samples = appendSample(g.samples, line.Message, x.cfg.SampleCap)
	for i := range g.tokens {
		if i >= len(tokens) {
			break
		}
		if g.tokens[i] == wildcard {
			x.recordSlotVal(g, i, tokens[i])
		} else if g.tokens[i] != tokens[i] {
			x.recordSlotVal(g, i, g.tokens[i])
			x.recordSlotVal(g, i, tokens[i])
			g.tokens[i] = wildcard
		}
	}
}

func (x *xdrain) recordSlotVal(g *group, pos int, val string) {
	g.slots[pos] = appendSample(g.slots[pos], val, x.cfg.SlotSampleCap)
	if n, u, ok := parseNumeric(val); ok {
		g.numVals[pos] = append(g.numVals[pos], n)
		if u != "" {
			g.numUnits[pos][u]++
		}
	}
}

func appendSample(s []string, v string, cap int) []string {
	if len(s) >= cap {
		return s
	}
	for _, e := range s {
		if e == v {
			return s
		}
	}
	return append(s, v)
}

func similarity(a, b []string) float64 {
	n := len(a)
	if n == 0 || n != len(b) {
		return 0
	}
	match := 0
	for i := range a {
		if a[i] == b[i] || a[i] == wildcard || b[i] == wildcard {
			match++
		}
	}
	return float64(match) / float64(n)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Analyze runs XDrain on the input lines and returns a compressed Result.
func Analyze(lines []LogLine, cfg Config) Result {
	x := newXDrain(cfg)

	for start := 0; start < len(lines); start += cfg.BatchSize {
		end := start + cfg.BatchSize
		if end > len(lines) {
			end = len(lines)
		}
		batch := make([]LogLine, end-start)
		copy(batch, lines[start:end])
		rand.Shuffle(len(batch), func(i, j int) { batch[i], batch[j] = batch[j], batch[i] })
		for _, l := range batch {
			x.addLine(l)
		}
	}

	all := make([]*group, 0, len(x.groups))
	for _, g := range x.groups {
		all = append(all, g)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].count > all[j].count })

	groups := make([]TemplateGroup, 0, len(all))
	for _, g := range all {
		tg := TemplateGroup{
			ID:        g.id,
			Template:  strings.Join(g.tokens, " "),
			Count:     g.count,
			Samples:   g.samples,
			FirstSeen: g.firstSeen,
			Slots:     buildSlotSummaries(g, cfg.SlotSampleCap),
		}
		groups = append(groups, tg)
	}

	compression := 0.0
	if len(groups) > 0 {
		compression = float64(len(lines)) / float64(len(groups))
	}
	return Result{
		Groups:          groups,
		OriginalCount:   len(lines),
		TemplateCount:   len(groups),
		LineCompression: compression,
	}
}

func buildSlotSummaries(g *group, cap int) []SlotSummary {
	var summaries []SlotSummary
	for i, tok := range g.tokens {
		if tok != wildcard {
			continue
		}
		vals := g.slots[i]
		nums := g.numVals[i]
		units := g.numUnits[i]
		s := SlotSummary{Distinct: len(vals)}

		// Treat as numeric if ≥half of sampled values are numeric.
		if len(nums) > 0 && len(nums)*2 >= len(vals) {
			s.Numeric = true
			s.Min, s.Max = nums[0], nums[0]
			for _, n := range nums {
				if n < s.Min {
					s.Min = n
				}
				if n > s.Max {
					s.Max = n
				}
			}
			s.Median = median(nums)
			s.Unit = dominantUnit(units)
		} else {
			s.Samples = dedup(vals, cap)
		}
		summaries = append(summaries, s)
	}
	return summaries
}

func parseNumeric(s string) (float64, string, bool) {
	s = strings.TrimSpace(s)
	i := len(s)
	for i > 0 && (unicode.IsLetter(rune(s[i-1])) || s[i-1] == '%') {
		i--
	}
	unit := s[i:]
	n, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0, "", false
	}
	return n, unit, true
}

func median(nums []float64) float64 {
	sorted := make([]float64, len(nums))
	copy(sorted, nums)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func dominantUnit(units map[string]int) string {
	best, bestN := "", 0
	for u, n := range units {
		if n > bestN {
			bestN = n
			best = u
		}
	}
	return best
}

func dedup(vals []string, cap int) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range vals {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
			if len(out) >= cap {
				break
			}
		}
	}
	return out
}

// Render formats a Result as human-readable text for LLM consumption.
func Render(r Result) string {
	var sb strings.Builder

	compressionStr := formatCompression(r.LineCompression)
	sb.WriteString("Compressed ")
	sb.WriteString(formatLargeNum(r.OriginalCount))
	sb.WriteString(" lines → ")
	sb.WriteString(strconv.Itoa(r.TemplateCount))
	sb.WriteString(" templates (")
	sb.WriteString(compressionStr)
	sb.WriteString(" compression)\n")

	for _, g := range r.Groups {
		sb.WriteString("\n[x")
		sb.WriteString(formatLargeNum(g.Count))
		sb.WriteString("] ")
		sb.WriteString(renderTemplate(g))
		sb.WriteString("\n")
		if len(g.Samples) > 0 {
			sb.WriteString("  samples: ")
			parts := make([]string, len(g.Samples))
			for i, s := range g.Samples {
				if g.FirstSeen != "" {
					parts[i] = g.FirstSeen + " " + s
				} else {
					parts[i] = s
				}
			}
			sb.WriteString(strings.Join(parts, " | "))
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func renderTemplate(g TemplateGroup) string {
	slotIdx := 0
	parts := strings.Fields(g.Template)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == wildcard && slotIdx < len(g.Slots) {
			s := g.Slots[slotIdx]
			slotIdx++
			if s.Numeric {
				result = append(result, wildcard+" "+formatNumericSlot(s))
			} else if len(s.Samples) > 0 {
				result = append(result, wildcard+" ["+strings.Join(s.Samples, ",")+"]")
			} else {
				result = append(result, wildcard)
			}
		} else {
			result = append(result, p)
		}
	}
	return strings.Join(result, " ")
}

func formatNumericSlot(s SlotSummary) string {
	if s.Min == s.Max {
		return "[" + formatFloat(s.Min) + s.Unit + "]"
	}
	out := "[" + formatFloat(s.Min) + ".." + formatFloat(s.Max) + s.Unit
	if s.Median != s.Min && s.Median != s.Max {
		out += " p50=" + formatFloat(s.Median) + s.Unit
	}
	return out + "]"
}

func formatCompression(c float64) string {
	if c >= 1000 {
		return formatLargeNum(int(math.Round(c))) + "x"
	}
	return strconv.FormatFloat(c, 'f', 0, 64) + "x"
}

func formatFloat(f float64) string {
	if f == math.Trunc(f) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', 2, 64)
}

func formatLargeNum(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	result := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
