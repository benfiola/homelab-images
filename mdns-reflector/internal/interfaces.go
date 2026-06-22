package internal

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func detectSourceInterface() (string, error) {
	// Try to find the interface with the default route
	iface, err := findDefaultRouteInterface()
	if err == nil {
		return iface, nil
	}

	// Fallback: look for eth*, veth*, or first non-loopback interface
	return findFallbackInterface()
}

// findDefaultRouteInterface reads /proc/net/route to find the interface with the default route
func findDefaultRouteInterface() (string, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return "", fmt.Errorf("failed to open /proc/net/route: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		// Skip header and incomplete lines
		if len(fields) < 2 {
			continue
		}

		// Look for default route (destination 00000000)
		if fields[1] == "00000000" {
			return fields[0], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("no default route found in /proc/net/route")
}

// findFallbackInterface looks for eth*, veth*, or the first non-loopback interface
func findFallbackInterface() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to list interfaces: %w", err)
	}

	// First pass: look for eth* or veth* interfaces
	for _, iface := range interfaces {
		if strings.HasPrefix(iface.Name, "eth") || strings.HasPrefix(iface.Name, "veth") {
			if iface.Flags&net.FlagLoopback == 0 {
				return iface.Name, nil
			}
		}
	}

	// Second pass: return first non-loopback interface
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 {
			return iface.Name, nil
		}
	}

	return "", fmt.Errorf("no suitable network interface found")
}

// parseInterfaces parses a comma-separated string of interface names
func parseInterfaces(s string) []string {
	if s == "" {
		return []string{}
	}

	var result []string
	for _, iface := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(iface); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// validateInterface checks if an interface exists and is up
func validateInterface(name string) error {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return fmt.Errorf("interface %s not found: %w", name, err)
	}

	if iface.Flags&net.FlagUp == 0 {
		return fmt.Errorf("interface %s is not up", name)
	}

	// Check if interface has an IP address
	addrs, err := iface.Addrs()
	if err != nil {
		return fmt.Errorf("failed to get addresses for interface %s: %w", name, err)
	}

	if len(addrs) == 0 {
		return fmt.Errorf("interface %s has no IP addresses assigned", name)
	}

	return nil
}
