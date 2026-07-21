package internal

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
)

// Server runs the Radarr/Sonarr-compatible webhook receiver and a /healthz
// endpoint for Kubernetes probes.
type Server struct {
	addr   string
	token  string
	worker *Worker
}

func NewServer(addr, token string, worker *Worker) *Server {
	return &Server{addr: addr, token: token, worker: worker}
}

func (s *Server) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /webhooks/radarr", s.handleRadarr)
	mux.HandleFunc("POST /webhooks/sonarr", s.handleSonarr)

	srv := &http.Server{Addr: s.addr, Handler: mux}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("webhook server listening", "addr", s.addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) checkAuth(r *http.Request) bool {
	if s.token == "" {
		return true
	}
	provided := r.URL.Query().Get("token")
	if provided == "" {
		provided = r.Header.Get("X-Loudnorm-Token")
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(s.token)) == 1
}

type radarrPayload struct {
	EventType string `json:"eventType"`
	Movie     *struct {
		FolderPath string `json:"folderPath"`
	} `json:"movie"`
	MovieFile *struct {
		Path string `json:"path"`
	} `json:"movieFile"`
	DeletedFiles *bool `json:"deletedFiles"`
}

type sonarrPayload struct {
	EventType string `json:"eventType"`
	Series    *struct {
		Path string `json:"path"`
	} `json:"series"`
	EpisodeFile *struct {
		Path string `json:"path"`
	} `json:"episodeFile"`
	DeletedFiles *bool `json:"deletedFiles"`
}

func (s *Server) handleRadarr(w http.ResponseWriter, r *http.Request) {
	if !s.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var p radarrPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	logger := logging.FromContext(r.Context())
	switch p.EventType {
	case "Download":
		if p.MovieFile != nil && p.MovieFile.Path != "" {
			s.worker.Enqueue(p.MovieFile.Path)
		}
	case "MovieFileDelete":
		if p.MovieFile != nil && p.MovieFile.Path != "" {
			s.worker.HandleDelete(r.Context(), p.MovieFile.Path)
		}
	case "MovieDelete":
		if p.Movie != nil && p.Movie.FolderPath != "" && p.DeletedFiles != nil && *p.DeletedFiles {
			s.worker.HandleDeletePrefix(r.Context(), p.Movie.FolderPath)
		}
	default:
		logger.Debug("ignoring webhook event", "eventType", p.EventType)
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSonarr(w http.ResponseWriter, r *http.Request) {
	if !s.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var p sonarrPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	logger := logging.FromContext(r.Context())
	switch p.EventType {
	case "Download":
		if p.EpisodeFile != nil && p.EpisodeFile.Path != "" {
			s.worker.Enqueue(p.EpisodeFile.Path)
		}
	case "EpisodeFileDelete":
		if p.EpisodeFile != nil && p.EpisodeFile.Path != "" {
			s.worker.HandleDelete(r.Context(), p.EpisodeFile.Path)
		}
	case "SeriesDelete":
		if p.Series != nil && p.Series.Path != "" && p.DeletedFiles != nil && *p.DeletedFiles {
			s.worker.HandleDeletePrefix(r.Context(), p.Series.Path)
		}
	default:
		logger.Debug("ignoring webhook event", "eventType", p.EventType)
	}
	w.WriteHeader(http.StatusOK)
}
