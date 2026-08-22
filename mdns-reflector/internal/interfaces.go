package internal

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func isHardwareInterface(name string) bool {
	_, err := os.Lstat(fmt.Sprintf("/sys/class/net/%s/device", name))
	return err == nil
}

func detectInterfaces() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	var result []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		if !isHardwareInterface(iface.Name) && iface.Name != "mdns0" {
			continue
		}
		result = append(result, iface.Name)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no suitable network interfaces found")
	}

	return result, nil
}

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

func validateInterface(name string) error {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return fmt.Errorf("interface %s not found: %w", name, err)
	}

	if iface.Flags&net.FlagUp == 0 {
		return fmt.Errorf("interface %s is not up", name)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return fmt.Errorf("failed to get addresses for interface %s: %w", name, err)
	}

	if len(addrs) == 0 {
		return fmt.Errorf("interface %s has no IP addresses assigned", name)
	}

	return nil
}
