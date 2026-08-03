package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/benfiola/homelab-images/shared/pkg/cache"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/benfiola/homelab-images/shared/pkg/steam"
)

const (
	AppId   = 2394010
	DepotId = 2394012
)

// DownloadGame installs the Palworld dedicated server into gamePath, using c
// as a squashfs cache keyed by Steam manifest id so repeat deploys of the
// same version skip re-downloading. Returns the manifest id actually
// installed (never 0, even if manifestId was 0/latest).
func DownloadGame(ctx context.Context, c *cache.Cache, manifestId int, gamePath string) (int, error) {
	logger := logging.FromContext(ctx)

	if manifestId == 0 {
		logger.Info("determining latest manifest id", "app", AppId, "depot", DepotId)
		latest, err := steam.GetLatestManifestId(ctx, AppId, DepotId)
		if err != nil {
			return 0, err
		}
		manifestId = latest
	}

	key := fmt.Sprintf("palworld-%d", manifestId)

	if !c.Exists(ctx, key) {
		logger.Info("downloading game", "app", AppId, "depot", DepotId, "manifest", manifestId)
		if err := steam.Download(ctx, AppId, DepotId, manifestId, gamePath); err != nil {
			return 0, err
		}
		logger.Info("caching game", "key", key)
		if err := c.Put(ctx, key, gamePath); err != nil {
			return 0, err
		}
	} else {
		logger.Info("using cached game", "app", AppId, "depot", DepotId, "manifest", manifestId)
		if err := c.Get(ctx, key, gamePath); err != nil {
			return 0, err
		}
	}

	serverScript := filepath.Join(gamePath, "PalServer.sh")
	if err := os.Chmod(serverScript, 0755); err != nil {
		logger.Error("failed to chmod server script", "path", serverScript, "error", err)
	}

	return manifestId, nil
}

// InstallSteamClient places steamclient.so where PalServer.sh expects it -
// DepotDownloader never populates it (see Dockerfile for where
// /opt/steamclient.so comes from), and PalServer.sh hard-exits without it.
// No-op if already present.
func InstallSteamClient(gamePath string) error {
	dst := filepath.Join(gamePath, "Pal", "Binaries", "Linux", "steamclient.so")
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	data, err := os.ReadFile("/opt/steamclient.so")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// LinkSaveDir symlinks gamePath/Pal/Saved to dataPath, the one real
// persistent volume (gamePath itself is scratch, re-derived from the cache
// on every start). Safe to call repeatedly: removes the target first.
func LinkSaveDir(gamePath, dataPath string) error {
	savedPath := filepath.Join(gamePath, "Pal", "Saved")
	if err := os.MkdirAll(filepath.Dir(savedPath), 0755); err != nil {
		return err
	}
	if err := os.RemoveAll(savedPath); err != nil {
		return err
	}
	return os.Symlink(dataPath, savedPath)
}

// StartServer builds the unstarted command to launch PalServer. The
// supervisor starts/tracks/waits on it directly, since Reboot() needs to
// signal the running process.
func StartServer(gamePath string, port int) *exec.Cmd {
	cmd := exec.Command("./PalServer.sh",
		fmt.Sprintf("-port=%d", port),
		"-useperfthreads",
		"-NoAsyncLoadingThread",
		"-UseMultithreadForDS",
	)
	cmd.Dir = gamePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}
