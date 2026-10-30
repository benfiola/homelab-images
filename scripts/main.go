// Command scripts plans, builds and releases the repo's components.
//
// A component is a top-level directory containing a Dockerfile. Every
// component is versioned independently via '<component>/v<semver>' git tags.
package main

import (
	"fmt"
	"os"
	"strings"
)

const (
	repoURL    = "https://github.com/benfiola/homelab-images"
	modulePath = "github.com/benfiola/homelab-images"
	imageBase  = "ghcr.io/benfiola/homelab-images"
	// defaultPlatforms can be overridden per component (see componentPlatforms)
	defaultPlatforms = "linux/amd64"
)

const usage = `usage: go run ./scripts <command>

commands:
  plan [-all] [-components a,b]  print the components to release and their versions as json
  build <component> <version>    generate code, compile binaries and package the helm chart
  publish <component> <version>  push the container image and helm chart
  release <component> <version>  create the git tag and github release`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%s", usage)
	}

	root, err := output("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	if err := os.Chdir(root); err != nil {
		return err
	}

	command, args := args[0], args[1:]
	if command == "plan" {
		return plan(args)
	}

	if len(args) != 2 {
		return fmt.Errorf("%s", usage)
	}
	component, version := args[0], strings.TrimSpace(args[1])
	if !isComponent(component) {
		return fmt.Errorf("unknown component %q", component)
	}
	if _, ok := parseSemver(version); !ok {
		return fmt.Errorf("version %q is not a semver with a leading 'v'", version)
	}

	switch command {
	case "build":
		return build(component, version)
	case "publish":
		return publish(component, version)
	case "release":
		return release(component, version)
	}
	return fmt.Errorf("unknown command %q\n%s", command, usage)
}
