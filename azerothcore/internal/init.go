package internal

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/cmd"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

type Opts struct {
	GameDataURL      string
	RealmlistAddress string
	ConfigFile       string
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

	markerFile := filepath.Join(dataDir(), ".ac-migrated")
	if _, err := os.Stat(markerFile); err == nil {
		logger.Info("migrations already complete, skipping")
		return nil
	}

	info, err := parseDBInfo(os.Getenv("AC_LOGIN_DATABASE_INFO"))
	if err != nil {
		return err
	}
	worldInfo, err := parseDBInfo(os.Getenv("AC_WORLD_DATABASE_INFO"))
	if err != nil {
		return err
	}
	charInfo, err := parseDBInfo(os.Getenv("AC_CHARACTER_DATABASE_INFO"))
	if err != nil {
		return err
	}
	pbInfo, err := parseDBInfo(os.Getenv("AC_PLAYERBOTS_DATABASE_INFO"))
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", info.adminDSN())
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	logger.Info("dropping existing databases")
	for _, dbname := range []string{"acore_auth", "acore_characters", "acore_world", pbInfo.dbname} {
		if _, err := db.ExecContext(ctx, "DROP DATABASE IF EXISTS `"+dbname+"`"); err != nil {
			return fmt.Errorf("drop %s: %w", dbname, err)
		}
	}

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

	// dbimport doesn't create or populate the playerbots database - see
	// https://github.com/mod-playerbots/mod-playerbots/wiki/Installation-Guide
	if err := i.importPlayerbots(ctx, db, pbInfo, charInfo, worldInfo); err != nil {
		return fmt.Errorf("import playerbots: %w", err)
	}

	if err := os.WriteFile(markerFile, nil, 0644); err != nil {
		return fmt.Errorf("write marker: %w", err)
	}

	return nil
}

// importPlayerbots creates the playerbots database and imports the module's
// characters/world SQL additions - none of this is handled by dbimport. See
// https://github.com/mod-playerbots/mod-playerbots/wiki/Installation-Guide
func (i *Init) importPlayerbots(ctx context.Context, db *sql.DB, pbInfo, charInfo, worldInfo *dbInfo) error {
	logger := logging.FromContext(ctx)

	logger.Info("creating playerbots database")
	if _, err := db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+pbInfo.dbname+"` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"); err != nil {
		return fmt.Errorf("create %s: %w", pbInfo.dbname, err)
	}

	mysql := os.Getenv("AC_MY_SQLEXECUTABLE")
	if mysql == "" {
		var err error
		mysql, err = exec.LookPath("mysql")
		if err != nil {
			return fmt.Errorf("mysql client not found: set AC_MY_SQLEXECUTABLE")
		}
	}

	imports := []struct {
		info *dbInfo
		dir  string
	}{
		{charInfo, "/azerothcore/modules/mod-playerbots/data/sql/characters/base"},
		{worldInfo, "/azerothcore/modules/mod-playerbots/data/sql/world/base"},
	}
	for _, imp := range imports {
		files, err := filepath.Glob(filepath.Join(imp.dir, "*.sql"))
		if err != nil {
			return fmt.Errorf("glob %s: %w", imp.dir, err)
		}
		sort.Strings(files)
		for _, file := range files {
			logger.Info("importing playerbots sql", "file", file, "database", imp.info.dbname)
			if err := mysqlImport(ctx, mysql, imp.info, file); err != nil {
				return fmt.Errorf("import %s: %w", file, err)
			}
		}
	}

	return nil
}

// mysqlImport pipes file into the mysql CLI against the database described
// by info, matching how AzerothCore's own DBUpdater applies SQL files (a
// temp --defaults-extra-file avoids leaking the password via argv/ps).
func mysqlImport(ctx context.Context, mysqlBin string, info *dbInfo, file string) error {
	in, err := os.Open(file)
	if err != nil {
		return err
	}
	defer in.Close()

	optFile, err := os.CreateTemp("", "mysql-*.cnf")
	if err != nil {
		return err
	}
	defer os.Remove(optFile.Name())
	if _, err := fmt.Fprintf(optFile, "[client]\npassword=%s\n", info.pass); err != nil {
		optFile.Close()
		return err
	}
	if err := optFile.Close(); err != nil {
		return err
	}

	c := exec.CommandContext(ctx, mysqlBin,
		"--defaults-extra-file="+optFile.Name(),
		"-h", info.host,
		"-P", info.port,
		"-u", info.user,
		info.dbname,
	)
	c.Stdin = in
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

type config struct {
	Accounts []accountConfig `yaml:"accounts"`
}

type accountConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	GmLevel  int    `yaml:"gm_level"`
}

func (i *Init) initializeServer(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	info, err := parseDBInfo(os.Getenv("AC_LOGIN_DATABASE_INFO"))
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", info.dsn())
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	if i.opts.RealmlistAddress != "" {
		logger.Info("updating realmlist", "address", i.opts.RealmlistAddress)
		if _, err := db.ExecContext(ctx, "UPDATE realmlist SET address = ? WHERE id = 1", i.opts.RealmlistAddress); err != nil {
			return fmt.Errorf("update realmlist: %w", err)
		}
	}

	if _, err := os.Stat(i.opts.ConfigFile); err != nil {
		logger.Info("no config file, skipping account sync", "path", i.opts.ConfigFile)
		return nil
	}

	data, err := os.ReadFile(i.opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	desired := make(map[string]bool, len(cfg.Accounts))
	for _, a := range cfg.Accounts {
		desired[strings.ToUpper(a.Username)] = true
	}

	rows, err := db.QueryContext(ctx, "SELECT username FROM account")
	if err != nil {
		return fmt.Errorf("query accounts: %w", err)
	}
	var existing []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			rows.Close()
			return fmt.Errorf("scan account: %w", err)
		}
		existing = append(existing, username)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate accounts: %w", err)
	}

	for _, username := range existing {
		if !desired[username] {
			logger.Info("deleting stale account", "username", username)
			if _, err := db.ExecContext(ctx, "DELETE FROM account WHERE username = ?", username); err != nil {
				return fmt.Errorf("delete account %s: %w", username, err)
			}
		}
	}

	for _, a := range cfg.Accounts {
		salt, verifier, err := srpVerifier(a.Username, a.Password)
		if err != nil {
			return fmt.Errorf("srp verifier for %s: %w", a.Username, err)
		}

		username := strings.ToUpper(a.Username)
		logger.Info("upserting account", "username", username)

		_, err = db.ExecContext(ctx, `
			INSERT INTO account
				(username, salt, verifier, email, reg_mail, joindate, last_ip, last_attempt_ip,
				 failed_logins, locked, lock_country, online, expansion, Flags,
				 mutetime, mutereason, muteby, locale, os, recruiter, totaltime)
			VALUES
				(?, ?, ?, '', '', NOW(), '127.0.0.1', '127.0.0.1',
				 0, 0, '00', 0, 2, 0,
				 0, '', '', 0, '', 0, 0)
			ON DUPLICATE KEY UPDATE salt = VALUES(salt), verifier = VALUES(verifier)
		`, username, salt, verifier)
		if err != nil {
			return fmt.Errorf("upsert account %s: %w", username, err)
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO account_access (id, gmlevel, RealmID)
			SELECT id, ?, -1 FROM account WHERE username = ?
			ON DUPLICATE KEY UPDATE gmlevel = VALUES(gmlevel)
		`, a.GmLevel, username)
		if err != nil {
			return fmt.Errorf("upsert gm level %s: %w", username, err)
		}
	}

	return nil
}

// srpVerifier computes the SRP-6 salt and verifier for AzerothCore account creation.
func srpVerifier(username, password string) (salt, verifier []byte, err error) {
	g := big.NewInt(7)
	N := new(big.Int)
	N.SetString("894B645E89E1535BBDAD5B8B290650530801B18EBFBF5E8FAB3C82872A3E9BB7", 16)

	salt = make([]byte, 32)
	if _, err = rand.Read(salt); err != nil {
		return
	}

	h := sha1.New()
	h.Write([]byte(strings.ToUpper(username) + ":" + strings.ToUpper(password)))
	innerHash := h.Sum(nil)

	h = sha1.New()
	h.Write(salt)
	h.Write(innerHash)
	xHash := h.Sum(nil)

	// Reverse bytes: treat the SHA1 output as a little-endian integer
	for lo, hi := 0, len(xHash)-1; lo < hi; lo, hi = lo+1, hi-1 {
		xHash[lo], xHash[hi] = xHash[hi], xHash[lo]
	}
	x := new(big.Int).SetBytes(xHash)

	v := new(big.Int).Exp(g, x, N)

	// Zero-pad to 32 bytes (big-endian), then reverse to little-endian for storage
	vBE := make([]byte, 32)
	copy(vBE[32-len(v.Bytes()):], v.Bytes())
	verifier = make([]byte, 32)
	for idx, b := range vBE {
		verifier[31-idx] = b
	}

	return
}

type dbInfo struct {
	host   string
	port   string
	user   string
	pass   string
	dbname string
}

func parseDBInfo(s string) (*dbInfo, error) {
	parts := strings.SplitN(s, ";", 5)
	if len(parts) != 5 {
		return nil, fmt.Errorf("expected host;port;user;pass;dbname, got %q", s)
	}
	return &dbInfo{
		host:   parts[0],
		port:   parts[1],
		user:   parts[2],
		pass:   parts[3],
		dbname: parts[4],
	}, nil
}

func (d *dbInfo) dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", d.user, d.pass, d.host, d.port, d.dbname)
}

func (d *dbInfo) adminDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/", d.user, d.pass, d.host, d.port)
}
