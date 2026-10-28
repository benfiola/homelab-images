module github.com/benfiola/homelab-images/dynamic-dns

go 1.27.3

require (
	github.com/benfiola/homelab-images/shared v0.0.0
	github.com/cloudflare/cloudflare-go/v7 v7.13.1
	github.com/urfave/cli/v3 v3.14.0
)

require (
	github.com/tidwall/gjson v1.20.0 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.2 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
)

replace github.com/benfiola/homelab-images/shared => ../shared
