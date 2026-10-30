package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
)

type plannedRelease struct {
	Component string `json:"component"`
	Version   string `json:"version"`
}

// plan prints the components with unreleased changes and the version each would get.
func plan(args []string) error {
	flags := flag.NewFlagSet("plan", flag.ContinueOnError)
	all := flags.Bool("all", false, "release every component, even without changes")
	only := flags.String("components", "", "comma-separated components to release, even without changes")
	if err := flags.Parse(args); err != nil {
		return err
	}

	components, err := listComponents()
	if err != nil {
		return err
	}
	forced := map[string]bool{}
	for _, name := range strings.Split(*only, ",") {
		if name = strings.TrimSpace(name); name == "" {
			continue
		}
		if !isComponent(name) {
			return fmt.Errorf("unknown component %q", name)
		}
		forced[name] = true
	}

	releases := []plannedRelease{}
	for _, component := range components {
		tag, last, err := latestRelease(component)
		if err != nil {
			return err
		}
		deps, err := listDependencies(component)
		if err != nil {
			return err
		}
		specs, err := deps.pathspecs(tag)
		if err != nil {
			return err
		}
		commits, err := commitsSince(tag, specs)
		if err != nil {
			return err
		}
		if len(commits) == 0 && !*all && !forced[component] {
			continue
		}
		version := nextVersion(last, tag != "", commits)
		releases = append(releases, plannedRelease{component, version.String()})
	}

	out, err := json.Marshal(releases)
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
