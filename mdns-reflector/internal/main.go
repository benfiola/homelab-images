package internal

import (
	"fmt"
)

type Opts struct {
	SourceInterfaces string
	DestInterfaces   string
}

type MDNSReflector struct {
	SourceInterfaces []string
	DestInterfaces   []string
}

func New(opts *Opts) (*MDNSReflector, error) {
	// Parse destination interfaces (required)
	destInterfaces := parseInterfaces(opts.DestInterfaces)
	if len(destInterfaces) == 0 {
		return nil, fmt.Errorf("at least one destination interface must be specified")
	}

	// Parse source interfaces (optional)
	sourceInterfaces := parseInterfaces(opts.SourceInterfaces)

	// If no source interfaces specified, try to auto-detect
	if len(sourceInterfaces) == 0 {
		detected, err := detectSourceInterface()
		if err != nil {
			return nil, fmt.Errorf("no source interfaces provided and auto-detection failed: %w", err)
		}
		sourceInterfaces = []string{detected}
	}

	// Validate all interfaces exist and are up
	for _, iface := range sourceInterfaces {
		if err := validateInterface(iface); err != nil {
			return nil, fmt.Errorf("invalid source interface: %w", err)
		}
	}

	for _, iface := range destInterfaces {
		if err := validateInterface(iface); err != nil {
			return nil, fmt.Errorf("invalid destination interface: %w", err)
		}
	}

	return &MDNSReflector{
		SourceInterfaces: sourceInterfaces,
		DestInterfaces:   destInterfaces,
	}, nil
}
