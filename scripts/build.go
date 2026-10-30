package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// build generates code, compiles binaries and packages the helm chart into <component>/.build.
func build(component, version string) error {
	v, _ := parseSemver(version)
	out := filepath.Join(component, ".build")
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}

	platforms, err := componentPlatforms(component)
	if err != nil {
		return err
	}

	if hasKubebuilderMarkers(component) {
		if err := generateController(component); err != nil {
			return err
		}
	}

	if exists(component, "cmd", "main.go") {
		for _, platform := range platforms {
			goos, goarch, _ := strings.Cut(platform, "/")
			err := stream(
				[]string{"CGO_ENABLED=0", "GOOS=" + goos, "GOARCH=" + goarch},
				"go", "build",
				"-o", filepath.Join(out, fmt.Sprintf("%s_%s_%s", component, goos, goarch)),
				"-ldflags", fmt.Sprintf("-s -w -X %s/%s/internal.Version=%s", modulePath, component, version),
				"./"+component+"/cmd/main.go",
			)
			if err != nil {
				return err
			}
		}
	}

	if exists(component, "chart") {
		return stream(nil, "helm", "package", filepath.Join(component, "chart"),
			"--app-version", v.helmVersion(),
			"--version", v.helmVersion(),
			"--destination", out,
		)
	}
	return nil
}

func hasKubebuilderMarkers(component string) bool {
	found := false
	filepath.WalkDir(component, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || found || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		data, err := os.ReadFile(path)
		found = err == nil && strings.Contains(string(data), "+kubebuilder:")
		return nil
	})
	return found
}

// generateController writes deepcopy code next to the component's api types and
// crd/rbac manifests into its chart.
func generateController(component string) error {
	paths := "paths=./" + component + "/internal/..."
	generated := filepath.Join(component, "chart", "generated")
	if err := os.RemoveAll(generated); err != nil {
		return err
	}
	if err := os.MkdirAll(generated, 0o755); err != nil {
		return err
	}

	if err := stream(nil, "controller-gen", "object", paths); err != nil {
		return err
	}
	manifests := map[string]string{
		"crds.yaml": "crd",
		"rbac.yaml": "rbac:roleName=" + component,
	}
	for file, generator := range manifests {
		if err := streamToFile(filepath.Join(generated, file), "controller-gen", generator, paths, "output:stdout"); err != nil {
			return err
		}
	}
	return nil
}
