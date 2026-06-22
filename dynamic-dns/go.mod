module github.com/benfiola/homelab-images/dynamic-dns

go 1.25.8

require (
	github.com/benfiola/homelab-images/shared v0.0.0
	github.com/cloudflare/cloudflare-go/v6 v6.10.0
	github.com/urfave/cli/v3 v3.10.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
)

replace github.com/benfiola/homelab-images/shared => ../shared
