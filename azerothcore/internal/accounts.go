package internal

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"fmt"
	"strings"
)

// webServiceUsername is the hidden account the "web" command uses to
// authenticate to worldserver's SOAP interface. It's never shown in the
// account listing and never touched outside of bootstrap.
const webServiceUsername = "WEBUI"

// randomBotAccountPrefix matches the accounts mod-playerbots creates for
// itself at runtime (e.g. "RNDBOT0"). These are filtered out of the account
// listing so the admin dashboard isn't cluttered with bot accounts.
const randomBotAccountPrefix = "RNDBOT"

type Account struct {
	ID       int64
	Username string
	GMLevel  int
	Locked   bool
	Online   bool
}

// listAccounts returns every account except the hidden web service account
// and mod-playerbots' own bot accounts.
func listAccounts(ctx context.Context, db *sql.DB) ([]Account, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT a.id, a.username, COALESCE(aa.gmlevel, 0), a.locked, a.online
		FROM account a
		LEFT JOIN account_access aa ON aa.id = a.id AND aa.RealmID = -1
		WHERE a.username <> ? AND a.username NOT LIKE CONCAT(?, '%')
		ORDER BY a.username
	`, strings.ToUpper(webServiceUsername), strings.ToUpper(randomBotAccountPrefix))
	if err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Username, &a.GMLevel, &a.Locked, &a.Online); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accounts: %w", err)
	}
	return accounts, nil
}

// getAccountByID looks up a single account by id, used by admin handlers
// that receive an id from the URL path and need the account's username to
// issue a SOAP command (which is username-addressed).
func getAccountByID(ctx context.Context, db *sql.DB, id int64) (*Account, error) {
	var a Account
	err := db.QueryRowContext(ctx, `
		SELECT a.id, a.username, COALESCE(aa.gmlevel, 0), a.locked, a.online
		FROM account a
		LEFT JOIN account_access aa ON aa.id = a.id AND aa.RealmID = -1
		WHERE a.id = ?
	`, id).Scan(&a.ID, &a.Username, &a.GMLevel, &a.Locked, &a.Online)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query account: %w", err)
	}
	return &a, nil
}

// getAccountByUsername looks up a single account by username, used to
// resolve the currently logged-in session's account (GM level, id) on each
// request.
func getAccountByUsername(ctx context.Context, db *sql.DB, username string) (*Account, error) {
	var a Account
	err := db.QueryRowContext(ctx, `
		SELECT a.id, a.username, COALESCE(aa.gmlevel, 0), a.locked, a.online
		FROM account a
		LEFT JOIN account_access aa ON aa.id = a.id AND aa.RealmID = -1
		WHERE a.username = ?
	`, strings.ToUpper(username)).Scan(&a.ID, &a.Username, &a.GMLevel, &a.Locked, &a.Online)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query account: %w", err)
	}
	return &a, nil
}

// verifyAccountPassword recomputes the SRP verifier for `password` against
// the account's stored salt and compares it (constant-time) to the stored
// verifier. Used both for login and for self-service old-password checks.
// Never reveals or requires knowing the actual stored password - AzerothCore
// itself never stores it.
func verifyAccountPassword(ctx context.Context, db *sql.DB, username, password string) (bool, error) {
	var salt, verifier []byte
	err := db.QueryRowContext(ctx, "SELECT salt, verifier FROM account WHERE username = ?", strings.ToUpper(username)).Scan(&salt, &verifier)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query account: %w", err)
	}

	computed, err := computeVerifier(username, password, salt)
	if err != nil {
		return false, fmt.Errorf("compute verifier: %w", err)
	}

	return subtle.ConstantTimeCompare(computed, verifier) == 1, nil
}

// getBootstrapCredentialPassword looks up the plaintext password recorded
// for `username` at bootstrap time (see Init.bootstrapAccount). Used by the
// "web" command at startup to fetch its own SOAP service-account password.
func getBootstrapCredentialPassword(ctx context.Context, db *sql.DB, username string) (string, error) {
	var password string
	err := db.QueryRowContext(ctx, "SELECT password FROM web_bootstrap_credentials WHERE username = ?", strings.ToUpper(username)).Scan(&password)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("no bootstrap credentials found for %q - has \"init\" run yet?", username)
	}
	if err != nil {
		return "", fmt.Errorf("query bootstrap credentials: %w", err)
	}
	return password, nil
}
