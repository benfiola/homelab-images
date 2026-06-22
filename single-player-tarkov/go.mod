module github.com/benfiola/homelab-images/single-player-tarkov

go 1.25.8

require (
	github.com/benfiola/homelab-images/shared v0.0.0
	github.com/urfave/cli/v3 v3.10.0
)

require github.com/evanphx/json-patch/v5 v5.9.11 // indirect

replace github.com/benfiola/homelab-images/shared => ../shared
