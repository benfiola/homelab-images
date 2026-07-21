package internal

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/benfiola/homelab-images/shared/pkg/process"
)

// ErrNoAudioStream indicates a file has no audio streams to normalize.
var ErrNoAudioStream = errors.New("no audio stream")

// AudioStream describes the audio stream chosen for normalization.
type AudioStream struct {
	Index string
	Codec string
	Rank  int // 0-based position among audio streams, for ffmpeg's a:N specifiers
	Count int // total number of audio streams in the file
}

// SelectPrimaryAudio picks the file's default audio stream, falling back to
// the first audio stream if none is marked default. Returns ErrNoAudioStream
// if the file has no audio streams.
func SelectPrimaryAudio(ctx context.Context, file string) (*AudioStream, error) {
	out, err := process.Output(ctx, []string{
		"ffprobe", "-v", "error",
		"-select_streams", "a",
		"-show_entries", "stream=index,codec_name:stream_disposition=default",
		"-of", "csv=p=0",
		file,
	})
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, ErrNoAudioStream
	}

	rows, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe audio stream output: %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrNoAudioStream
	}

	count := len(rows)
	for rank, row := range rows {
		if len(row) < 3 {
			continue
		}
		if row[2] == "1" {
			return &AudioStream{Index: row[0], Codec: row[1], Rank: rank, Count: count}, nil
		}
	}

	first := rows[0]
	if len(first) < 2 {
		return nil, fmt.Errorf("unexpected ffprobe audio stream output: %q", out)
	}
	return &AudioStream{Index: first[0], Codec: first[1], Rank: 0, Count: count}, nil
}

// SaveOriginalAudio extracts the audio stream at rank from file into
// backupPath, verbatim (no re-encode).
func SaveOriginalAudio(ctx context.Context, file string, rank int, backupPath string) error {
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return err
	}
	_, err := process.Output(ctx, []string{
		"ffmpeg", "-hide_banner", "-nostats", "-y",
		"-i", file,
		"-map", fmt.Sprintf("0:a:%d", rank),
		"-c:a", "copy",
		backupPath,
	})
	return err
}

// LoudnessMeasurement holds the loudnorm filter's first-pass measurement.
type LoudnessMeasurement struct {
	InputI       string `json:"input_i"`
	InputTP      string `json:"input_tp"`
	InputLRA     string `json:"input_lra"`
	InputThresh  string `json:"input_thresh"`
	TargetOffset string `json:"target_offset"`
}

func loudnormFilter(targetI, targetTP, targetLRA string) string {
	return fmt.Sprintf("I=%s:TP=%s:LRA=%s", targetI, targetTP, targetLRA)
}

// MeasureLoudness runs the loudnorm filter's analysis pass over the selected
// audio stream and returns the measured values used to drive the apply pass.
func MeasureLoudness(ctx context.Context, file string, rank int, targetI, targetTP, targetLRA string) (*LoudnessMeasurement, error) {
	out, err := process.CombinedOutput(ctx, []string{
		"ffmpeg", "-hide_banner", "-nostats",
		"-i", file,
		"-map", fmt.Sprintf("0:a:%d", rank),
		"-af", fmt.Sprintf("loudnorm=%s:print_format=json", loudnormFilter(targetI, targetTP, targetLRA)),
		"-f", "null", "-",
	})
	if err != nil {
		return nil, fmt.Errorf("ffmpeg measure pass failed: %w", err)
	}

	idx := strings.LastIndex(out, "{")
	if idx < 0 {
		return nil, fmt.Errorf("no loudnorm measurement found in ffmpeg output")
	}

	var m LoudnessMeasurement
	if err := json.NewDecoder(strings.NewReader(out[idx:])).Decode(&m); err != nil {
		return nil, fmt.Errorf("failed to parse loudnorm measurement: %w", err)
	}
	if m.InputI == "" || m.InputTP == "" || m.InputLRA == "" {
		return nil, fmt.Errorf("incomplete loudnorm measurement")
	}
	return &m, nil
}

// ApplyParams holds the inputs for the loudnorm apply (second) pass.
type ApplyParams struct {
	File    string // primary input: video/subtitle/data/other streams, and the audio source when AudioSource == ""
	TmpPath string

	// AudioSource, if set, supplies the primary audio stream instead of File.
	AudioSource     string
	AudioSourceRank int

	AudioStreamCount int // total audio streams in File
	PrimaryRank      int // which of those (0-based) is being replaced/re-encoded

	Encoder, Bitrate             string
	TargetI, TargetTP, TargetLRA string
	Measurement                  *LoudnessMeasurement
}

// ApplyLoudnorm re-encodes the primary audio stream to the target loudness
// (from AudioSource if set, otherwise from File itself), copying every
// other stream through untouched. Metadata is preserved via -map_metadata;
// the processing marker itself is a separate sidecar file (see marker.go).
func ApplyLoudnorm(ctx context.Context, p ApplyParams) error {
	usingSecondInput := p.AudioSource != "" && p.AudioSource != p.File

	args := []string{"ffmpeg", "-hide_banner", "-nostats", "-y", "-i", p.File}
	if usingSecondInput {
		args = append(args, "-i", p.AudioSource)
	}

	args = append(args, "-map", "0:v?")
	for i := 0; i < p.AudioStreamCount; i++ {
		if i == p.PrimaryRank && usingSecondInput {
			args = append(args, "-map", fmt.Sprintf("1:a:%d", p.AudioSourceRank))
		} else {
			args = append(args, "-map", fmt.Sprintf("0:a:%d", i))
		}
	}
	args = append(args, "-map", "0:s?", "-map", "0:d?", "-map", "0:t?")
	args = append(args, "-c:v", "copy", "-c:s", "copy", "-c:d", "copy", "-c:t", "copy", "-c:a", "copy")

	audioSpec := fmt.Sprintf("a:%d", p.PrimaryRank)
	filter := fmt.Sprintf(
		"loudnorm=%s:measured_I=%s:measured_TP=%s:measured_LRA=%s:measured_thresh=%s:offset=%s:linear=true:print_format=summary",
		loudnormFilter(p.TargetI, p.TargetTP, p.TargetLRA),
		p.Measurement.InputI, p.Measurement.InputTP, p.Measurement.InputLRA,
		p.Measurement.InputThresh, p.Measurement.TargetOffset,
	)
	args = append(args,
		"-c:"+audioSpec, p.Encoder, "-b:"+audioSpec, p.Bitrate,
		"-filter:"+audioSpec, filter,
		"-map_metadata", "0",
		p.TmpPath,
	)

	if _, err := process.Output(ctx, args); err != nil {
		return err
	}

	info, err := os.Stat(p.TmpPath)
	if err != nil {
		return fmt.Errorf("ffmpeg apply pass produced no output: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("ffmpeg apply pass produced an empty output file")
	}
	return nil
}
