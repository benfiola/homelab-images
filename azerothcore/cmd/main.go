package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/benfiola/homelab-images/azerothcore/internal"
	"github.com/benfiola/homelab-images/shared/pkg/cliutil"
	"github.com/urfave/cli/v3"
)

func main() {
	cliutil.Run(
		cliutil.Setup(&cli.Command{
			Name:    "azerothcore",
			Version: internal.Version,
			Commands: []*cli.Command{
				initCmd(),
				authserverCmd(),
				worldserverCmd(),
				dbimportCmd(),
				webCmd(),
			},
		}),
	)
}

// setDefaultEnv sets key to def unless it's already present in the
// environment.
func setDefaultEnv(key, def string) {
	if os.Getenv(key) == "" {
		os.Setenv(key, def)
	}
}

func initCmd() *cli.Command {
	return &cli.Command{
		Name: "init",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "game-data-url",
				Required: true,
				Sources:  cli.EnvVars("AC_GAME_DATA_URL"),
			},
			&cli.StringFlag{
				Name:    "realmlist-address",
				Sources: cli.EnvVars("AC_REALMLIST_ADDRESS"),
			},
			&cli.StringFlag{
				Name:    "admin-username",
				Sources: cli.EnvVars("AC_ADMIN_USERNAME"),
				Value:   "admin",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			i, err := internal.New(&internal.Opts{
				GameDataURL:      c.String("game-data-url"),
				RealmlistAddress: c.String("realmlist-address"),
				AdminUsername:    c.String("admin-username"),
			})
			if err != nil {
				return err
			}
			return i.Run(ctx)
		},
	}
}

func authserverCmd() *cli.Command {
	return &cli.Command{
		Name: "authserver",
		Action: func(ctx context.Context, c *cli.Command) error {
			setDefaultEnv("AC_DATA_DIR", "/data")
			setDefaultEnv("AC_LOGS_DIR", "/logs")
			setDefaultEnv("AC_TEMP_DIR", "/tmp/azerothcore")
			// migrations are the init step's responsibility (via dbimport)
			os.Setenv("AC_UPDATES_ENABLE_DATABASES", "0")
			os.Setenv("AC_DISABLE_INTERACTIVE", "1")

			if err := copyAllConfsIfMissing(); err != nil {
				return err
			}

			binary, err := exec.LookPath("authserver")
			if err != nil {
				return err
			}
			return syscall.Exec(binary, []string{"authserver"}, os.Environ())
		},
	}
}

func worldserverCmd() *cli.Command {
	return &cli.Command{
		Name: "worldserver",
		Action: func(ctx context.Context, c *cli.Command) error {
			setDefaultEnv("AC_DATA_DIR", "/data")
			setDefaultEnv("AC_LOGS_DIR", "/logs")
			setDefaultEnv("AC_TEMP_DIR", "/tmp/azerothcore")
			// migrations are the init step's responsibility (via dbimport)
			os.Setenv("AC_UPDATES_ENABLE_DATABASES", "0")
			os.Setenv("AC_DISABLE_INTERACTIVE", "1")
			// the "web" command needs to reach this over SOAP from a
			// separate container, so bind all interfaces rather than the
			// conf.dist default of localhost-only
			setDefaultEnv("AC_SOAP_ENABLED", "1")
			setDefaultEnv("AC_SOAP_IP", "0.0.0.0")

			if err := copyAllConfsIfMissing(); err != nil {
				return err
			}

			binary, err := exec.LookPath("worldserver")
			if err != nil {
				return err
			}
			return syscall.Exec(binary, []string{"worldserver"}, os.Environ())
		},
	}
}

func dbimportCmd() *cli.Command {
	return &cli.Command{
		Name: "dbimport",
		Action: func(ctx context.Context, c *cli.Command) error {
			setDefaultEnv("AC_DATA_DIR", "/data")
			setDefaultEnv("AC_LOGS_DIR", "/logs")
			setDefaultEnv("AC_TEMP_DIR", "/tmp/azerothcore")

			// dbimport bakes its mysql client path in at compile time (via
			// cmake's find_program) when creating a database - override it so
			// runtime behavior doesn't depend on what the build environment
			// happened to find.
			mysql := os.Getenv("AC_MY_SQLEXECUTABLE")
			if mysql == "" {
				var err error
				mysql, err = exec.LookPath("mysql")
				if err != nil {
					return fmt.Errorf("mysql client not found: set AC_MY_SQLEXECUTABLE")
				}
				os.Setenv("AC_MY_SQLEXECUTABLE", mysql)
			}

			if err := copyAllConfsIfMissing(); err != nil {
				return err
			}

			binary, err := exec.LookPath("dbimport")
			if err != nil {
				return err
			}
			return syscall.Exec(binary, []string{"dbimport"}, os.Environ())
		},
	}
}

func webCmd() *cli.Command {
	return &cli.Command{
		Name: "web",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "listen-address",
				Sources: cli.EnvVars("AC_WEB_LISTEN_ADDRESS"),
				Value:   ":8080",
			},
			&cli.StringFlag{
				Name:     "soap-address",
				Required: true,
				Sources:  cli.EnvVars("AC_WEB_SOAP_ADDRESS"),
			},
			&cli.IntFlag{
				Name:    "admin-gm-level-threshold",
				Sources: cli.EnvVars("AC_WEB_ADMIN_GM_LEVEL_THRESHOLD"),
				Value:   3,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			w, err := internal.NewWeb(&internal.WebOpts{
				ListenAddress:         c.String("listen-address"),
				SOAPAddress:           c.String("soap-address"),
				AdminGMLevelThreshold: int(c.Int("admin-gm-level-threshold")),
			})
			if err != nil {
				return err
			}
			return w.Run(ctx)
		},
	}
}

// copyAllConfsIfMissing copies every "*.conf.dist" file under
// /azerothcore/env/ref/etc to its "*.conf" counterpart, if missing.
func copyAllConfsIfMissing() error {
	root := "/azerothcore/env/ref/etc"
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".conf.dist") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		return copyConfIfMissing(rel)
	})
}

// copyConfIfMissing copies relPath (a "*.conf.dist" file relative to
// /azerothcore/env/ref/etc) to its "*.conf" counterpart under
// /azerothcore/env/dist/etc, if the destination doesn't already exist.
func copyConfIfMissing(relPath string) error {
	src := filepath.Join("/azerothcore/env/ref/etc", relPath)
	dst := filepath.Join("/azerothcore/env/dist/etc", strings.TrimSuffix(relPath, ".dist"))
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
