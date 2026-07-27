package internal

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/cmd"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	_ "github.com/go-sql-driver/mysql"
)

// bootstrapGMLevel is the GM level assigned to both bootstrapped accounts.
// AzerothCore's SOAP interface refuses to open a session for any account
// below this level, so it isn't configurable - anything lower would
// silently break the web service account's ability to reach SOAP.
const bootstrapGMLevel = 3

type Opts struct {
	GameDataURL      string
	RealmlistAddress string
	AdminUsername    string
}

// dataDir returns AC_DATA_DIR (AzerothCore's own config surface), defaulting
// to "/data" to match authserver/worldserver/dbimport.
func dataDir() string {
	if v := os.Getenv("AC_DATA_DIR"); v != "" {
		return v
	}
	return "/data"
}

type Init struct {
	opts *Opts
}

func New(opts *Opts) (*Init, error) {
	return &Init{opts: opts}, nil
}

func (i *Init) Run(ctx context.Context) error {
	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"download game data", i.downloadGameData},
		{"wait for db", i.waitForDB},
		{"run migrations", i.runMigrations},
		{"initialize server", i.initializeServer},
		{"bootstrap accounts", i.bootstrapAccounts},
	}

	for _, step := range steps {
		logger := logging.FromContext(ctx)
		logger.Info("starting", "step", step.name)
		if err := step.fn(ctx); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
		logger.Info("completed", "step", step.name)
	}
	return nil
}

func (i *Init) downloadGameData(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	dataDir := dataDir()

	markerFile := filepath.Join(dataDir, ".game-data-version")
	if existing, err := os.ReadFile(markerFile); err == nil {
		if strings.TrimSpace(string(existing)) == i.opts.GameDataURL {
			logger.Info("game data already present and up to date, skipping")
			return nil
		}
		logger.Info("game data URL changed, re-downloading", "url", i.opts.GameDataURL)
		if err := os.RemoveAll(dataDir); err != nil {
			return fmt.Errorf("clean data dir: %w", err)
		}
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	tmpFile := "/tmp/game-data.archive"
	logger.Info("downloading", "url", i.opts.GameDataURL)
	if err := cmd.Stream(ctx, "curl", "-fsSL", i.opts.GameDataURL, "-o", tmpFile); err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer os.Remove(tmpFile)

	logger.Info("extracting")
	if err := cmd.Stream(ctx, "bsdtar", "-xmf", tmpFile, "-C", dataDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	if err := os.WriteFile(markerFile, []byte(i.opts.GameDataURL+"\n"), 0644); err != nil {
		return fmt.Errorf("write marker: %w", err)
	}

	return nil
}

func (i *Init) waitForDB(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	info, err := parseDBInfo(os.Getenv("AC_LOGIN_DATABASE_INFO"))
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", info.adminDSN())
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()
	db.SetConnMaxLifetime(time.Minute)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	for {
		if err := db.PingContext(ctx); err == nil {
			break
		}
		logger.Info("db not ready, retrying in 5s")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}

	return nil
}

func (i *Init) runMigrations(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	logger.Info("running dbimport")
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find self: %w", err)
	}
	// shell out to our own "dbimport" subcommand rather than the AzerothCore
	// binary directly, so flag defaults/handling stay in one place; DB
	// connection info and dirs are AzerothCore's own config surface and are
	// inherited by the subprocess from the environment rather than passed
	// explicitly
	if err := cmd.Stream(ctx, self, "dbimport"); err != nil {
		return fmt.Errorf("dbimport: %w", err)
	}

	return nil
}

func (i *Init) initializeServer(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	if i.opts.RealmlistAddress == "" {
		return nil
	}

	info, err := parseDBInfo(os.Getenv("AC_LOGIN_DATABASE_INFO"))
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", info.dsn())
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	logger.Info("updating realmlist", "address", i.opts.RealmlistAddress)
	if _, err := db.ExecContext(ctx, "UPDATE realmlist SET address = ? WHERE id = 1", i.opts.RealmlistAddress); err != nil {
		return fmt.Errorf("update realmlist: %w", err)
	}

	return nil
}

// bootstrapAccounts idempotently creates the human-facing admin account and
// the hidden WEBUI service account (used only by the "web" command to
// authenticate to worldserver's SOAP interface) if they don't already
// exist. Unlike the old YAML-driven account sync, this never touches an
// account again once created - the web UI owns account lifecycle from here
// on. Both accounts' generated passwords are recorded in a small
// purpose-built table (not part of AzerothCore's own schema/migrations) so
// they're recoverable: a human can look up the admin's initial password,
// and the "web" command looks up its own service-account password there at
// startup.
func (i *Init) bootstrapAccounts(ctx context.Context) error {
	info, err := parseDBInfo(os.Getenv("AC_LOGIN_DATABASE_INFO"))
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", info.dsn())
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS web_bootstrap_credentials (
			username VARCHAR(32) NOT NULL PRIMARY KEY,
			password VARCHAR(64) NOT NULL,
			created_at DATETIME NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("create web_bootstrap_credentials table: %w", err)
	}

	for _, username := range []string{i.opts.AdminUsername, webServiceUsername} {
		if err := i.bootstrapAccount(ctx, db, username); err != nil {
			return fmt.Errorf("bootstrap account %s: %w", username, err)
		}
	}

	return nil
}

func (i *Init) bootstrapAccount(ctx context.Context, db *sql.DB, username string) error {
	logger := logging.FromContext(ctx)
	upper := strings.ToUpper(username)

	var existing int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM account WHERE username = ?", upper).Scan(&existing); err != nil {
		return fmt.Errorf("check existing account: %w", err)
	}
	if existing > 0 {
		logger.Info("account already exists, skipping bootstrap", "username", username)
		return nil
	}

	password, err := generatePassword(24)
	if err != nil {
		return fmt.Errorf("generate password: %w", err)
	}

	salt, verifier, err := generateVerifier(username, password)
	if err != nil {
		return fmt.Errorf("generate verifier: %w", err)
	}

	logger.Info("bootstrapping account", "username", username, "gm_level", bootstrapGMLevel)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO account
			(username, salt, verifier, email, reg_mail, joindate, last_ip, last_attempt_ip,
			 failed_logins, locked, lock_country, online, expansion, Flags,
			 mutetime, mutereason, muteby, locale, os, recruiter, totaltime)
		VALUES
			(?, ?, ?, '', '', NOW(), '127.0.0.1', '127.0.0.1',
			 0, 0, '00', 0, 2, 0,
			 0, '', '', 0, '', 0, 0)
	`, upper, salt, verifier); err != nil {
		return fmt.Errorf("insert account: %w", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO account_access (id, gmlevel, RealmID)
		SELECT id, ?, -1 FROM account WHERE username = ?
	`, bootstrapGMLevel, upper); err != nil {
		return fmt.Errorf("insert account_access: %w", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO web_bootstrap_credentials (username, password, created_at)
		VALUES (?, ?, NOW())
	`, upper, password); err != nil {
		return fmt.Errorf("insert bootstrap credentials: %w", err)
	}

	return nil
}

const passwordCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// generatePassword generates a cryptographically random password. The
// charset deliberately excludes spaces (and any non-alnum characters),
// since generated passwords for the bootstrap accounts get passed as
// space-delimited arguments to SOAP-issued AzerothCore console commands.
func generatePassword(length int) (string, error) {
	buf := make([]byte, length)
	for idx := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordCharset))))
		if err != nil {
			return "", err
		}
		buf[idx] = passwordCharset[n.Int64()]
	}
	return string(buf), nil
}
