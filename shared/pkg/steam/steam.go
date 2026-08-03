package steam

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/cmd"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
)

func Download(ctx context.Context, appId int, depotId int, manifestId int, output string) error {
	return cmd.Stream(ctx, "DepotDownloader", "-app", strconv.Itoa(appId), "-depot", strconv.Itoa(depotId), "-manifest", strconv.Itoa(manifestId), "-dir", output)
}

var regexpManifest = regexp.MustCompile(`(?m)^Manifest ([\d]+).*$`)

func GetLatestManifestId(ctx context.Context, appId int, depotId int) (int, error) {
	tempdir, err := os.MkdirTemp("", "depotdownloader-*")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(tempdir)

	output, err := cmd.Capture(ctx, "DepotDownloader", "-app", strconv.Itoa(appId), "-depot", strconv.Itoa(depotId), "-manifest-only", "-dir", tempdir)
	if err != nil {
		return 0, err
	}
	match := regexpManifest.FindStringSubmatch(output)
	if match == nil {
		return 0, fmt.Errorf("latest manifest for app %d and depot %d not found", appId, depotId)
	}
	manifestId, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, err
	}
	return manifestId, nil
}

// WatchForUpdate polls the latest manifest id for appId/depotId every
// interval, starting from a baseline of currentManifestId, and invokes
// onUpdate with the new id whenever it changes - continuously, not just
// once, updating its own baseline after each fire. Runs until ctx is
// cancelled.
func WatchForUpdate(ctx context.Context, appId int, depotId int, currentManifestId int, interval time.Duration, onUpdate func(ctx context.Context, newManifestId int)) {
	go func() {
		logger := logging.FromContext(ctx)
		last := currentManifestId
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				latest, err := GetLatestManifestId(ctx, appId, depotId)
				if err != nil {
					logger.Error("failed to check for game updates", "error", err)
					continue
				}
				if latest != last {
					logger.Info("new game version detected", "previous", last, "latest", latest)
					last = latest
					onUpdate(ctx, latest)
				}
			}
		}
	}()
}
