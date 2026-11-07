package main

import (
	"encoding/json"
	"os"
)

// tag claims the releases from 'plan' (json on stdin) by pushing their git tags atomically.
// Fails if any tag already exists on a different commit.
func tag() error {
	var releases []plannedRelease
	if err := json.NewDecoder(os.Stdin).Decode(&releases); err != nil {
		return err
	}
	if len(releases) == 0 {
		return nil
	}
	args := []string{"push", "--atomic", "origin"}
	for _, r := range releases {
		args = append(args, "HEAD:refs/tags/"+r.Component+"/"+r.Version)
	}
	return stream(nil, "git", args...)
}
