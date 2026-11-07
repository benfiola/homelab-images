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

	// branches release what changed since their last release, or since forking from main
	branch := currentBranch()
	fork := ""
	if branch != "main" {
		if fork, err = output("git", "merge-base", "origin/main", "HEAD"); err != nil {
			return err
		}
	}

	releases := []plannedRelease{}
	for _, component := range components {
		tag, last, err := latestRelease(component, "")
		if err != nil {
			return err
		}
		since := tag
		if fork != "" {
			since = fork
			previous, err := latestBranchRelease(component, branch)
			if err != nil {
				return err
			}
			if previous != "" {
				since = previous
			}
		}
		deps, err := listDependencies(component)
		if err != nil {
			return err
		}
		specs, err := deps.pathspecs(since)
		if err != nil {
			return err
		}
		commits, err := commitsSince(since, specs)
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
