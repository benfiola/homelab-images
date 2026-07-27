package internal

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"os"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
	_ "github.com/go-sql-driver/mysql"
)

//go:embed webassets/templates/*.html
var templateFS embed.FS

//go:embed webassets/static
var rawStaticFS embed.FS

var templatePages = []string{"login", "dashboard", "account"}

type WebOpts struct {
	ListenAddress         string
	SOAPAddress           string
	AdminGMLevelThreshold int
}

type Web struct {
	opts     *WebOpts
	db       *sql.DB
	soap     *soapClient
	sessions *sessionStore
	pages    map[string]*template.Template
	static   fs.FS
}

type pageData struct {
	Title    string
	Account  *Account
	IsAdmin  bool
	Flash    string
	Error    string
	Accounts []Account
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

	info, err := parseDBInfo(os.Getenv("AC_LOGIN_DATABASE_INFO"))
	if err != nil {
		return err
	}
	db, err := sql.Open("mysql", info.dsn())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	w.db = db

	password, err := getBootstrapCredentialPassword(ctx, db, webServiceUsername)
	if err != nil {
		return fmt.Errorf("look up web service account credentials: %w", err)
	}
	w.soap = newSOAPClient(w.opts.SOAPAddress, webServiceUsername, password)

	mux := http.NewServeMux()
	w.routes(mux)

	ln, err := net.Listen("tcp", w.opts.ListenAddress)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	logger.Info("listening", "address", w.opts.ListenAddress)

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
	case "password-reset":
		return "Password reset."
	case "deleted":
		return "Account deleted."
	case "password-changed":
		return "Password changed."
	default:
		return ""
	}
}
