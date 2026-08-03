package internal

import (
	"fmt"
	"sort"
	"strings"
)

// KV is one Key=Value pair from the OptionSettings=(...) block in
// PalWorldSettings.ini. Order is preserved so Parse+Render round-trips
// without reshuffling the file.
type KV struct {
	Key   string
	Value string
}

const optionSettingsMarker = "OptionSettings=("

// Parse locates the OptionSettings=(...) block and splits its body into
// ordered Key=Value pairs. Paren- and quote-aware, so a comma/paren inside a
// quoted value (ServerName="Some, Name") or a nested list value
// (CrossplayPlatforms=(Steam,Xbox)) doesn't split or close the block early.
func Parse(content string) ([]KV, error) {
	start := strings.Index(content, optionSettingsMarker)
	if start == -1 {
		return nil, fmt.Errorf("no OptionSettings block found")
	}
	body := content[start+len(optionSettingsMarker):]

	depth := 1
	inQuotes := false
	segStart := 0
	var segments []string
	i := 0
	for ; i < len(body); i++ {
		c := body[i]
		switch {
		case c == '"':
			inQuotes = !inQuotes
		case inQuotes:
			// nothing to do, comma/paren handling below is skipped while quoted
		case c == '(':
			depth++
		case c == ')':
			depth--
			if depth == 0 {
				segments = append(segments, body[segStart:i])
				segStart = i + 1
			}
		case c == ',' && depth == 1:
			segments = append(segments, body[segStart:i])
			segStart = i + 1
		}
		if depth == 0 {
			break
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unbalanced parens in OptionSettings block")
	}
	if inQuotes {
		return nil, fmt.Errorf("unbalanced quotes in OptionSettings block")
	}

	seen := map[string]bool{}
	kvs := make([]KV, 0, len(segments))
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		seg = strings.Trim(seg, "\r\n")
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		eq := strings.Index(seg, "=")
		if eq == -1 {
			return nil, fmt.Errorf("malformed entry (missing '='): %q", seg)
		}
		key := strings.TrimSpace(seg[:eq])
		value := strings.TrimSpace(seg[eq+1:])
		if key == "" {
			return nil, fmt.Errorf("malformed entry (empty key): %q", seg)
		}
		if seen[key] {
			return nil, fmt.Errorf("duplicate key: %s", key)
		}
		seen[key] = true
		kvs = append(kvs, KV{Key: key, Value: value})
	}
	return kvs, nil
}

// Render rebuilds the ini file content from kvs, collapsing OptionSettings
// to a single physical line - matches how the game itself writes the file,
// so we don't risk the engine's parser handling a reshaped file differently.
func Render(kvs []KV) string {
	pairs := make([]string, len(kvs))
	for i, kv := range kvs {
		pairs[i] = kv.Key + "=" + kv.Value
	}
	var b strings.Builder
	b.WriteString("[/Script/Pal.PalGameWorldSettings]\n")
	b.WriteString(optionSettingsMarker)
	b.WriteString(strings.Join(pairs, ","))
	b.WriteString(")\n")
	return b.String()
}

// RenderPretty formats kvs one Key=Value per line, sorted alphabetically and
// without trailing commas - what the web editor shows, so a missing comma
// can't produce a malformed save. Render adds the commas back for disk.
func RenderPretty(kvs []KV) string {
	sorted := make([]KV, len(kvs))
	copy(sorted, kvs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Key < sorted[j].Key })

	var b strings.Builder
	for _, kv := range sorted {
		b.WriteString(kv.Key)
		b.WriteString("=")
		b.WriteString(kv.Value)
		b.WriteString("\n")
	}
	return b.String()
}

// ParsePretty parses the one-Key=Value-per-line format RenderPretty
// produces (and the web editor submits) - a plain newline split, not
// Parse's comma scanning. A trailing comma is tolerated but not required.
func ParsePretty(content string) ([]KV, error) {
	seen := map[string]bool{}
	var kvs []KV
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(line, ",")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		eq := strings.Index(line, "=")
		if eq == -1 {
			return nil, fmt.Errorf("malformed entry (missing '='): %q", line)
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		if key == "" {
			return nil, fmt.Errorf("malformed entry (empty key): %q", line)
		}
		if seen[key] {
			return nil, fmt.Errorf("duplicate key: %s", key)
		}
		seen[key] = true
		kvs = append(kvs, KV{Key: key, Value: value})
	}
	return kvs, nil
}

// unquote strips a wrapping pair of double quotes, if present. Quotes are
// ini syntax, not part of the value - used when displaying (not editing) a
// value, e.g. the read-only protected-fields panel.
func unquote(v string) string {
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return v[1 : len(v)-1]
	}
	return v
}

// Get returns the value for key, if present.
func Get(kvs []KV, key string) (string, bool) {
	for _, kv := range kvs {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return "", false
}

// Set updates key's value in place if present, otherwise appends it.
func Set(kvs []KV, key, value string) []KV {
	for i, kv := range kvs {
		if kv.Key == key {
			kvs[i].Value = value
			return kvs
		}
	}
	return append(kvs, KV{Key: key, Value: value})
}

// Without returns a copy of kvs with any of the given keys removed.
func Without(kvs []KV, keys ...string) []KV {
	drop := map[string]bool{}
	for _, k := range keys {
		drop[k] = true
	}
	out := make([]KV, 0, len(kvs))
	for _, kv := range kvs {
		if !drop[kv.Key] {
			out = append(out, kv)
		}
	}
	return out
}
