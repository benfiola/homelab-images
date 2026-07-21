module github.com/benfiola/homelab-images/loudnorm

go 1.25.8

require github.com/urfave/cli/v3 v3.10.0

require (
	github.com/benfiola/homelab-images/shared v0.0.0
	go.etcd.io/bbolt v1.5.0
)

require golang.org/x/sys v0.45.0 // indirect

replace github.com/benfiola/homelab-images/shared => ../shared
