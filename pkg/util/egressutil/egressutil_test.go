package egressutil

import (
	"strings"
	"testing"
)

func TestUpdateHostnames_ContainsGitHub(t *testing.T) {
	hosts := map[string]bool{}
	for _, h := range UpdateHostnames {
		hosts[h] = true
	}
	for _, required := range []string{
		"api.github.com",
		"github.com",
		"objects.githubusercontent.com",
	} {
		if !hosts[required] {
			t.Errorf("UpdateHostnames missing required host: %s", required)
		}
	}
}

func TestUpdateEgressRules_DNSIncluded(t *testing.T) {
	rules := UpdateEgressRules()
	var hasDNSUDP, hasDNSTCP bool
	for _, r := range rules {
		if r.Port == "53" && r.Proto == "udp" {
			hasDNSUDP = true
		}
		if r.Port == "53" && r.Proto == "tcp" {
			hasDNSTCP = true
		}
	}
	if !hasDNSUDP {
		t.Error("missing DNS UDP egress rule")
	}
	if !hasDNSTCP {
		t.Error("missing DNS TCP egress rule")
	}
}

func TestUpdateEgressRules_HTTPSForAllHosts(t *testing.T) {
	rules := UpdateEgressRules()
	hostsCovered := map[string]bool{}
	for _, r := range rules {
		if r.Port == "443" && r.Proto == "tcp" {
			hostsCovered[r.Host] = true
		}
	}
	for _, h := range UpdateHostnames {
		if !hostsCovered[h] {
			t.Errorf("no HTTPS egress rule for update host: %s", h)
		}
	}
}

func TestUpdateEgressRules_CommentsSet(t *testing.T) {
	for _, r := range UpdateEgressRules() {
		if r.Comment == "" {
			t.Errorf("rule for host=%q port=%s has empty comment", r.Host, r.Port)
		}
	}
}

func TestUFWCommands_AnyHost(t *testing.T) {
	rules := []EgressRule{
		{Host: "any", Port: "53", Proto: "udp", Comment: "DNS"},
	}
	cmds := UFWCommands(rules)
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	joined := strings.Join(cmds[0], " ")
	// Should not contain "to any" — should use "ufw allow out 53/udp" form.
	if strings.Contains(joined, "to any") {
		t.Errorf("DNS rule should not use 'to any' form: %s", joined)
	}
	if !strings.Contains(joined, "53/udp") {
		t.Errorf("expected '53/udp' in command: %s", joined)
	}
}

func TestUFWCommands_SpecificHost(t *testing.T) {
	rules := []EgressRule{
		{Host: "api.github.com", Port: "443", Proto: "tcp", Comment: "GitHub API"},
	}
	cmds := UFWCommands(rules)
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	joined := strings.Join(cmds[0], " ")
	if !strings.Contains(joined, "api.github.com") {
		t.Errorf("expected host in command: %s", joined)
	}
	if !strings.Contains(joined, "443") {
		t.Errorf("expected port 443 in command: %s", joined)
	}
	if !strings.Contains(joined, "proto tcp") {
		t.Errorf("expected 'proto tcp' in command: %s", joined)
	}
}

func TestUFWCommands_CountMatchesRules(t *testing.T) {
	rules := UpdateEgressRules()
	cmds := UFWCommands(rules)
	if len(cmds) != len(rules) {
		t.Errorf("UFWCommands returned %d commands for %d rules", len(cmds), len(rules))
	}
}

func TestDetectBackend_ReturnsValid(t *testing.T) {
	backend := DetectBackend()
	switch backend {
	case BackendUFW, BackendNone:
		// valid
	default:
		t.Errorf("DetectBackend returned unexpected value: %q", backend)
	}
}
