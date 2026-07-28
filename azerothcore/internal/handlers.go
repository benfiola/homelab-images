package internal

import (
	"context"
	"fmt"
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
	mux.Handle("POST /accounts", w.requireAdmin(http.HandlerFunc(w.handleAdminCreateAccount)))
	mux.Handle("POST /accounts/{id}/reset-password", w.requireAdmin(http.HandlerFunc(w.handleAdminResetPassword)))
	mux.Handle("POST /accounts/{id}/gm-level", w.requireAdmin(http.HandlerFunc(w.handleAdminSetGMLevel)))
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

// handleDashboard renders the account list. It also accepts two optional
// query params, "patch_id"/"patch_gm_level", set only by the redirect at
// the end of handleAdminSetGMLevel: AzerothCore applies ".account set
// gmlevel" via an async DB write, so a plain re-query immediately after the
// SOAP call can race that write and still show the old value even though
// the command already succeeded. Carrying the just-applied value through
// the redirect and patching it into the freshly-queried list sidesteps that
// race without giving up the redirect (which is what keeps the address bar
// on "/" and avoids a "confirm form resubmission" prompt on reload).
func (w *Web) handleDashboard(rw http.ResponseWriter, r *http.Request) {
	account := accountFromContext(r.Context())
	accounts, err := listAccounts(r.Context(), w.db)
	if err != nil {
		logging.FromContext(r.Context()).Error("list accounts", "error", err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	if patchID, err := strconv.ParseInt(r.URL.Query().Get("patch_id"), 10, 64); err == nil {
		if patchLevel, err := strconv.Atoi(r.URL.Query().Get("patch_gm_level")); err == nil {
			for i := range accounts {
				if accounts[i].ID == patchID {
					accounts[i].GMLevel = patchLevel
					break
				}
			}
		}
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

func (w *Web) handleAdminCreateAccount(rw http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || containsWhitespace(username) {
		w.renderDashboardError(rw, r, "Username must not be empty or contain whitespace.")
		return
	}
	if password == "" || containsWhitespace(password) {
		w.renderDashboardError(rw, r, "Password must not be empty or contain whitespace.")
		return
	}

	if err := w.soap.CreateAccount(r.Context(), username, password); err != nil {
		logging.FromContext(r.Context()).Error("soap create account", "error", err)
		w.renderDashboardError(rw, r, "Failed to create account.")
		return
	}

	http.Redirect(rw, r, "/?msg=created", http.StatusSeeOther)
}

// resolveTargetAccount looks up the account named by the "{id}" path value,
// refusing to resolve the hidden web service account - it's excluded from
// the dashboard listing precisely so it can't be reset/deleted/modified
// through the UI, and that guarantee needs to hold even if an admin guesses
// or otherwise obtains its id.
func (w *Web) resolveTargetAccount(rw http.ResponseWriter, r *http.Request) (*Account, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(rw, "invalid account id", http.StatusBadRequest)
		return nil, false
	}
	target, err := getAccountByID(r.Context(), w.db, id)
	if err != nil {
		logging.FromContext(r.Context()).Error("get account", "error", err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return nil, false
	}
	if target == nil || target.Username == webServiceUsername {
		http.Error(rw, "account not found", http.StatusNotFound)
		return nil, false
	}
	return target, true
}

func (w *Web) handleAdminResetPassword(rw http.ResponseWriter, r *http.Request) {
	target, ok := w.resolveTargetAccount(rw, r)
	if !ok {
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
	target, ok := w.resolveTargetAccount(rw, r)
	if !ok {
		return
	}

	// Refuse to delete the account the current session is logged in as -
	// otherwise an admin can lock themselves out of the UI with no way
	// back in short of direct DB access.
	if session := accountFromContext(r.Context()); session.ID == target.ID {
		w.renderDashboardError(rw, r, "You can't delete the account you're currently logged in as.")
		return
	}

	if err := w.soap.DeleteAccount(r.Context(), target.Username); err != nil {
		logging.FromContext(r.Context()).Error("soap delete account", "error", err)
		w.renderDashboardError(rw, r, "Failed to delete account.")
		return
	}

	http.Redirect(rw, r, "/?msg=deleted", http.StatusSeeOther)
}

func (w *Web) handleAdminSetGMLevel(rw http.ResponseWriter, r *http.Request) {
	target, ok := w.resolveTargetAccount(rw, r)
	if !ok {
		return
	}

	level, err := strconv.Atoi(r.FormValue("gm_level"))
	if err != nil || level < 0 || level > 3 {
		w.renderDashboardError(rw, r, "GM level must be between 0 and 3.")
		return
	}

	if err := w.soap.SetGMLevel(r.Context(), target.Username, level); err != nil {
		logging.FromContext(r.Context()).Error("soap set gmlevel", "error", err)
		w.renderDashboardError(rw, r, "Failed to update GM level.")
		return
	}

	redirectURL := fmt.Sprintf("/?msg=gmlevel-updated&patch_id=%d&patch_gm_level=%d", target.ID, level)
	http.Redirect(rw, r, redirectURL, http.StatusSeeOther)
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
