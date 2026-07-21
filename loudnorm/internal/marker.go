package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

var encoderFor = map[string]string{"aac": "aac", "ac3": "ac3", "eac3": "eac3", "mp3": "libmp3lame"}
var bitrateFor = map[string]string{"aac": "256k", "ac3": "640k", "eac3": "768k", "mp3": "320k"}

// EncoderForCodec returns the ffmpeg encoder and bitrate to re-encode the given
// audio codec to, or ok=false if the codec isn't supported.
func EncoderForCodec(codec string) (encoder, bitrate string, ok bool) {
	encoder, ok = encoderFor[codec]
	if !ok {
		return "", "", false
	}
	return encoder, bitrateFor[codec], true
}

// Fingerprint captures the settings that determine whether a file needs
// (re)processing. Any change to it invalidates every marker written under
// the previous fingerprint.
type Fingerprint struct {
	TargetI   string `json:"target_i"`
	TargetTP  string `json:"target_tp"`
	TargetLRA string `json:"target_lra"`
	Salt      string `json:"salt"`
}

// Hash returns a short, stable identifier for the fingerprint.
func (f Fingerprint) Hash() string {
	data, _ := json.Marshal(f)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

// Marker builds a marker tag value embedding the fingerprint's hash and the
// given descriptive fields (e.g. {"i": ..., "codec": ...} for an applied
// marker, or {"skip": reason} for a skip marker).
func (f Fingerprint) Marker(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("v")
	b.WriteString(f.Hash())
	for _, k := range keys {
		b.WriteString(";")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(fields[k])
	}
	return b.String()
}

// ParseMarkerHash extracts the hash portion of a marker value written by Marker.
func ParseMarkerHash(marker string) (hash string, ok bool) {
	if !strings.HasPrefix(marker, "v") {
		return "", false
	}
	rest := marker[1:]
	if i := strings.IndexByte(rest, ';'); i >= 0 {
		rest = rest[:i]
	}
	if rest == "" {
		return "", false
	}
	return rest, true
}

// IsCurrent reports whether marker was written under this fingerprint.
func (f Fingerprint) IsCurrent(marker string) bool {
	hash, ok := ParseMarkerHash(marker)
	return ok && hash == f.Hash()
}
