module github.com/benfiola/homelab-images/vault-unseal

go 1.25.8

require (
	github.com/benfiola/homelab-images/shared v0.0.0
	github.com/hashicorp/vault-client-go v0.4.3
	github.com/urfave/cli/v3 v3.10.0
)

require (
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	golang.org/x/sys v0.4.0 // indirect
	golang.org/x/time v0.0.0-20220922220347-f3bd1da661af // indirect
)

replace github.com/benfiola/homelab-images/shared => ../shared
