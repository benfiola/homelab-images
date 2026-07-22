package processor

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type Processor struct {
	scratchDir string
	mediaDir   string
	loudnessTarget int
	logger     *slog.Logger
}

type loudnessOutput struct {
	InputI   string `json:"input_i"`
	InputTP  string `json:"input_tp"`
	InputLRA string `json:"input_lra"`
	OutputI  string `json:"output_i"`
	OutputTP string `json:"output_tp"`
}

func New(scratchDir, mediaDir string, loudnessTarget int, logger *slog.Logger) *Processor {
	return &Processor{
		scratchDir:     scratchDir,
		mediaDir:       mediaDir,
		loudnessTarget: loudnessTarget,
		logger:         logger,
	}
}

func (p *Processor) SettingsHash() string {
	data := fmt.Sprintf("loudness_target=%d", p.loudnessTarget)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)[:16]
}

func (p *Processor) Process(ctx context.Context, filePath string) error {
	logger := p.logger.With("file", filePath)
	logger.InfoContext(ctx, "starting processing")

	if _, err := os.Stat(filePath); err != nil {
		logger.ErrorContext(ctx, "file not found or not readable", "error", err)
		return fmt.Errorf("file not accessible: %w", err)
	}

	if !p.hasAudioTrack(ctx, filePath) {
		logger.InfoContext(ctx, "file has no audio tracks, skipping")
		return nil
	}

	uuid := uuid.New().String()
	scratchPath := filepath.Join(p.scratchDir, uuid)
	if err := os.MkdirAll(scratchPath, 0755); err != nil {
		logger.ErrorContext(ctx, "failed to create scratch directory", "error", err)
		return fmt.Errorf("scratch dir create failed: %w", err)
	}
	defer os.RemoveAll(scratchPath)

	ext := filepath.Ext(filePath)
	inputCopy := filepath.Join(scratchPath, "input_copy"+ext)
	audioWav := filepath.Join(scratchPath, "audio.wav")
	normalizedAac := filepath.Join(scratchPath, "normalized.aac")
	remuxedFile := filepath.Join(scratchPath, "remuxed"+ext)

	logger.InfoContext(ctx, "copying input file")
	if err := p.copyFile(filePath, inputCopy); err != nil {
		logger.ErrorContext(ctx, "failed to copy input", "error", err)
		return err
	}

	logger.InfoContext(ctx, "extracting audio")
	if err := p.extractAudio(ctx, inputCopy, audioWav, logger); err != nil {
		logger.ErrorContext(ctx, "audio extraction failed", "error", err)
		return err
	}

	logger.InfoContext(ctx, "analyzing loudness (pass 1)")
	measuredI, err := p.analyzeLoudness(ctx, audioWav, logger)
	if err != nil {
		logger.InfoContext(ctx, "skipping normalization", "reason", err.Error())
		return nil
	}
	logger.InfoContext(ctx, "loudness analysis complete", "measured_I", measuredI)

	logger.InfoContext(ctx, "normalizing loudness (pass 2)")
	if err := p.normalizeLoudness(ctx, audioWav, normalizedAac, logger); err != nil {
		logger.ErrorContext(ctx, "loudness normalization failed", "error", err)
		return err
	}

	logger.InfoContext(ctx, "re-muxing audio and video")
	if err := p.remuxMedia(ctx, inputCopy, normalizedAac, remuxedFile, logger); err != nil {
		logger.ErrorContext(ctx, "re-muxing failed", "error", err)
		return err
	}

	tmpDir := filepath.Join(p.mediaDir, ".tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		logger.ErrorContext(ctx, "failed to create temp directory in media-dir", "error", err)
		return fmt.Errorf("temp dir create failed: %w", err)
	}

	finalTmpPath := filepath.Join(tmpDir, fmt.Sprintf("final_%s%s", uuid, ext))
	logger.InfoContext(ctx, "copying to temp location", "temp_path", finalTmpPath)
	if err := p.copyFile(remuxedFile, finalTmpPath); err != nil {
		logger.ErrorContext(ctx, "failed to copy to temp location", "error", err)
		os.Remove(finalTmpPath)
		return err
	}

	logger.InfoContext(ctx, "atomically moving to original location")
	if err := os.Rename(finalTmpPath, filePath); err != nil {
		logger.ErrorContext(ctx, "atomic move failed", "error", err)
		os.Remove(finalTmpPath)
		return err
	}

	logger.InfoContext(ctx, "processing complete")
	return nil
}

func (p *Processor) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func (p *Processor) hasAudioTrack(ctx context.Context, filePath string) bool {
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=codec_type", "-of", "csv=p=0", filePath)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "audio"
}

func (p *Processor) extractAudio(ctx context.Context, inputFile, outputWav string, logger *slog.Logger) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputFile, "-q:a", "0", "-map", "a", outputWav)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (p *Processor) analyzeLoudness(ctx context.Context, audioWav string, logger *slog.Logger) (float64, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", audioWav, "-filter", fmt.Sprintf("loudnorm=I=%d:TP=-1.5:LRA=11:print_format=json", p.loudnessTarget), "-f", "null", "-")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	text := string(output)
	start := strings.LastIndex(text, "{")
	end := strings.LastIndex(text, "}")

	if start < 0 || end < 0 || start >= end {
		return 0, fmt.Errorf("no loudness JSON found in ffmpeg output")
	}

	jsonStr := text[start : end+1]

	var result loudnessOutput
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return 0, fmt.Errorf("failed to parse loudness JSON: %w", err)
	}

	if result.InputI == "-inf" || result.InputI == "" {
		return 0, fmt.Errorf("invalid audio (no measurable loudness)")
	}

	inputI := parseFloat(result.InputI)
	return inputI, nil
}

func parseFloat(s string) float64 {
	if s == "-inf" {
		return -100.0
	}
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func (p *Processor) normalizeLoudness(ctx context.Context, audioWav, outputAac string, logger *slog.Logger) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", audioWav, "-filter", fmt.Sprintf("loudnorm=I=%d:TP=-1.5:LRA=11:print_format=summary", p.loudnessTarget), "-c:a", "aac", "-b:a", "256k", outputAac)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (p *Processor) remuxMedia(ctx context.Context, inputVideo, inputAudio, outputFile string, logger *slog.Logger) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputVideo, "-i", inputAudio, "-c:v", "copy", "-c:a", "aac", "-map", "0:v:0", "-map", "1:a:0", "-map", "0:s:?", outputFile)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
