package internal

import (
	"fmt"
)

type Opts struct {
	Interfaces string
}

type MDNSReflector struct {
	Interfaces []string
}

func New(opts *Opts) (*MDNSReflector, error) {
	interfaces := parseInterfaces(opts.Interfaces)

	if len(interfaces) == 0 {
		detected, err := detectInterfaces()
		if err != nil {
			return nil, fmt.Errorf("no interfaces provided and auto-detection failed: %w", err)
		}
		interfaces = detected
	}

	if len(interfaces) < 2 {
		return nil, fmt.Errorf("at least two interfaces are required for reflection")
	}

	for _, iface := range interfaces {
		if err := validateInterface(iface); err != nil {
			return nil, fmt.Errorf("invalid interface: %w", err)
		}
	}

	return &MDNSReflector{
		Interfaces: interfaces,
	}, nil
}
