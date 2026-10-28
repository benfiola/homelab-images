module github.com/benfiola/homelab-images/linstor-provision-disk

go 1.27.3

require (
	github.com/benfiola/homelab-images/shared v0.0.0
	github.com/urfave/cli/v3 v3.14.0
)

replace github.com/benfiola/homelab-images/shared => ../shared
