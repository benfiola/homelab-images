package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
				Name:    "config",
				Sources: cli.EnvVars("AC_CONFIG"),
				Value:   "/config/config.yaml",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			i, err := internal.New(&internal.Opts{
				GameDataURL:      c.String("game-data-url"),
				RealmlistAddress: c.String("realmlist-address"),
				ConfigFile:       c.String("config"),
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

			if err := copyConfIfMissing("authserver"); err != nil {
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

			for _, conf := range []string{"worldserver", "playerbots"} {
				if err := copyConfIfMissing(conf); err != nil {
					return err
				}
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

			if err := copyConfIfMissing("dbimport"); err != nil {
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

func copyConfIfMissing(name string) error {
	dst := filepath.Join("/azerothcore/env/dist/etc", name+".conf")
	src := filepath.Join("/azerothcore/env/ref/etc", name+".conf.dist")
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
