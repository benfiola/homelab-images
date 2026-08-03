package internal

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
)

//go:embed webassets/templates/*.html
var templateFS embed.FS

//go:embed webassets/static
var rawStaticFS embed.FS

var templatePages = []string{"login", "editor", "history"}

type WebOpts struct {
	Opts
	Supervisor *Supervisor
}

type Web struct {
	opts     *WebOpts
	sessions *sessionStore
	pages    map[string]*template.Template
	static   fs.FS
}

type pageData struct {
	Title     string
	ShowNav   bool
	Flash     string
	Error     string
	Disabled  bool // true when ADMIN_PASSWORD is unset - UI is read-only/inert
	Paused    bool
	Protected []KV
	Editable  string
	History   []HistoryEntry
}

func NewWeb(opts *WebOpts) (*Web, error) {
	base, err := template.ParseFS(templateFS, "webassets/templates/layout.html")
	if err != nil {
		return nil, fmt.Errorf("parse layout template: %w", err)
	}

	pages := make(map[string]*template.Template, len(templatePages))
	for _, name := range templatePages {
		clone, err := base.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone layout template: %w", err)
		}
		t, err := clone.ParseFS(templateFS, fmt.Sprintf("webassets/templates/%s.html", name))
		if err != nil {
			return nil, fmt.Errorf("parse %s template: %w", name, err)
		}
		pages[name] = t
	}

	static, err := fs.Sub(rawStaticFS, "webassets/static")
	if err != nil {
		return nil, fmt.Errorf("open static assets: %w", err)
	}

	return &Web{
		opts:     opts,
		sessions: newSessionStore(),
		pages:    pages,
		static:   static,
	}, nil
}

func (w *Web) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	mux := http.NewServeMux()
	w.routes(mux)

	ln, err := net.Listen("tcp", w.opts.AdminAddress)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	logger.Info("web ui listening", "address", w.opts.AdminAddress)

	srv := &http.Server{Handler: mux}
	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (w *Web) render(rw http.ResponseWriter, page string, data pageData) {
	t, ok := w.pages[page]
	if !ok {
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = t.ExecuteTemplate(rw, "layout", data)
}

func flashMessage(code string) string {
	switch code {
	case "saved":
		return "Settings saved. Reboot to apply."
	case "rebooting":
		return "Reboot triggered - settings will apply once the server comes back up."
	case "restored":
		return "Snapshot restored. Reboot to apply."
	case "resumed":
		return "Server resumed."
	case "paused":
		return "Server paused."
	default:
		return ""
	}
}
