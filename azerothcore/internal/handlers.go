package internal

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
)

func (w *Web) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", w.handleLoginPage)
	mux.HandleFunc("POST /login", w.handleLoginSubmit)
	mux.HandleFunc("POST /logout", w.handleLogout)

	mux.Handle("GET /{$}", w.requireAdmin(http.HandlerFunc(w.handleDashboard)))
	mux.Handle("POST /accounts/{id}/reset-password", w.requireAdmin(http.HandlerFunc(w.handleAdminResetPassword)))
	mux.Handle("POST /accounts/{id}/delete", w.requireAdmin(http.HandlerFunc(w.handleAdminDeleteAccount)))

	mux.Handle("GET /account", w.requireAuth(http.HandlerFunc(w.handleAccountPage)))
	mux.Handle("POST /account", w.requireAuth(http.HandlerFunc(w.handleAccountChangePassword)))

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(w.static)))
}

// currentAccount resolves the logged-in account (if any) for the request's
// session cookie, re-reading its GM level fresh from the DB on every
// request rather than caching it in the session.
func (w *Web) currentAccount(r *http.Request) *Account {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	sess, ok := w.sessions.get(cookie.Value)
	if !ok {
		return nil
	}
	account, err := getAccountByUsername(r.Context(), w.db, sess.username)
	if err != nil || account == nil {
		return nil
	}
	return account
}

func (w *Web) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		account := w.currentAccount(r)
		if account == nil {
			http.Redirect(rw, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), accountCtxKey, account)
		next.ServeHTTP(rw, r.WithContext(ctx))
	})
}

// requireAdmin gates a route to accounts at or above the configured GM
// level threshold, bouncing anyone else to the self-service page rather
// than a bare 403 - they're logged in, just not privileged enough for this
// particular page.
func (w *Web) requireAdmin(next http.Handler) http.Handler {
	return w.requireAuth(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		account := accountFromContext(r.Context())
		if account.GMLevel < w.opts.AdminGMLevelThreshold {
			http.Redirect(rw, r, "/account", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(rw, r)
	}))
}

func (w *Web) handleLoginPage(rw http.ResponseWriter, r *http.Request) {
	if w.currentAccount(r) != nil {
		http.Redirect(rw, r, "/", http.StatusSeeOther)
		return
	}
	w.render(rw, "login", pageData{Title: "Log in"})
}

func (w *Web) handleLoginSubmit(rw http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	ok, err := verifyAccountPassword(r.Context(), w.db, username, password)
	if err != nil {
		logging.FromContext(r.Context()).Error("verify password", "error", err)
		w.render(rw, "login", pageData{Title: "Log in", Error: "Something went wrong. Please try again."})
		return
	}
	if !ok {
		w.render(rw, "login", pageData{Title: "Log in", Error: "Invalid username or password."})
		return
	}

	token, err := w.sessions.create(strings.ToUpper(username))
	if err != nil {
		logging.FromContext(r.Context()).Error("create session", "error", err)
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

func (w *Web) handleDashboard(rw http.ResponseWriter, r *http.Request) {
	account := accountFromContext(r.Context())
	accounts, err := listAccounts(r.Context(), w.db)
	if err != nil {
		logging.FromContext(r.Context()).Error("list accounts", "error", err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}
	data := pageData{Title: "Accounts", Account: account, IsAdmin: true, Accounts: accounts}
	if msg := r.URL.Query().Get("msg"); msg != "" {
		data.Flash = flashMessage(msg)
	}
	w.render(rw, "dashboard", data)
}

func (w *Web) renderDashboardError(rw http.ResponseWriter, r *http.Request, msg string) {
	account := accountFromContext(r.Context())
	accounts, err := listAccounts(r.Context(), w.db)
	if err != nil {
		logging.FromContext(r.Context()).Error("list accounts", "error", err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}
	w.render(rw, "dashboard", pageData{Title: "Accounts", Account: account, IsAdmin: true, Accounts: accounts, Error: msg})
}

func containsWhitespace(s string) bool {
	return strings.ContainsAny(s, " \t\n\r")
}

func (w *Web) handleAdminResetPassword(rw http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(rw, "invalid account id", http.StatusBadRequest)
		return
	}
	target, err := getAccountByID(r.Context(), w.db, id)
	if err != nil {
		logging.FromContext(r.Context()).Error("get account", "error", err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}
	if target == nil {
		http.Error(rw, "account not found", http.StatusNotFound)
		return
	}

	newPassword := r.FormValue("new_password")
	if newPassword == "" || containsWhitespace(newPassword) {
		w.renderDashboardError(rw, r, "Password must not be empty or contain whitespace.")
		return
	}

	if err := w.soap.SetPassword(r.Context(), target.Username, newPassword); err != nil {
		logging.FromContext(r.Context()).Error("soap set password", "error", err)
		w.renderDashboardError(rw, r, "Failed to reset password.")
		return
	}

	http.Redirect(rw, r, "/?msg=password-reset", http.StatusSeeOther)
}

func (w *Web) handleAdminDeleteAccount(rw http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(rw, "invalid account id", http.StatusBadRequest)
		return
	}
	target, err := getAccountByID(r.Context(), w.db, id)
	if err != nil {
		logging.FromContext(r.Context()).Error("get account", "error", err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}
	if target == nil {
		http.Error(rw, "account not found", http.StatusNotFound)
		return
	}

	if err := w.soap.DeleteAccount(r.Context(), target.Username); err != nil {
		logging.FromContext(r.Context()).Error("soap delete account", "error", err)
		w.renderDashboardError(rw, r, "Failed to delete account.")
		return
	}

	http.Redirect(rw, r, "/?msg=deleted", http.StatusSeeOther)
}

func (w *Web) handleAccountPage(rw http.ResponseWriter, r *http.Request) {
	account := accountFromContext(r.Context())
	data := pageData{Title: "My Account", Account: account, IsAdmin: account.GMLevel >= w.opts.AdminGMLevelThreshold}
	if msg := r.URL.Query().Get("msg"); msg != "" {
		data.Flash = flashMessage(msg)
	}
	w.render(rw, "account", data)
}

func (w *Web) handleAccountChangePassword(rw http.ResponseWriter, r *http.Request) {
	account := accountFromContext(r.Context())

	renderErr := func(msg string) {
		w.render(rw, "account", pageData{
			Title:   "My Account",
			Account: account,
			IsAdmin: account.GMLevel >= w.opts.AdminGMLevelThreshold,
			Error:   msg,
		})
	}

	oldPassword := r.FormValue("old_password")
	newPassword := r.FormValue("new_password")
	newPasswordConfirm := r.FormValue("new_password_confirm")

	if newPassword == "" || containsWhitespace(newPassword) {
		renderErr("New password must not be empty or contain whitespace.")
		return
	}
	if newPassword != newPasswordConfirm {
		renderErr("New passwords do not match.")
		return
	}

	ok, err := verifyAccountPassword(r.Context(), w.db, account.Username, oldPassword)
	if err != nil {
		logging.FromContext(r.Context()).Error("verify password", "error", err)
		renderErr("Something went wrong. Please try again.")
		return
	}
	if !ok {
		renderErr("Current password is incorrect.")
		return
	}

	// account.Username comes from the authenticated session, never from a
	// form field or URL parameter - a non-admin can only ever change their
	// own password this way.
	if err := w.soap.SetPassword(r.Context(), account.Username, newPassword); err != nil {
		logging.FromContext(r.Context()).Error("soap set password", "error", err)
		renderErr("Failed to change password.")
		return
	}

	http.Redirect(rw, r, "/account?msg=password-changed", http.StatusSeeOther)
}
