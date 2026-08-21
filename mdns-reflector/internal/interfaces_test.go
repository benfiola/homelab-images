package internal

import (
	"testing"
)

func TestParseInterfaces_Empty(t *testing.T) {
	result := parseInterfaces("")
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %v", result)
	}
}

func TestParseInterfaces_Single(t *testing.T) {
	result := parseInterfaces("eth0")
	if len(result) != 1 || result[0] != "eth0" {
		t.Fatalf("expected [eth0], got %v", result)
	}
}

func TestParseInterfaces_Multiple(t *testing.T) {
	result := parseInterfaces("eth0,eth1,eth2")
	if len(result) != 3 || result[0] != "eth0" || result[1] != "eth1" || result[2] != "eth2" {
		t.Fatalf("expected [eth0 eth1 eth2], got %v", result)
	}
}

func TestParseInterfaces_Whitespace(t *testing.T) {
	result := parseInterfaces("eth0, eth1 , eth2")
	if len(result) != 3 || result[0] != "eth0" || result[1] != "eth1" || result[2] != "eth2" {
		t.Fatalf("expected whitespace trimmed to [eth0 eth1 eth2], got %v", result)
	}
}

func TestParseInterfaces_EmptySegments(t *testing.T) {
	result := parseInterfaces(",eth0,,eth1,")
	if len(result) != 2 || result[0] != "eth0" || result[1] != "eth1" {
		t.Fatalf("expected empty segments ignored, got %v", result)
	}
}

func TestDetectInterfaces_ReturnsNonLoopback(t *testing.T) {
	ifaces, err := detectInterfaces()
	if err != nil {
		t.Skipf("no suitable interfaces in test environment: %v", err)
	}
	for _, name := range ifaces {
		if name == "lo" {
			t.Fatalf("loopback interface should not be included, got %v", ifaces)
		}
	}
}

func TestDetectInterfaces_ReturnsAtLeastOne(t *testing.T) {
	ifaces, err := detectInterfaces()
	if err != nil {
		t.Skipf("no suitable interfaces in test environment: %v", err)
	}
	if len(ifaces) == 0 {
		t.Fatal("expected at least one interface")
	}
}
