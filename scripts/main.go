package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
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

// imageTagVersion converts a semver string to a valid OCI image tag by replacing
// the '+' build-metadata separator (not allowed in OCI tags) with '-'.
func imageTagVersion(version string) (string, error) {
	p := parseSemver(version)
	if p == nil {
		return "", fmt.Errorf("version %q does not match semver", version)
	}
	v := fmt.Sprintf("v%s.%s.%s", p["major"], p["minor"], p["patch"])
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
	case "build-helm":
		buildHelm()
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

func buildHelm() {
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

	fmt.Printf("Building Helm chart for %s (version: %s)...\n", component, version)

	cmd := exec.Command("helm", "package", "chart",
		"--app-version", version,
		"--version", version,
		"--destination", outputDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error building helm chart: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Built Helm chart in %s\n", outputDir)
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

	cmd := exec.Command("helm", "package", "chart",
		"--app-version", version,
		"--version", version,
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

func packageDocker() {
	// Check if Dockerfile exists first
	if _, err := os.Stat("Dockerfile"); err != nil {
		fmt.Println("No Dockerfile found, skipping Docker build")
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
	component, err := deriveComponentName()
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

	// Build svu command flags
	svuArgs := []string{
		"next",
		fmt.Sprintf("--tag.prefix=%s/v", component),
		fmt.Sprintf("--tag.pattern=%s/v*", component),
		"--tag.output=v",
		"--always=true",
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

func detectComponents() {
	repoRoot := os.Getenv("REPO_ROOT")
	if repoRoot == "" {
		repoRoot = "."
	}

	buildAll := os.Getenv("BUILD_ALL") == "true"
	manualComponents := os.Getenv("MANUAL_COMPONENTS")
	// GITHUB_REF is optional - only used in CI/CD
	_ = os.Getenv("GITHUB_REF")

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
		// Detect changed components
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

func getChangedComponents(repoRoot string) []string {
	validComponents := getAllComponents(repoRoot)
	validSet := make(map[string]bool)
	for _, c := range validComponents {
		validSet[c] = true
	}

	// Run git diff to get changed files
	var cmd *exec.Cmd
	ref := os.Getenv("GITHUB_REF")

	if ref == "refs/heads/main" || ref == "refs/heads/dev" {
		cmd = exec.Command("git", "diff", "--name-only", "HEAD~1...HEAD")
	} else {
		cmd = exec.Command("git", "diff", "--name-only", "origin/main...HEAD")
	}

	output, err := cmd.Output()
	if err != nil {
		output, _ = exec.Command("git", "ls-tree", "-r", "--name-only", "HEAD").Output()
	}

	var components []string
	changedFiles := strings.Split(string(output), "\n")
	seen := make(map[string]bool)

	for _, file := range changedFiles {
		if file == "" {
			continue
		}
		parts := strings.Split(file, "/")
		component := parts[0]

		if validSet[component] && !seen[component] {
			components = append(components, component)
			seen[component] = true
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
	// Try to read go.mod
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	// Parse module line
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if mod, ok := strings.CutPrefix(line, "module "); ok {
			return mod, nil
		}
	}

	return "", fmt.Errorf("module directive not found in go.mod")
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

	chartFile := filepath.Join(".build", fmt.Sprintf("%s-%s.tgz", component, version))

	// Check if chart file was built
	if _, err := os.Stat(chartFile); err != nil {
		fmt.Fprintf(os.Stderr, "error: chart not found at %s\n", chartFile)
		os.Exit(1)
	}

	imageBase := DefaultImageBase
	ociRef := fmt.Sprintf("oci://%s/charts", imageBase)
	chartRef := fmt.Sprintf("%s/%s:%s", ociRef, component, version)

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

	// Generate changelog
	var changelogCmd *exec.Cmd
	if previousTag != "" {
		changelogCmd = exec.Command("git", "log", fmt.Sprintf("%s...HEAD", previousTag), "--oneline")
	} else {
		changelogCmd = exec.Command("git", "log", "--oneline")
	}

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
		helmPull := fmt.Sprintf("helm pull oci://%s/charts/%s --version %s", DefaultImageBase, component, version)
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
