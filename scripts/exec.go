package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// output runs a command and returns its trimmed stdout.
func output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// stream runs a command with its output attached to ours.
func stream(env []string, name string, args ...string) error {
	fmt.Printf("+ %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), env...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

// streamToFile runs a command and writes its stdout to a file.
func streamToFile(path, name string, args ...string) error {
	fmt.Printf("+ %s %s > %s\n", name, strings.Join(args, " "), path)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	cmd := exec.Command(name, args...)
	cmd.Stdout = file
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
