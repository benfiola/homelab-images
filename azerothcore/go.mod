module github.com/benfiola/homelab-images/azerothcore

go 1.25.8

replace github.com/benfiola/homelab-images/shared => ../shared

require (
	github.com/benfiola/homelab-images/shared v0.0.0-00010101000000-000000000000
	github.com/go-sql-driver/mysql v1.10.0
	github.com/urfave/cli/v3 v3.10.1
	gopkg.in/yaml.v3 v3.0.1
)

require filippo.io/edwards25519 v1.2.0 // indirect
