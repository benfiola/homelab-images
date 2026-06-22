package archive

import (
	"context"

	"github.com/benfiola/homelab-images/shared/pkg/cmd"
)

func Extract(ctx context.Context, source string, dest string) error {
	return cmd.Stream(ctx, "bsdtar", "-x", "-f", source, "-C", dest)
}
