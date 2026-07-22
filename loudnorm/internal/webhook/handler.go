package webhook

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/benfiola/homelab-images/loudnorm/internal/queue"
	"github.com/benfiola/homelab-images/loudnorm/internal/state"
)

type Handler struct {
	queue        *queue.Queue
	db           *state.DB
	logger       *slog.Logger
	webhookToken string
}

func New(q *queue.Queue, db *state.DB, logger *slog.Logger, webhookToken string) *Handler {
	return &Handler{
		queue:        q,
		db:           db,
		logger:       logger,
		webhookToken: webhookToken,
	}
}

func (h *Handler) authenticate(r *http.Request) bool {
	if h.webhookToken == "" {
		return true
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return false
	}

	return parts[1] == h.webhookToken
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) Sonarr(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.authenticate(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload SonarrPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to decode sonarr payload", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch payload.EventType {
	case "Download":
		if payload.EpisodeFile.Path != "" {
			h.logger.InfoContext(r.Context(), "webhook: sonarr download", "file", payload.EpisodeFile.Path)
			h.queue.Add(payload.EpisodeFile.Path)
		}

	case "Rename":
		for _, rf := range payload.RenamedEpisodeFiles {
			if rf.PreviousPath != "" {
				h.db.DeleteEntry(rf.PreviousPath)
				h.logger.InfoContext(r.Context(), "webhook: sonarr rename delete old", "path", rf.PreviousPath)
			}
			if rf.Path != "" {
				h.queue.Add(rf.Path)
				h.logger.InfoContext(r.Context(), "webhook: sonarr rename add new", "path", rf.Path)
			}
		}

	case "EpisodeFileDelete":
		if payload.EpisodeFile.Path != "" {
			h.db.DeleteEntry(payload.EpisodeFile.Path)
			h.logger.InfoContext(r.Context(), "webhook: sonarr delete", "file", payload.EpisodeFile.Path, "reason", payload.DeleteReason)
		}

	default:
		h.logger.DebugContext(r.Context(), "sonarr webhook unrecognized event", "event_type", payload.EventType)
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) Radarr(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.authenticate(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload RadarrPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to decode radarr payload", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch payload.EventType {
	case "Download":
		if payload.MovieFile.Path != "" {
			h.logger.InfoContext(r.Context(), "webhook: radarr download", "file", payload.MovieFile.Path)
			h.queue.Add(payload.MovieFile.Path)
		}

	case "Rename":
		for _, rf := range payload.RenamedMovieFiles {
			if rf.PreviousPath != "" {
				h.db.DeleteEntry(rf.PreviousPath)
				h.logger.InfoContext(r.Context(), "webhook: radarr rename delete old", "path", rf.PreviousPath)
			}
			if rf.Path != "" {
				h.queue.Add(rf.Path)
				h.logger.InfoContext(r.Context(), "webhook: radarr rename add new", "path", rf.Path)
			}
		}

	case "MovieFileDelete":
		if payload.MovieFile.Path != "" {
			h.db.DeleteEntry(payload.MovieFile.Path)
			h.logger.InfoContext(r.Context(), "webhook: radarr delete", "file", payload.MovieFile.Path, "reason", payload.DeleteReason)
		}

	default:
		h.logger.DebugContext(r.Context(), "radarr webhook unrecognized event", "event_type", payload.EventType)
	}

	w.WriteHeader(http.StatusAccepted)
}
