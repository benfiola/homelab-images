package main

import (
	"fmt"
	"strings"
)

// release creates the component's git tag and github release.
func release(component, version string) error {
	v, _ := parseSemver(version)
	tag := component + "/" + version

	previous, _, err := latestRelease(component)
	if err != nil {
		return err
	}
	deps, err := listDependencies(component)
	if err != nil {
		return err
	}
	specs, err := deps.pathspecs(previous)
	if err != nil {
		return err
	}
	rev := "HEAD"
	if previous != "" {
		rev = previous + "..HEAD"
	}
	changes, err := output("git", append([]string{"log", "--oneline", rev, "--"}, specs...)...)
	if err != nil {
		return err
	}

	var notes strings.Builder
	fmt.Fprintf(&notes, "## %s\n\n", version)
	if previous != "" {
		fmt.Fprintf(&notes, "[Full diff](%s/compare/%s...%s)\n\n", repoURL, previous, tag)
	}
	fmt.Fprintf(&notes, "### Docker Image\n\n```\ndocker pull %s/%s:%s\n```\n\n", imageBase, component, v.imageTag())
	if exists(component, "chart") {
		pull := fmt.Sprintf("helm pull oci://%s/charts/%s --version %s", imageBase, component, v.helmVersion())
		if v.prerelease != "" {
			pull += " --devel"
		}
		fmt.Fprintf(&notes, "### Helm Chart\n\n```\n%s\n```\n\n", pull)
	}
	fmt.Fprintf(&notes, "### Changes\n\n%s\n", changes)

	sha, err := output("git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	args := []string{"release", "create", tag, "--target", sha, "--title", component + " " + version, "--notes", notes.String()}
	if v.prerelease != "" {
		args = append(args, "--prerelease")
	}
	return stream(nil, "gh", args...)
}
