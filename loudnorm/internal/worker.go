package internal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
)

// Worker runs every file through a single queue so ffmpeg calls stay sequential.
type Worker struct {
	mediaDirs      []string
	fingerprint    Fingerprint
	rescanInterval time.Duration
	backupDir      string // "" disables the original-audio backup feature
	store          *Store

	queue chan string

	mu         sync.Mutex
	inFlight   map[string]struct{}
	currentTmp string
}

func NewWorker(mediaDirs []string, fingerprint Fingerprint, rescanInterval time.Duration, backupDir string, store *Store) *Worker {
	return &Worker{
		mediaDirs:      mediaDirs,
		fingerprint:    fingerprint,
		rescanInterval: rescanInterval,
		backupDir:      backupDir,
		store:          store,
		queue:          make(chan string, 10000),
		inFlight:       map[string]struct{}{},
	}
}

// Enqueue schedules path, deduplicating against anything already queued.
func (w *Worker) Enqueue(path string) {
	w.mu.Lock()
	if _, ok := w.inFlight[path]; ok {
		w.mu.Unlock()
		return
	}
	w.inFlight[path] = struct{}{}
	w.mu.Unlock()

	w.queue <- path
}

func (w *Worker) clearInFlight(path string) {
	w.mu.Lock()
	delete(w.inFlight, path)
	w.mu.Unlock()
}

func (w *Worker) setCurrentTmp(path string) {
	w.mu.Lock()
	w.currentTmp = path
	w.mu.Unlock()
}

// CleanupInFlight removes the tmp file for whatever's currently being written, if any.
func (w *Worker) CleanupInFlight() {
	w.mu.Lock()
	tmp := w.currentTmp
	w.currentTmp = ""
	w.mu.Unlock()

	if tmp != "" {
		os.Remove(tmp)
	}
}

// Run blocks until ctx is cancelled: crash-recovery sweep, the processing
// loop, an initial scan, and (if configured) a periodic rescan.
func (w *Worker) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	if removed := CleanupTmpFiles(ctx, w.mediaDirs); removed > 0 {
		logger.Info("removed orphaned tmp files", "count", removed)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case path := <-w.queue:
				w.processFile(ctx, path)
				w.clearInFlight(path)
			}
		}
	}()

	go w.scanOnce(ctx, "startup")

	if w.rescanInterval > 0 {
		go func() {
			ticker := time.NewTicker(w.rescanInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					w.scanOnce(ctx, "rescan")
				}
			}
		}()
	}

	<-ctx.Done()
	<-done
	return nil
}

func (w *Worker) scanOnce(ctx context.Context, source string) {
	logger := logging.FromContext(ctx)

	files := ScanDirs(ctx, w.mediaDirs)
	logger.Debug("scan complete", "source", source, "files", len(files))
	for _, f := range files {
		w.Enqueue(f)
	}

	known := make(map[string]bool, len(files))
	for _, f := range files {
		known[f] = true
	}
	pruned, err := w.store.PruneOrphans(known)
	if err != nil {
		logger.Warn("failed to prune orphaned markers", "error", err)
		return
	}
	for _, f := range pruned {
		w.removeBackup(ctx, f)
	}
	if len(pruned) > 0 {
		logger.Info("pruned orphaned markers", "count", len(pruned))
	}
}

// HandleDelete removes the marker and backup for a single deleted file.
func (w *Worker) HandleDelete(ctx context.Context, file string) {
	logger := logging.FromContext(ctx)
	if err := w.store.DeleteMarker(file); err != nil {
		logger.Warn("failed to delete marker", "file", file, "error", err)
	}
	w.removeBackup(ctx, file)
}

// HandleDeletePrefix removes every marker and backup under dir.
func (w *Worker) HandleDeletePrefix(ctx context.Context, dir string) {
	logger := logging.FromContext(ctx)
	removed, err := w.store.DeletePrefix(dir)
	if err != nil {
		logger.Warn("failed to delete markers under directory", "dir", dir, "error", err)
		return
	}
	for _, f := range removed {
		w.removeBackup(ctx, f)
	}
	logger.Info("removed markers for deleted directory", "dir", dir, "count", len(removed))
}

func (w *Worker) removeBackup(ctx context.Context, file string) {
	if w.backupDir == "" {
		return
	}
	if err := os.Remove(BackupPathFor(w.backupDir, file)); err != nil && !os.IsNotExist(err) {
		logging.FromContext(ctx).Warn("failed to remove orphaned audio backup", "file", file, "error", err)
	}
}

func (w *Worker) processFile(ctx context.Context, file string) {
	logger := logging.FromContext(ctx)

	marker, err := w.store.GetMarker(file)
	if err != nil {
		logger.Error("failed to read marker", "file", file, "error", err)
		return
	}
	if w.fingerprint.IsCurrent(marker) {
		return
	}

	audio, err := SelectPrimaryAudio(ctx, file)
	if err != nil {
		if errors.Is(err, ErrNoAudioStream) {
			w.markSkip(ctx, file, "no-audio-stream")
			return
		}
		logger.Error("failed to select audio stream", "file", file, "error", err)
		return
	}

	encoder, bitrate, ok := EncoderForCodec(audio.Codec)
	if !ok {
		w.markSkip(ctx, file, "unsupported-codec-"+audio.Codec)
		return
	}

	audioSource, audioSourceRank := file, audio.Rank
	if w.backupDir != "" {
		backupPath := BackupPathFor(w.backupDir, file)
		if _, err := os.Stat(backupPath); err == nil {
			audioSource, audioSourceRank = backupPath, 0
		} else if marker == "" {
			if err := SaveOriginalAudio(ctx, file, audio.Rank, backupPath); err != nil {
				logger.Warn("failed to save original audio backup, continuing without one", "file", file, "error", err)
			}
		} else {
			logger.Warn("no original-audio backup found for a previously-processed file; reprocessing from the current (already re-encoded) audio", "file", file)
		}
	}

	measurement, err := MeasureLoudness(ctx, audioSource, audioSourceRank, w.fingerprint.TargetI, w.fingerprint.TargetTP, w.fingerprint.TargetLRA)
	if err != nil {
		logger.Error("failed to measure loudness", "file", file, "error", err)
		return
	}

	if err := w.applyLoudnorm(ctx, file, audio, audioSource, audioSourceRank, measurement, encoder, bitrate); err != nil {
		logger.Error("failed to apply loudnorm", "file", file, "error", err)
		return
	}
	logger.Info("normalized", "file", file, "track", fmt.Sprintf("a:%d", audio.Rank), "codec", encoder)
}

func (w *Worker) markSkip(ctx context.Context, file, reason string) {
	logger := logging.FromContext(ctx)

	marker := w.fingerprint.Marker(map[string]string{"skip": reason})
	if err := w.store.SetMarker(file, marker); err != nil {
		logger.Error("failed to write skip marker", "file", file, "reason", reason, "error", err)
		return
	}
	logger.Info("skipped", "file", file, "reason", reason)
}

func (w *Worker) applyLoudnorm(ctx context.Context, file string, audio *AudioStream, audioSource string, audioSourceRank int, m *LoudnessMeasurement, encoder, bitrate string) error {
	logger := logging.FromContext(ctx)

	tmp := TmpPathFor(file)
	w.setCurrentTmp(tmp)
	defer w.setCurrentTmp("")

	err := ApplyLoudnorm(ctx, ApplyParams{
		File: file, TmpPath: tmp,
		AudioSource: audioSource, AudioSourceRank: audioSourceRank,
		AudioStreamCount: audio.Count, PrimaryRank: audio.Rank,
		Encoder: encoder, Bitrate: bitrate,
		TargetI: w.fingerprint.TargetI, TargetTP: w.fingerprint.TargetTP, TargetLRA: w.fingerprint.TargetLRA,
		Measurement: m,
	})
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if err := w.replaceAtomic(file, tmp); err != nil {
		return err
	}

	marker := w.fingerprint.Marker(map[string]string{
		"i": w.fingerprint.TargetI, "tp": w.fingerprint.TargetTP, "lra": w.fingerprint.TargetLRA, "codec": encoder,
	})
	if err := w.store.SetMarker(file, marker); err != nil {
		logger.Warn("applied loudnorm but failed to write marker; file will be reprocessed next scan", "file", file, "error", err)
	}
	return nil
}

func (w *Worker) replaceAtomic(orig, tmp string) error {
	if info, err := os.Stat(orig); err == nil {
		_ = os.Chmod(tmp, info.Mode())
	}
	return os.Rename(tmp, orig)
}
