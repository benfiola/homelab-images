package main

import (
	"context"
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
				Name:    "data-dir",
				Sources: cli.EnvVars("AC_DATA_DIR"),
				Value:   "/data",
			},
			&cli.StringFlag{
				Name:     "login-db",
				Required: true,
				Sources:  cli.EnvVars("AC_LOGIN_DATABASE_INFO"),
			},
			&cli.StringFlag{
				Name:     "world-db",
				Required: true,
				Sources:  cli.EnvVars("AC_WORLD_DATABASE_INFO"),
			},
			&cli.StringFlag{
				Name:     "character-db",
				Required: true,
				Sources:  cli.EnvVars("AC_CHARACTER_DATABASE_INFO"),
			},
			&cli.StringFlag{
				Name:     "playerbots-db",
				Required: true,
				Sources:  cli.EnvVars("AC_PLAYERBOTS_DATABASE_INFO"),
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
				DataDir:          c.String("data-dir"),
				LoginDB:          c.String("login-db"),
				WorldDB:          c.String("world-db"),
				CharacterDB:      c.String("character-db"),
				PlayerbotsDB:     c.String("playerbots-db"),
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
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "login-db",
				Required: true,
				Sources:  cli.EnvVars("AC_LOGIN_DATABASE_INFO"),
			},
			&cli.StringFlag{
				Name:    "data-dir",
				Sources: cli.EnvVars("AC_DATA_DIR"),
				Value:   "/data",
			},
			&cli.StringFlag{
				Name:    "logs-dir",
				Sources: cli.EnvVars("AC_LOGS_DIR"),
				Value:   "/logs",
			},
			&cli.StringFlag{
				Name:    "temp-dir",
				Sources: cli.EnvVars("AC_TEMP_DIR"),
				Value:   "/tmp/azerothcore",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			os.Setenv("AC_LOGIN_DATABASE_INFO", c.String("login-db"))
			os.Setenv("AC_DATA_DIR", c.String("data-dir"))
			os.Setenv("AC_LOGS_DIR", c.String("logs-dir"))
			os.Setenv("AC_TEMP_DIR", c.String("temp-dir"))

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
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "login-db",
				Required: true,
				Sources:  cli.EnvVars("AC_LOGIN_DATABASE_INFO"),
			},
			&cli.StringFlag{
				Name:     "world-db",
				Required: true,
				Sources:  cli.EnvVars("AC_WORLD_DATABASE_INFO"),
			},
			&cli.StringFlag{
				Name:     "character-db",
				Required: true,
				Sources:  cli.EnvVars("AC_CHARACTER_DATABASE_INFO"),
			},
			&cli.StringFlag{
				Name:     "playerbots-db",
				Required: true,
				Sources:  cli.EnvVars("AC_PLAYERBOTS_DATABASE_INFO"),
			},
			&cli.StringFlag{
				Name:    "data-dir",
				Sources: cli.EnvVars("AC_DATA_DIR"),
				Value:   "/data",
			},
			&cli.StringFlag{
				Name:    "logs-dir",
				Sources: cli.EnvVars("AC_LOGS_DIR"),
				Value:   "/logs",
			},
			&cli.StringFlag{
				Name:    "temp-dir",
				Sources: cli.EnvVars("AC_TEMP_DIR"),
				Value:   "/tmp/azerothcore",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			os.Setenv("AC_LOGIN_DATABASE_INFO", c.String("login-db"))
			os.Setenv("AC_WORLD_DATABASE_INFO", c.String("world-db"))
			os.Setenv("AC_CHARACTER_DATABASE_INFO", c.String("character-db"))
			os.Setenv("AC_PLAYERBOTS_DATABASE_INFO", c.String("playerbots-db"))
			os.Setenv("AC_DATA_DIR", c.String("data-dir"))
			os.Setenv("AC_LOGS_DIR", c.String("logs-dir"))
			os.Setenv("AC_TEMP_DIR", c.String("temp-dir"))

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
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "login-db",
				Required: true,
				Sources:  cli.EnvVars("AC_LOGIN_DATABASE_INFO"),
			},
			&cli.StringFlag{
				Name:     "world-db",
				Required: true,
				Sources:  cli.EnvVars("AC_WORLD_DATABASE_INFO"),
			},
			&cli.StringFlag{
				Name:     "character-db",
				Required: true,
				Sources:  cli.EnvVars("AC_CHARACTER_DATABASE_INFO"),
			},
			&cli.StringFlag{
				Name:     "playerbots-db",
				Required: true,
				Sources:  cli.EnvVars("AC_PLAYERBOTS_DATABASE_INFO"),
			},
			&cli.StringFlag{
				Name:    "data-dir",
				Sources: cli.EnvVars("AC_DATA_DIR"),
				Value:   "/data",
			},
			&cli.StringFlag{
				Name:    "logs-dir",
				Sources: cli.EnvVars("AC_LOGS_DIR"),
				Value:   "/logs",
			},
			&cli.StringFlag{
				Name:    "temp-dir",
				Sources: cli.EnvVars("AC_TEMP_DIR"),
				Value:   "/tmp/azerothcore",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			os.Setenv("AC_LOGIN_DATABASE_INFO", c.String("login-db"))
			os.Setenv("AC_WORLD_DATABASE_INFO", c.String("world-db"))
			os.Setenv("AC_CHARACTER_DATABASE_INFO", c.String("character-db"))
			os.Setenv("AC_PLAYERBOTS_DATABASE_INFO", c.String("playerbots-db"))
			os.Setenv("AC_DATA_DIR", c.String("data-dir"))
			os.Setenv("AC_LOGS_DIR", c.String("logs-dir"))
			os.Setenv("AC_TEMP_DIR", c.String("temp-dir"))

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
