package main

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// listComponents returns every top-level directory that has a Dockerfile.
func listComponents() ([]string, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var components []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(entry.Name(), "Dockerfile")); err == nil {
			components = append(components, entry.Name())
		}
	}
	return components, nil
}

func isComponent(name string) bool {
	_, err := os.Stat(filepath.Join(name, "Dockerfile"))
	return err == nil
}

func exists(parts ...string) bool {
	_, err := os.Stat(filepath.Join(parts...))
	return err == nil
}

// dependencies describes what a component is built from. A component is
// affected by changes to its own directory, to the repo packages it imports
// and to the versions of the external modules it imports - and nothing else.
type dependencies struct {
	component string
	// repo directories of imported packages (excluding the component itself)
	packages []string
	// external go modules of imported packages
	modules []string
}

func listDependencies(component string) (*dependencies, error) {
	deps := &dependencies{component: component}
	if !exists(component, "cmd", "main.go") {
		return deps, nil
	}

	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	out, err := output("go", "list", "-deps", "-f", "{{if not .Standard}}{{.Dir}} {{with .Module}}{{.Path}}{{end}}{{end}}", "./"+component+"/...")
	if err != nil {
		return nil, err
	}
	for line := range strings.SplitSeq(out, "\n") {
		dir, module, _ := strings.Cut(line, " ")
		if dir == "" {
			continue
		}
		if module != modulePath {
			deps.modules = append(deps.modules, module)
			continue
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		if rel != component && !strings.HasPrefix(rel, component+"/") {
			deps.packages = append(deps.packages, rel)
		}
	}
	slices.Sort(deps.packages)
	slices.Sort(deps.modules)
	deps.packages = slices.Compact(deps.packages)
	deps.modules = slices.Compact(deps.modules)
	return deps, nil
}

// pathspecs returns git pathspecs matching the files the component is built
// from. Docs and tests never change what is shipped, so they are excluded.
// go.mod and go.sum are only included when a module the component imports
// changed since base.
func (d *dependencies) pathspecs(base string) ([]string, error) {
	specs := []string{
		":(top)" + d.component,
		":(top,exclude,glob)**/*.md",
		":(top,exclude,glob)**/*_test.go",
	}
	for _, dir := range d.packages {
		specs = append(specs, ":(top,glob)"+dir+"/*")
	}
	if len(d.modules) > 0 {
		changed, err := d.modulesChanged(base)
		if err != nil {
			return nil, err
		}
		if changed {
			specs = append(specs, ":(top)go.mod", ":(top)go.sum")
		}
	}
	return specs, nil
}

// modulesChanged reports whether go.mod changed since base in a way that
// touches the toolchain or a module this component imports.
func (d *dependencies) modulesChanged(base string) (bool, error) {
	if base == "" {
		return true, nil
	}
	diff, err := output("git", "diff", "-U0", base, "HEAD", "--", "go.mod")
	if err != nil {
		return false, err
	}
	for line := range strings.SplitSeq(diff, "\n") {
		if len(line) < 2 || (line[0] != '+' && line[0] != '-') || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line[1:]), "require "))
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "go" || fields[0] == "toolchain" || slices.Contains(d.modules, path.Clean(fields[0])) {
			return true, nil
		}
	}
	return false, nil
}

var platformsRegex = regexp.MustCompile(`(?m)^#\s*platforms:\s*(\S+)\s*$`)

// componentPlatforms returns the platforms a component is built for: the
// comma-separated list from a '# platforms: linux/amd64,linux/arm64' line in
// its Dockerfile, or defaultPlatforms.
func componentPlatforms(component string) ([]string, error) {
	dockerfile, err := os.ReadFile(filepath.Join(component, "Dockerfile"))
	if err != nil {
		return nil, err
	}
	value := defaultPlatforms
	if match := platformsRegex.FindSubmatch(dockerfile); match != nil {
		value = string(match[1])
	}
	return strings.Split(value, ","), nil
}
