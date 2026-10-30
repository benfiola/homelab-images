package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// publish pushes the component's container image and helm chart (see build).
func publish(component, version string) error {
	v, _ := parseSemver(version)

	platforms, err := componentPlatforms(component)
	if err != nil {
		return err
	}

	err = stream(nil, "docker", "buildx", "build",
		"--platform", strings.Join(platforms, ","),
		"--push",
		"--tag", fmt.Sprintf("%s/%s:%s", imageBase, component, v.imageTag()),
		component,
	)
	if err != nil {
		return err
	}

	if exists(component, "chart") {
		chart := filepath.Join(component, ".build", fmt.Sprintf("%s-%s.tgz", component, v.helmVersion()))
		return stream(nil, "helm", "push", chart, "oci://"+imageBase+"/charts")
	}
	return nil
}
