package main

import (
	"bufio"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

const DefaultImageBase = "ghcr.io/benfiola/homelab-images"

// semverRegex is the canonical semver regex from semver.org, extended with a leading 'v'.
var semverRegex = regexp.MustCompile(`^v(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

func parseSemver(v string) map[string]string {
	m := semverRegex.FindStringSubmatch(v)
	if m == nil {
		return nil
	}
	result := make(map[string]string)
	for i, name := range semverRegex.SubexpNames() {
		if name != "" {
			result[name] = m[i]
		}
	}
	return result
}

// helmVersion removes a leading 'v' from a version string.
func helmVersion(version string) string {
	return strings.TrimPrefix(version, "v")
}

// imageTagVersion converts a semver string to a valid OCI image tag by replacing
// the '+' build-metadata separator (not allowed in OCI tags) with '-'.
// The returned tag has no leading 'v' (Docker convention).
func imageTagVersion(version string) (string, error) {
	p := parseSemver(version)
	if p == nil {
		return "", fmt.Errorf("version %q does not match semver", version)
	}
	v := fmt.Sprintf("%s.%s.%s", p["major"], p["minor"], p["patch"])
	if p["prerelease"] != "" {
		v += "-" + p["prerelease"]
	}
	if p["buildmetadata"] != "" {
		v += "_" + p["buildmetadata"]
	}
	return v, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: scripts <command> [args...]\n")
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "generate-k8s-controller":
		generateK8sController()
	case "build-go":
		buildGo()
	case "package-docker":
		packageDocker()
	case "publish-docker":
		publishDocker()
	case "package-helm":
		packageHelm()
	case "publish-helm":
		publishHelm()
	case "get-next-version":
		getNextVersion()
	case "create-github-release":
		createGithubRelease()
	case "detect-components":
		detectComponents()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func buildGo() {
	// Check if cmd/main.go exists first
	if _, err := os.Stat("cmd/main.go"); err != nil {
		return // no-op
	}

	version := os.Getenv("VERSION")
	outputDir := os.Getenv("BUILD_DIR")
	platforms := os.Getenv("PLATFORMS")

	if version == "" || outputDir == "" {
		fmt.Fprintf(os.Stderr, "error: missing required environment variables: VERSION, BUILD_DIR\n")
		os.Exit(1)
	}

	component, err := deriveComponentName()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	modulePath, err := deriveModulePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output dir: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Building Go binary for %s (version: %s)...\n", component, version)

	// Parse platforms or build for current platform only
	var platformsList []string
	if platforms != "" {
		// Parse comma-separated platforms (e.g., "linux/amd64,linux/arm64")
		for p := range strings.SplitSeq(platforms, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				platformsList = append(platformsList, p)
			}
		}
	} else {
		// Use current platform
		platformsList = []string{""}
	}

	for _, platform := range platformsList {
		var goos, goarch string
		var outputName string

		if platform == "" {
			// Current platform
			outputName = component
		} else {
			// Parse "os/arch" format
			parts := strings.Split(platform, "/")
			if len(parts) != 2 {
				fmt.Fprintf(os.Stderr, "error: invalid platform format: %s (expected os/arch)\n", platform)
				os.Exit(1)
			}
			goos = parts[0]
			goarch = parts[1]
			outputName = fmt.Sprintf("%s_%s_%s", component, goos, goarch)
		}

		env := os.Environ()
		if platform != "" {
			env = append(env, fmt.Sprintf("GOOS=%s", goos), fmt.Sprintf("GOARCH=%s", goarch))
		}

		cmd := exec.Command("go", "build",
			"-o", filepath.Join(outputDir, outputName),
			"-ldflags", fmt.Sprintf("-s -w -X %s/internal.Version=%s", modulePath, version),
			"./cmd/main.go",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error building for %s: %v\n", platform, err)
			os.Exit(1)
		}

		fmt.Printf("✓ Built %s/%s\n", outputDir, outputName)
	}
}

func generateK8sController() {
	hasKubebuilder := false

	var errFound = errors.New("found") // sentinel error
	err := filepath.WalkDir(".", func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(string(data), "+kubebuilder:") {
			return errFound
		}
		return nil
	})

	hasKubebuilder = errors.Is(err, errFound)
	if err != nil && !hasKubebuilder {
		fmt.Fprintf(os.Stderr, "error walking directory: %v\n", err)
		os.Exit(1)
	} else if !hasKubebuilder { 
		return // no-op
	}

	component, err := deriveComponentName()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	generatedDir := filepath.Join("chart", "generated")

	// Clean up generated directory
	if err := os.RemoveAll(generatedDir); err != nil {
		fmt.Fprintf(os.Stderr, "error removing generated dir: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(generatedDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating generated dir: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Running controller-gen...")

	// Generate deepcopy implementations
	cmd := exec.Command("controller-gen",
		"object",
		"paths=./internal/...",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error generating deepcopy: %v\n", err)
		os.Exit(1)
	}

	// Generate CRD manifests
	crdsFile := filepath.Join(generatedDir, "crds.yaml")
	crdsOut, err := os.Create(crdsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating crds.yaml: %v\n", err)
		os.Exit(1)
	}
	defer crdsOut.Close()

	cmd = exec.Command("controller-gen",
		"crd",
		"paths=./internal/...",
		"output:stdout",
	)
	cmd.Stdout = crdsOut
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error generating CRDs: %v\n", err)
		os.Exit(1)
	}

	// Generate RBAC manifests
	rbacFile := filepath.Join(generatedDir, "rbac.yaml")
	rbacOut, err := os.Create(rbacFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating rbac.yaml: %v\n", err)
		os.Exit(1)
	}
	defer rbacOut.Close()

	cmd = exec.Command("controller-gen",
		fmt.Sprintf("rbac:roleName=%s", component),
		"paths=./internal/...",
		"output:stdout",
	)
	cmd.Stdout = rbacOut
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error generating RBAC: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Code generation complete")
}

func packageHelm() {
	// Check if chart directory exists first
	if _, err := os.Stat("chart"); err != nil {
		return // no-op
	}

	version := os.Getenv("VERSION")
	outputDir := os.Getenv("BUILD_DIR")

	if version == "" || outputDir == "" {
		fmt.Fprintf(os.Stderr, "error: missing required environment variables: VERSION, BUILD_DIR\n")
		os.Exit(1)
	}

	component, err := deriveComponentName()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output dir: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Packaging Helm chart for %s (version: %s)...\n", component, version)

	bareVer := helmVersion(version)
	cmd := exec.Command("helm", "package", "chart",
		"--app-version", bareVer,
		"--version", bareVer,
		"--destination", outputDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error packaging helm chart: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Packaged Helm chart in %s\n", outputDir)
}

// stageSharedFiles scans Dockerfile for COPY/ADD instructions referencing a
// .shared/<path> source and copies the corresponding file from
// ../shared/<path> into place before the docker build runs, since Docker's
// build context can't reach outside the component directory directly.
// --from=<stage> copies are skipped - those reference a previous build
// stage's filesystem, not the host build context. No-op for components
// whose Dockerfile doesn't reference .shared/ at all.
func stageSharedFiles() error {
	content, err := os.ReadFile("Dockerfile")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, relPath := range sharedFileReferences(string(content)) {
		src := filepath.Join("..", "shared", strings.TrimPrefix(relPath, ".shared/"))
		info, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("failed to stat shared file %s: %w", src, err)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read shared file %s: %w", src, err)
		}
		if err := os.MkdirAll(filepath.Dir(relPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", relPath, err)
		}
		if err := os.WriteFile(relPath, data, info.Mode()); err != nil {
			return fmt.Errorf("failed to write staged file %s: %w", relPath, err)
		}
		fmt.Printf("Staged shared file: %s -> %s\n", src, relPath)
	}
	return nil
}

func packageDocker() {
	// Check if Dockerfile exists first
	if _, err := os.Stat("Dockerfile"); err != nil {
		fmt.Println("No Dockerfile found, skipping Docker build")
		return
	}

	if err := stageSharedFiles(); err != nil {
		fmt.Fprintf(os.Stderr, "error staging shared files: %v\n", err)
		os.Exit(1)
	}

	version := os.Getenv("VERSION")
	platforms := os.Getenv("PLATFORMS")

	if version == "" || platforms == "" {
		fmt.Fprintf(os.Stderr, "error: missing required environment variables: VERSION, PLATFORMS\n")
		os.Exit(1)
	}

	component, err := deriveComponentName()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	imageVersion, err := imageTagVersion(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	imageBase := DefaultImageBase
	imageTag := fmt.Sprintf("%s/%s:%s", imageBase, component, imageVersion)

	fmt.Printf("Building Docker image: %s\n", imageTag)
	fmt.Printf("Platforms: %s\n", platforms)

	cmd := exec.Command("docker", "buildx", "build",
		"--platform", platforms,
		"--tag", imageTag,
		"-f", "Dockerfile",
		".",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error building docker image: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Built %s\n", imageTag)
}

func getNextVersion() {
	component, paths, err := currentComponentPaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	rc := os.Getenv("RC")
	alpha := os.Getenv("ALPHA")
	metadata := os.Getenv("METADATA")

	// Validate ALPHA and METADATA are together
	if alpha == "" && metadata != "" {
		fmt.Fprintf(os.Stderr, "error: METADATA requires ALPHA to be set\n")
		os.Exit(1)
	}
	if alpha != "" && metadata == "" {
		fmt.Fprintf(os.Stderr, "error: ALPHA requires METADATA to be set\n")
		os.Exit(1)
	}

	// Determine version bump level from the commits touching this component
	bumpType := getVersionBumpFromCommits(paths)

	// Build svu command flags
	svuCmd := "next"
	if bumpType != "" {
		svuCmd = bumpType
	}

	svuArgs := []string{
		svuCmd,
		fmt.Sprintf("--tag.prefix=%s/v", component),
		fmt.Sprintf("--tag.pattern=%s/v*", component),
		"--tag.output=v",
	}

	// Only use --always and --log.directory with 'next' command
	if svuCmd == "next" {
		svuArgs = append(svuArgs, "--always", fmt.Sprintf("--log.directory=%s", strings.Join(relativeToComponent(component, paths), ",")))
	}

	if rc != "" {
		svuArgs = append(svuArgs, fmt.Sprintf("--prerelease=rc.%s", rc))
	}
	if alpha != "" {
		svuArgs = append(svuArgs, fmt.Sprintf("--prerelease=alpha.%s", alpha))
	}
	if metadata != "" {
		svuArgs = append(svuArgs, fmt.Sprintf("--metadata=%s", metadata))
	}

	cmd := exec.Command("svu", svuArgs...)
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: svu failed: %v (command: svu %s)\n", err, strings.Join(svuArgs, " "))
		// Emit bootstrap version with prerelease/metadata if needed
		version := "v1.0.0"
		if rc != "" {
			version += fmt.Sprintf("-rc.%s", rc)
		}
		if alpha != "" {
			version += fmt.Sprintf("-alpha.%s", alpha)
		}
		if metadata != "" {
			version += fmt.Sprintf("+%s", metadata)
		}
		fmt.Println(version)
		return
	}

	fmt.Print(strings.TrimSpace(string(output)))
}

// getVersionBumpFromCommits returns the bump implied by the conventional
// commits that touched the component's paths. Scopes are not considered.
func getVersionBumpFromCommits(paths []string) string {
	commitRange := "origin/main...HEAD"
	ref := os.Getenv("GITHUB_REF")
	if ref == "refs/heads/main" || ref == "refs/heads/dev" {
		commitRange = "HEAD~1...HEAD"
	}

	args := []string{"log", commitRange, "--format=%B%n---COMMIT_DELIMITER---", "--"}
	for _, path := range paths {
		args = append(args, ":(top)"+path)
	}
	output, err := exec.Command("git", args...).Output()
	if err != nil {
		output, _ = exec.Command("git", "log", "--format=%B%n---COMMIT_DELIMITER---").Output()
	}

	headerRegex := regexp.MustCompile(`^(feat|fix|chore|docs|style|refactor|perf|test)(\([^)]*\))?(!?):`)
	hasFeature := false
	for _, msg := range strings.Split(string(output), "---COMMIT_DELIMITER---") {
		msg = strings.TrimSpace(msg)
		if msg == "" {
			continue
		}
		lines := strings.Split(msg, "\n")
		if matches := headerRegex.FindStringSubmatch(strings.TrimSpace(lines[0])); matches != nil {
			if matches[3] == "!" {
				return "major"
			}
			if matches[1] == "feat" {
				hasFeature = true
			}
		}
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "BREAKING CHANGE:") {
				return "major"
			}
		}
	}

	if hasFeature {
		return "minor"
	}
	return "patch"
}

func detectComponents() {
	repoRoot := os.Getenv("REPO_ROOT")
	if repoRoot == "" {
		repoRoot = "."
	}

	buildAll := os.Getenv("BUILD_ALL") == "true"
	manualComponents := os.Getenv("MANUAL_COMPONENTS")

	var components []string

	if buildAll {
		// Get all components
		components = getAllComponents(repoRoot)
	} else if manualComponents != "" {
		// Use manually specified components
		components = strings.Split(manualComponents, ",")
		for i, c := range components {
			components[i] = strings.TrimSpace(c)
		}
	} else {
		// Detect components from changed files
		components = getChangedComponents(repoRoot)
	}

	// Output as JSON array
	fmt.Printf("[%s]\n", strings.Join(quote(components), ","))
}

func getAllComponents(repoRoot string) []string {
	excluded := map[string]bool{
		".github": true,
		"vendor":  true,
		"shared":  true,
	}

	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading repo: %v\n", err)
		os.Exit(1)
	}

	var components []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name()[0] == '.' || excluded[entry.Name()] {
			continue
		}

		// Check for Makefile
		makefilePath := filepath.Join(repoRoot, entry.Name(), "Makefile")
		if _, err := os.Stat(makefilePath); err == nil {
			components = append(components, entry.Name())
		}
	}

	sort.Strings(components)
	return components
}

func deriveComponentName() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	return filepath.Base(cwd), nil
}

func deriveModulePath() (string, error) {
	return readModulePath("go.mod")
}

func readModulePath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", path, err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		if mod, ok := strings.CutPrefix(scanner.Text(), "module "); ok {
			return mod, nil
		}
	}

	return "", fmt.Errorf("module directive not found in %s", path)
}

func quote(s []string) []string {
	result := make([]string, len(s))
	for i, v := range s {
		result[i] = fmt.Sprintf("\"%s\"", v)
	}
	return result
}

func publishDocker() {
	// Check if Dockerfile exists first
	if _, err := os.Stat("Dockerfile"); err != nil {
		fmt.Println("No Dockerfile found, skipping Docker push")
		return
	}

	version := os.Getenv("VERSION")
	platforms := os.Getenv("PLATFORMS")

	if version == "" || platforms == "" {
		fmt.Fprintf(os.Stderr, "error: missing required environment variables: VERSION, PLATFORMS\n")
		os.Exit(1)
	}

	component, err := deriveComponentName()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	imageVersion, err := imageTagVersion(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	imageBase := DefaultImageBase
	imageTag := fmt.Sprintf("%s/%s:%s", imageBase, component, imageVersion)

	fmt.Printf("Pushing Docker image: %s\n", imageTag)
	fmt.Printf("Platforms: %s\n", platforms)

	cmd := exec.Command("docker", "buildx", "build",
		"--platform", platforms,
		"--push",
		"--tag", imageTag,
		"-f", "Dockerfile",
		".",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error pushing docker image: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Pushed %s\n", imageTag)
}

func publishHelm() {
	// Check if chart exists first
	if _, err := os.Stat("chart"); err != nil {
		fmt.Println("No chart directory found, skipping Helm push")
		return
	}

	version := os.Getenv("VERSION")
	if version == "" {
		fmt.Fprintf(os.Stderr, "error: missing required environment variable: VERSION\n")
		os.Exit(1)
	}

	component, err := deriveComponentName()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	bareVer := helmVersion(version)
	chartFile := filepath.Join(".build", fmt.Sprintf("%s-%s.tgz", component, bareVer))

	// Check if chart file was built
	if _, err := os.Stat(chartFile); err != nil {
		fmt.Fprintf(os.Stderr, "error: chart not found at %s\n", chartFile)
		os.Exit(1)
	}

	imageBase := DefaultImageBase
	ociRef := fmt.Sprintf("oci://%s/charts", imageBase)
	chartRef := fmt.Sprintf("%s/%s:%s", ociRef, component, bareVer)

	fmt.Printf("Pushing Helm chart: %s\n", chartRef)

	cmd := exec.Command("helm", "push", chartFile, ociRef)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error pushing helm chart: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Pushed %s\n", chartRef)
}

func createGithubRelease() {
	version := os.Getenv("VERSION")
	if version == "" {
		fmt.Fprintf(os.Stderr, "error: missing required environment variable: VERSION\n")
		os.Exit(1)
	}

	component, err := deriveComponentName()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	parsed := parseSemver(version)
	if parsed == nil {
		fmt.Fprintf(os.Stderr, "error: version %q does not match semver\n", version)
		os.Exit(1)
	}

	isPrerelease := parsed["prerelease"] != "" || parsed["buildmetadata"] != ""
	buildMetadata := parsed["buildmetadata"] // non-empty for branch-scoped pre-releases

	// Get previous tag
	tagPrefix := fmt.Sprintf("%s/", component)
	listCmd := exec.Command("git", "tag", "-l", fmt.Sprintf("%s*", tagPrefix), "--sort=-version:refname")
	tagsOutput, err := listCmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting git tags: %v\n", err)
		os.Exit(1)
	}

	tags := strings.Split(strings.TrimSpace(string(tagsOutput)), "\n")
	var previousTag string
	currentTag := fmt.Sprintf("%s%s", tagPrefix, version)

	// Find the previous tag in the same branch lineage.
	// When buildMetadata is set (e.g. +my-branch), restrict to tags whose parsed
	// buildmetadata matches — otherwise we'd cross into another branch's pre-releases.
	for _, tag := range tags {
		if tag == currentTag || tag == "" {
			continue
		}
		if buildMetadata != "" {
			tagVersion := strings.TrimPrefix(tag, tagPrefix)
			tagParsed := parseSemver(tagVersion)
			if tagParsed == nil || tagParsed["buildmetadata"] != buildMetadata {
				continue
			}
		}
		previousTag = tag
		break
	}

	// Generate changelog — restrict to commits touching this component or the parts of shared/ it uses
	_, paths, err := currentComponentPaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	logArgs := []string{"log", currentTag, "--oneline", "--"}
	if previousTag != "" {
		logArgs = []string{"log", fmt.Sprintf("%s...%s", previousTag, currentTag), "--oneline", "--"}
	}
	for _, path := range paths {
		logArgs = append(logArgs, ":(top)"+path)
	}
	changelogCmd := exec.Command("git", logArgs...)

	changelogOutput, err := changelogCmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting changelog: %v\n", err)
		os.Exit(1)
	}

	// OCI tags don't allow '+'; normalize via imageTagVersion
	imageVersion, err := imageTagVersion(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Build release notes with comparison link
	var releaseNotes strings.Builder
	fmt.Fprintf(&releaseNotes, "## [%s](https://github.com/benfiola/homelab-images/compare/%s...%s) (%s)\n\n",
		version, previousTag, currentTag, time.Now().Format("2006-01-02"))

	if _, err := os.Stat("Dockerfile"); err == nil {
		fmt.Fprintf(&releaseNotes, "### Docker Image\n\n")
		fmt.Fprintf(&releaseNotes, "```\ndocker pull %s/%s:%s\n```\n\n", DefaultImageBase, component, imageVersion)
	}

	if _, err := os.Stat("chart"); err == nil {
		fmt.Fprintf(&releaseNotes, "### Helm Chart\n\n")
		helmPull := fmt.Sprintf("helm pull oci://%s/charts/%s --version %s", DefaultImageBase, component, helmVersion(version))
		if isPrerelease {
			helmPull += " --devel"
		}
		fmt.Fprintf(&releaseNotes, "```\n%s\n```\n\n", helmPull)
	}

	fmt.Fprintf(&releaseNotes, "### Changes\n\n")
	fmt.Fprintf(&releaseNotes, "%s", string(changelogOutput))

	// Create release with gh CLI
	ghArgs := []string{
		"release", "create",
		currentTag,
		"--title", fmt.Sprintf("%s %s", component, version),
		"--notes", releaseNotes.String(),
	}

	if isPrerelease {
		ghArgs = append(ghArgs, "--prerelease")
	}

	cmd := exec.Command("gh", ghArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error creating github release: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Created GitHub release %s\n", currentTag)
}

// A component is built, versioned and released from the paths it depends on:
// its own directory plus only the parts of shared/ that it uses. Everything
// that asks "did this component change?" (build detection, version bumps,
// changelogs) goes through componentPaths so they agree.

var (
	dockerfileCopyRegex = regexp.MustCompile(`(?m)^\s*(?:COPY|ADD)\s+(.*)$`)
	sharedFileRegex     = regexp.MustCompile(`\.shared/\S+`)
)

// sharedFileReferences returns the '.shared/<path>' sources a Dockerfile copies
// from shared/ (see stageSharedFiles).
func sharedFileReferences(dockerfile string) []string {
	var refs []string
	for _, lineMatch := range dockerfileCopyRegex.FindAllStringSubmatch(dockerfile, -1) {
		line := lineMatch[1]
		if strings.Contains(line, "--from=") {
			continue
		}
		for _, ref := range sharedFileRegex.FindAllString(line, -1) {
			if !slices.Contains(refs, ref) {
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

// parseImports returns the import paths of a go source file.
func parseImports(path string) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var imports []string
	for _, spec := range file.Imports {
		imp, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		imports = append(imports, imp)
	}
	return imports, nil
}

func isGoSource(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// componentPaths returns the repo-relative paths a component depends on.
func componentPaths(repoRoot string, component string) ([]string, error) {
	sharedModule, err := readModulePath(filepath.Join(repoRoot, "shared", "go.mod"))
	if err != nil {
		return nil, err
	}

	// shared/ package directories the component imports, directly or transitively
	sharedPkgs := map[string]bool{}
	var queue []string
	addImports := func(imports []string) {
		for _, imp := range imports {
			var dir string
			switch {
			case imp == sharedModule:
				dir = "shared"
			case strings.HasPrefix(imp, sharedModule+"/"):
				dir = "shared/" + strings.TrimPrefix(imp, sharedModule+"/")
			default:
				continue
			}
			if !sharedPkgs[dir] {
				sharedPkgs[dir] = true
				queue = append(queue, dir)
			}
		}
	}

	err = filepath.WalkDir(filepath.Join(repoRoot, component), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !isGoSource(entry.Name()) {
			return err
		}
		imports, err := parseImports(path)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}
		addImports(imports)
		return nil
	})
	if err != nil {
		return nil, err
	}

	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]
		entries, err := os.ReadDir(filepath.Join(repoRoot, dir))
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || !isGoSource(entry.Name()) {
				continue
			}
			imports, err := parseImports(filepath.Join(repoRoot, dir, entry.Name()))
			if err != nil {
				return nil, err
			}
			addImports(imports)
		}
	}

	paths := []string{component}
	for dir := range sharedPkgs {
		paths = append(paths, dir)
	}
	if len(sharedPkgs) > 0 {
		// shared's dependencies
		paths = append(paths, "shared/go.mod", "shared/go.sum")
	}

	// shared files copied into the docker build
	dockerfile, err := os.ReadFile(filepath.Join(repoRoot, component, "Dockerfile"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, ref := range sharedFileReferences(string(dockerfile)) {
		paths = append(paths, "shared/"+strings.TrimPrefix(ref, ".shared/"))
	}

	// shared build tooling affects every component
	paths = append(paths, "shared/Makefile.include.mk")

	slices.Sort(paths)
	return slices.Compact(paths), nil
}

func pathContains(path string, file string) bool {
	return file == path || strings.HasPrefix(file, path+"/")
}

// componentsForFiles returns the components affected by the changed files.
func componentsForFiles(repoRoot string, files []string) ([]string, error) {
	var affected []string
	for _, component := range getAllComponents(repoRoot) {
		paths, err := componentPaths(repoRoot, component)
		if err != nil {
			return nil, err
		}
		changed := slices.ContainsFunc(files, func(file string) bool {
			return slices.ContainsFunc(paths, func(path string) bool {
				return pathContains(path, file)
			})
		})
		if changed {
			affected = append(affected, component)
		}
	}
	return affected, nil
}

func getChangedComponents(repoRoot string) []string {
	ref := os.Getenv("GITHUB_REF")
	diffRange := "origin/main...HEAD"
	if ref == "refs/heads/main" || ref == "refs/heads/dev" {
		diffRange = "HEAD~1...HEAD"
	}

	output, err := exec.Command("git", "diff", "--name-only", diffRange).Output()
	if err != nil {
		output, _ = exec.Command("git", "ls-tree", "-r", "--name-only", "HEAD").Output()
	}

	components, err := componentsForFiles(repoRoot, strings.Fields(string(output)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error detecting components: %v\n", err)
		os.Exit(1)
	}
	return components
}

// currentComponentPaths returns the paths for the component in the current directory.
func currentComponentPaths() (string, []string, error) {
	component, err := deriveComponentName()
	if err != nil {
		return "", nil, err
	}
	paths, err := componentPaths("..", component)
	if err != nil {
		return "", nil, err
	}
	return component, paths, nil
}

// relativeToComponent converts repo-relative paths to paths relative to the
// component directory (the current directory).
func relativeToComponent(component string, paths []string) []string {
	relative := make([]string, len(paths))
	for i, path := range paths {
		if path == component {
			relative[i] = "."
		} else {
			relative[i] = "../" + path
		}
	}
	return relative
}
