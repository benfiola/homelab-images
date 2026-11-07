package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type semver struct {
	major, minor, patch int
	prerelease, build   string
}

// semverRegex is the semver.org regex with a leading 'v'.
var semverRegex = regexp.MustCompile(`^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$`)

func parseSemver(s string) (semver, bool) {
	m := semverRegex.FindStringSubmatch(s)
	if m == nil {
		return semver{}, false
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return semver{major, minor, patch, m[4], m[5]}, true
}

func (v semver) String() string {
	s := fmt.Sprintf("v%d.%d.%d", v.major, v.minor, v.patch)
	if v.prerelease != "" {
		s += "-" + v.prerelease
	}
	if v.build != "" {
		s += "+" + v.build
	}
	return s
}

func (v semver) less(o semver) bool {
	if v.major != o.major {
		return v.major < o.major
	}
	if v.minor != o.minor {
		return v.minor < o.minor
	}
	return v.patch < o.patch
}

func (v semver) bump(level string) semver {
	switch level {
	case "major":
		return semver{major: v.major + 1}
	case "minor":
		return semver{major: v.major, minor: v.minor + 1}
	}
	return semver{major: v.major, minor: v.minor, patch: v.patch + 1}
}

// imageTag converts a version to an OCI tag: no leading 'v', and '+' (which
// OCI tags disallow) replaced with '_'.
func (v semver) imageTag() string {
	return strings.ReplaceAll(strings.TrimPrefix(v.String(), "v"), "+", "_")
}

// helmVersion is the version without its leading 'v'.
func (v semver) helmVersion() string {
	return strings.TrimPrefix(v.String(), "v")
}

// latestRelease returns the newest stable '<component>/v*' tag (other than exclude), or "" if there is none.
func latestRelease(component, exclude string) (string, semver, error) {
	out, err := output("git", "tag", "--list", component+"/v*")
	if err != nil {
		return "", semver{}, err
	}
	var tag string
	var latest semver
	for line := range strings.SplitSeq(out, "\n") {
		v, ok := parseSemver(strings.TrimPrefix(line, component+"/"))
		if !ok || v.prerelease != "" || v.build != "" || line == exclude {
			continue
		}
		if tag == "" || latest.less(v) {
			tag, latest = line, v
		}
	}
	return tag, latest, nil
}

// commit is the part of a conventional commit that decides the version bump.
type commit struct {
	subject string
	body    string
}

var conventionalRegex = regexp.MustCompile(`^(\w+)(?:\([^)]*\))?(!)?:`)

// level returns the bump implied by a commit: 'major' for breaking changes,
// 'minor' for features and 'patch' for everything else.
func (c commit) level() string {
	match := conventionalRegex.FindStringSubmatch(c.subject)
	switch {
	case match != nil && match[2] == "!", strings.Contains(c.body, "BREAKING CHANGE:"):
		return "major"
	case match != nil && match[1] == "feat":
		return "minor"
	}
	return "patch"
}

// commitsSince lists the commits since ref (or all history if ref is empty)
// that touched the given pathspecs.
func commitsSince(ref string, pathspecs []string) ([]commit, error) {
	rev := "HEAD"
	if ref != "" {
		rev = ref + "..HEAD"
	}
	args := append([]string{"log", rev, "--format=%s%x1f%b%x1e", "--"}, pathspecs...)
	out, err := output("git", args...)
	if err != nil {
		return nil, err
	}
	var commits []commit
	for entry := range strings.SplitSeq(out, "\x1e") {
		subject, body, found := strings.Cut(strings.TrimSpace(entry), "\x1f")
		if !found {
			continue
		}
		commits = append(commits, commit{subject, body})
	}
	return commits, nil
}

func currentBranch() string {
	if branch := os.Getenv("GITHUB_REF_NAME"); branch != "" {
		return branch
	}
	branch, _ := output("git", "branch", "--show-current")
	return branch
}

// branchSuffix returns the prerelease kind and build metadata identifying a
// branch's prereleases. Both are empty for main.
func branchSuffix(branch string) (kind, build string) {
	switch branch {
	case "main":
		return "", ""
	case "dev":
		return "rc", ""
	}
	return "alpha", regexp.MustCompile(`[^0-9A-Za-z-]+`).ReplaceAllString(branch, "-")
}

// latestBranchRelease returns the component's newest prerelease tag for the
// branch that is an ancestor of HEAD, or "" if there is none.
func latestBranchRelease(component, branch string) (string, error) {
	kind, build := branchSuffix(branch)
	out, err := output("git", "tag", "--merged", "HEAD", "--list", component+"/v*")
	if err != nil {
		return "", err
	}
	var tag string
	latest := -1
	for line := range strings.SplitSeq(out, "\n") {
		v, ok := parseSemver(strings.TrimPrefix(line, component+"/"))
		if !ok || v.build != build {
			continue
		}
		prefix, run, _ := strings.Cut(v.prerelease, ".")
		n, err := strconv.Atoi(run)
		if prefix != kind || err != nil {
			continue
		}
		if n > latest {
			tag, latest = line, n
		}
	}
	return tag, nil
}

// nextVersion computes the version for the commits since the last release.
// Releases from main are plain versions; other branches get prerelease versions.
func nextVersion(last semver, hasLast bool, commits []commit) semver {
	next := semver{major: 1}
	if hasLast {
		level := "patch"
		for _, c := range commits {
			switch c.level() {
			case "major":
				level = "major"
			case "minor":
				if level != "major" {
					level = "minor"
				}
			}
		}
		next = last.bump(level)
	}

	branch := currentBranch()
	run := os.Getenv("GITHUB_RUN_NUMBER")
	if run == "" {
		run = "0"
	}
	if kind, build := branchSuffix(branch); kind != "" {
		next.prerelease = kind + "." + run
		next.build = build
	}
	return next
}
