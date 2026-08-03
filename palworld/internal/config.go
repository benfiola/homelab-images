package internal

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed webassets/default-settings.ini
var defaultSettings string

// HistoryEntry describes one past config snapshot.
type HistoryEntry struct {
	Filename    string
	DisplayTime string
	Diff        []FieldDiff // non-protected settings that would change if restored, vs. current
	DiffMore    int         // count of additional changes beyond what's in Diff
	IsCurrent   bool
}

// FieldDiff describes one non-protected setting that differs between two
// snapshots.
type FieldDiff struct {
	Key      string
	OldValue string
	NewValue string
}

const maxDiffEntries = 8

// diffSettings returns the non-protected keys whose values differ between
// fromContent and toContent, sorted by key - "what would change if I
// restored this" when fromContent is current.
func diffSettings(fromContent, toContent string) ([]FieldDiff, error) {
	fromKvs, err := Parse(fromContent)
	if err != nil {
		return nil, err
	}
	toKvs, err := Parse(toContent)
	if err != nil {
		return nil, err
	}
	fromKvs = Without(fromKvs, ProtectedKeys...)
	toKvs = Without(toKvs, ProtectedKeys...)

	fromMap := make(map[string]string, len(fromKvs))
	seen := map[string]bool{}
	var keys []string
	for _, kv := range fromKvs {
		fromMap[kv.Key] = kv.Value
		if !seen[kv.Key] {
			seen[kv.Key] = true
			keys = append(keys, kv.Key)
		}
	}
	toMap := make(map[string]string, len(toKvs))
	for _, kv := range toKvs {
		toMap[kv.Key] = kv.Value
		if !seen[kv.Key] {
			seen[kv.Key] = true
			keys = append(keys, kv.Key)
		}
	}
	sort.Strings(keys)

	var diffs []FieldDiff
	for _, key := range keys {
		fv, fok := fromMap[key]
		tv, tok := toMap[key]
		if fok && tok && fv == tv {
			continue
		}
		diffs = append(diffs, FieldDiff{Key: key, OldValue: fv, NewValue: tv})
	}
	return diffs, nil
}

// The history directory is the single source of truth for "what settings
// should be live" - not a separate file we try to keep in sync. Every save
// appends a new timestamped entry; "current" is whichever sorts newest. The
// live ini path is a pure runtime artifact, rewritten from the newest entry
// before every launch cycle (see ReassertLive) - PalServer's own REST API
// save/shutdown flow rewrites that file from its stale in-memory state as a
// side effect of shutting down, so nothing durable can live only there.
func historyDir(dataPath string) string {
	return filepath.Join(dataPath, ".palworld-mgmt", "history")
}

func livePath(dataPath string) string {
	return filepath.Join(dataPath, "Config", "LinuxServer", "PalWorldSettings.ini")
}

func writeAtomic(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func historyFilenames(dataPath string) ([]string, error) {
	entries, err := os.ReadDir(historyDir(dataPath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names) // zero-padded nanosecond timestamps: lexicographic == chronological
	return names, nil
}

func writeHistoryEntry(dataPath, content string) error {
	dir := historyDir(dataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	name := fmt.Sprintf("%020d.ini", time.Now().UnixNano())
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
}

func pruneHistory(dataPath string, limit int) error {
	names, err := historyFilenames(dataPath)
	if err != nil {
		return err
	}
	if len(names) <= limit {
		return nil
	}
	for _, name := range names[:len(names)-limit] {
		if err := os.Remove(filepath.Join(historyDir(dataPath), name)); err != nil {
			return err
		}
	}
	return nil
}

// Current returns the newest history entry, or the embedded default if
// nothing's been saved yet (EnsureBootstrapped normally beats this to it).
func Current(dataPath string) (string, error) {
	names, err := historyFilenames(dataPath)
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return defaultSettings, nil
	}
	content, err := os.ReadFile(filepath.Join(historyDir(dataPath), names[len(names)-1]))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// EnsureBootstrapped writes the embedded default config (with protected
// fields reconciled from env) as the first history entry, if nothing's been
// saved yet. Without this, PalServer's first launch would use its own blank
// engine defaults - notably RESTAPIEnabled=False - breaking Reboot() until
// an admin happened to save once through the UI.
func EnsureBootstrapped(dataPath string, env []string) error {
	names, err := historyFilenames(dataPath)
	if err != nil {
		return err
	}
	if len(names) > 0 {
		return nil
	}
	kvs, err := Parse(defaultSettings)
	if err != nil {
		return fmt.Errorf("parse embedded default settings: %w", err)
	}
	kvs = ReconcileProtectedFields(kvs, env)
	return writeHistoryEntry(dataPath, Render(kvs))
}

// Read returns the current settings, parsed.
func Read(dataPath string) ([]KV, error) {
	content, err := Current(dataPath)
	if err != nil {
		return nil, err
	}
	return Parse(content)
}

func tzLocation() *time.Location {
	tz := os.Getenv("TZ")
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

// Save reconciles edited (the non-protected keys submitted through the
// editor) with the current env-derived protected fields and appends the
// result as a new history entry - the new "current."
func Save(dataPath string, edited []KV, env []string, historyLimit int) error {
	kvs := ReconcileProtectedFields(edited, env)
	if err := writeHistoryEntry(dataPath, Render(kvs)); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return pruneHistory(dataPath, historyLimit)
}

// ListHistory returns past snapshots, newest first. Each entry's Diff shows
// what would change if it were restored, relative to current - not a raw
// content preview, which often shows nothing actually different.
func ListHistory(dataPath string) ([]HistoryEntry, error) {
	names, err := historyFilenames(dataPath)
	if err != nil {
		return nil, err
	}

	current, err := Current(dataPath)
	if err != nil {
		return nil, err
	}

	loc := tzLocation()
	out := make([]HistoryEntry, 0, len(names))
	for i, name := range names {
		nanos, err := strconv.ParseInt(strings.TrimSuffix(name, ".ini"), 10, 64)
		if err != nil {
			continue
		}

		isCurrent := i == len(names)-1
		var diff []FieldDiff
		var diffMore int
		if !isCurrent {
			content, err := os.ReadFile(filepath.Join(historyDir(dataPath), name))
			if err == nil {
				if full, err := diffSettings(current, string(content)); err == nil {
					diff = full
					if len(diff) > maxDiffEntries {
						diffMore = len(diff) - maxDiffEntries
						diff = diff[:maxDiffEntries]
					}
				}
			}
		}

		out = append(out, HistoryEntry{
			Filename:    name,
			DisplayTime: time.Unix(0, nanos).In(loc).Format("Jan 2, 2006 3:04 PM MST"),
			Diff:        diff,
			DiffMore:    diffMore,
			IsCurrent:   isCurrent,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Filename > out[j].Filename })
	return out, nil
}

// RestoreHistory restores a past snapshot's non-protected settings via Save
// - protected fields are discarded, current env vars always win over
// whatever was deployed when the snapshot was taken. A restore is itself a
// new entry, so it's automatically undoable like any other save.
func RestoreHistory(dataPath string, filename string, env []string, historyLimit int) error {
	path := filepath.Join(historyDir(dataPath), filepath.Base(filename))
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}
	kvs, err := Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse snapshot: %w", err)
	}
	kvs = Without(kvs, ProtectedKeys...)
	return Save(dataPath, kvs, env, historyLimit)
}

// ReassertLive rewrites the live ini path from the current history entry.
// Called before every launch cycle - see historyDir's comment for why the
// live path can't be trusted to keep whatever was last written to it.
func ReassertLive(dataPath string) error {
	content, err := Current(dataPath)
	if err != nil {
		return err
	}
	return writeAtomic(livePath(dataPath), content)
}
