module github.com/benfiola/homelab-images/vault-push-secrets

go 1.25.8

require (
	github.com/benfiola/homelab-images/shared v0.0.0
	github.com/bitwarden/sdk-go/v2 v2.1.0
	github.com/goccy/go-yaml v1.19.2
	github.com/hashicorp/vault-client-go v0.4.3
	github.com/urfave/cli/v3 v3.10.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)

replace github.com/benfiola/homelab-images/shared => ../shared
