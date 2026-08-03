package internal

import (
	"crypto/subtle"
	"net/http"
	"os"
	"sort"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
)

func (w *Web) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", w.handleLoginPage)
	mux.HandleFunc("POST /login", w.handleLoginSubmit)
	mux.HandleFunc("POST /logout", w.handleLogout)

	mux.Handle("GET /{$}", w.requireAuth(http.HandlerFunc(w.handleEditor)))
	mux.Handle("POST /settings", w.requireAuth(http.HandlerFunc(w.handleSaveSettings)))
	mux.Handle("POST /settings/reboot", w.requireAuth(http.HandlerFunc(w.handleSaveAndReboot)))
	mux.Handle("POST /reboot", w.requireAuth(http.HandlerFunc(w.handleReboot)))
	mux.Handle("GET /history", w.requireAuth(http.HandlerFunc(w.handleHistory)))
	mux.Handle("POST /history/{filename}/restore", w.requireAuth(http.HandlerFunc(w.handleRestoreHistory)))

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(w.static)))
}

func (w *Web) disabled() bool {
	return w.opts.AdminPassword == ""
}

func (w *Web) currentSession(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return w.sessions.valid(cookie.Value)
}

func (w *Web) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if w.disabled() {
			w.render(rw, "login", pageData{Title: "Palworld Admin", Disabled: true})
			return
		}
		if !w.currentSession(r) {
			http.Redirect(rw, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(rw, r)
	})
}

func (w *Web) handleLoginPage(rw http.ResponseWriter, r *http.Request) {
	if w.disabled() {
		w.render(rw, "login", pageData{Title: "Palworld Admin", Disabled: true})
		return
	}
	if w.currentSession(r) {
		http.Redirect(rw, r, "/", http.StatusSeeOther)
		return
	}
	w.render(rw, "login", pageData{Title: "Log in"})
}

func (w *Web) handleLoginSubmit(rw http.ResponseWriter, r *http.Request) {
	if w.disabled() {
		w.render(rw, "login", pageData{Title: "Palworld Admin", Disabled: true})
		return
	}
	password := r.FormValue("password")
	if subtle.ConstantTimeCompare([]byte(password), []byte(w.opts.AdminPassword)) != 1 {
		w.render(rw, "login", pageData{Title: "Log in", Error: "Invalid password."})
		return
	}

	token, err := w.sessions.create()
	if err != nil {
		logging.FromContext(r.Context()).Error("create session", "err", err)
		w.render(rw, "login", pageData{Title: "Log in", Error: "Something went wrong. Please try again."})
		return
	}

	http.SetCookie(rw, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(rw, r, "/", http.StatusSeeOther)
}

func (w *Web) handleLogout(rw http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		w.sessions.delete(cookie.Value)
	}
	http.SetCookie(rw, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(rw, r, "/login", http.StatusSeeOther)
}

// loadEditorData reads the live ini and splits it into the read-only
// protected panel and the editable (non-protected) text.
func (w *Web) loadEditorData(title string) (pageData, error) {
	kvs, err := Read(w.opts.DataPath)
	if err != nil {
		return pageData{}, err
	}
	data := pageData{Title: title, ShowNav: true}
	protected := make([]KV, 0, len(ProtectedKeys))
	for _, key := range ProtectedKeys {
		v, ok := Get(kvs, key)
		if !ok {
			continue
		}
		v = unquote(v)
		if key == "AdminPassword" {
			v = "********"
		}
		protected = append(protected, KV{Key: key, Value: v})
	}
	sort.Slice(protected, func(i, j int) bool { return protected[i].Key < protected[j].Key })
	data.Protected = protected
	data.Editable = RenderPretty(Without(kvs, ProtectedKeys...))
	return data, nil
}

func (w *Web) handleEditor(rw http.ResponseWriter, r *http.Request) {
	data, err := w.loadEditorData("Settings")
	if err != nil {
		logging.FromContext(r.Context()).Error("read live ini", "err", err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}
	if msg := r.URL.Query().Get("msg"); msg != "" {
		data.Flash = flashMessage(msg)
	}
	w.render(rw, "editor", data)
}

func (w *Web) renderEditorError(rw http.ResponseWriter, r *http.Request, msg string) {
	data, err := w.loadEditorData("Settings")
	if err != nil {
		logging.FromContext(r.Context()).Error("read live ini", "err", err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}
	data.Error = msg
	w.render(rw, "editor", data)
}

// saveFromForm parses and saves the submitted editor content. On failure it
// has already rendered an error response and returns false.
func (w *Web) saveFromForm(rw http.ResponseWriter, r *http.Request) bool {
	logger := logging.FromContext(r.Context())

	edited, err := ParsePretty(r.FormValue("editable"))
	if err != nil {
		w.renderEditorError(rw, r, "Couldn't parse settings: "+err.Error())
		return false
	}
	if err := Save(w.opts.DataPath, edited, os.Environ(), w.opts.HistoryLimit); err != nil {
		logger.Error("save settings", "err", err)
		w.renderEditorError(rw, r, "Failed to save settings.")
		return false
	}
	logger.Info("settings saved")
	return true
}

func (w *Web) handleSaveSettings(rw http.ResponseWriter, r *http.Request) {
	if !w.saveFromForm(rw, r) {
		return
	}
	http.Redirect(rw, r, "/?msg=saved", http.StatusSeeOther)
}

func (w *Web) handleSaveAndReboot(rw http.ResponseWriter, r *http.Request) {
	if !w.saveFromForm(rw, r) {
		return
	}
	logger := logging.FromContext(r.Context())
	if err := w.opts.Supervisor.Reboot(r.Context()); err != nil {
		logger.Error("reboot", "err", err)
		w.renderEditorError(rw, r, "Saved, but failed to trigger reboot.")
		return
	}
	logger.Info("reboot triggered", "reason", "save-and-reboot")
	http.Redirect(rw, r, "/?msg=rebooting", http.StatusSeeOther)
}

func (w *Web) handleReboot(rw http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if err := w.opts.Supervisor.Reboot(r.Context()); err != nil {
		logger.Error("reboot", "err", err)
		w.renderEditorError(rw, r, "Failed to trigger reboot.")
		return
	}
	logger.Info("reboot triggered", "reason", "manual")
	http.Redirect(rw, r, "/?msg=rebooting", http.StatusSeeOther)
}

func (w *Web) handleHistory(rw http.ResponseWriter, r *http.Request) {
	entries, err := ListHistory(w.opts.DataPath)
	if err != nil {
		logging.FromContext(r.Context()).Error("list history", "err", err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}
	w.render(rw, "history", pageData{Title: "History", ShowNav: true, History: entries})
}

func (w *Web) handleRestoreHistory(rw http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	logger := logging.FromContext(r.Context())
	if err := RestoreHistory(w.opts.DataPath, filename, os.Environ(), w.opts.HistoryLimit); err != nil {
		logger.Error("restore history", "err", err, "filename", filename)
		http.Error(rw, "failed to restore snapshot", http.StatusInternalServerError)
		return
	}
	logger.Info("history restored", "filename", filename)
	http.Redirect(rw, r, "/?msg=restored", http.StatusSeeOther)
}
