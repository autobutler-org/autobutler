// Package egressutil manages outbound firewall allowlist rules for AutoButler.
//
// AutoButler is designed for airgap-friendly operation — by default no outbound
// traffic is required except to reach the software update servers. This package
// exposes the canonical set of update-egress hostnames and helper functions for
// generating and applying ufw rules that implement default-deny egress with an
// explicit update-server allowlist.
//
// Rule strategy (ufw):
//  1. Allow established/related connections (responses to local requests).
//  2. Allow outbound HTTPS (443) to each update hostname by DNS name.
//  3. Allow outbound DNS (53 UDP/TCP) to resolve those names.
//  4. Deny all other outbound traffic (default-deny egress).
//
// These rules are idempotent — running Apply() twice produces the same result.
package egressutil

import (
	"fmt"
	"os/exec"
	"strings"
)

// UpdateHostnames is the canonical set of FQDNs that AutoButler contacts for
// software update checks and downloads. Adding a host here automatically
// includes it in the generated firewall rules.
var UpdateHostnames = []string{
	// GitHub Releases API — release metadata, release listings.
	"api.github.com",
	// GitHub — release asset redirect host.
	"github.com",
	// GitHub CDN — actual binary downloads (resolved from GitHub redirect).
	"objects.githubusercontent.com",
	// GitHub CDN — alternate asset host.
	"codeload.github.com",
	// Azure Blob Storage — autobutlerrelease storage account.
	"autobutlerrelease.blob.core.windows.net",
}

// EgressRule represents a single outbound firewall allowlist rule.
type EgressRule struct {
	// Host is the destination hostname or CIDR.
	Host string
	// Port is the destination port (e.g. "443" or "53").
	Port string
	// Proto is the protocol: "tcp", "udp", or "any".
	Proto string
	// Comment describes the purpose of the rule.
	Comment string
}

// UpdateEgressRules returns the minimal set of outbound firewall rules needed
// for AutoButler to check for and download software updates.
func UpdateEgressRules() []EgressRule {
	rules := make([]EgressRule, 0, len(UpdateHostnames)+2)

	// DNS resolution for all update hosts.
	rules = append(rules,
		EgressRule{Host: "any", Port: "53", Proto: "udp", Comment: "DNS resolution (UDP)"},
		EgressRule{Host: "any", Port: "53", Proto: "tcp", Comment: "DNS resolution (TCP)"},
	)

	for _, host := range UpdateHostnames {
		rules = append(rules, EgressRule{
			Host:    host,
			Port:    "443",
			Proto:   "tcp",
			Comment: fmt.Sprintf("AutoButler update egress: %s", host),
		})
	}

	return rules
}

// FirewallBackend identifies the firewall management tool available on the system.
type FirewallBackend string

const (
	BackendUFW  FirewallBackend = "ufw"
	BackendNone FirewallBackend = "none"
)

// DetectBackend returns the firewall backend available on this system.
func DetectBackend() FirewallBackend {
	if _, err := exec.LookPath("ufw"); err == nil {
		return BackendUFW
	}
	return BackendNone
}

// UFWCommands returns the ufw command-line invocations that would apply the
// given rules. It does NOT execute them — callers decide whether to run.
func UFWCommands(rules []EgressRule) [][]string {
	cmds := make([][]string, 0, len(rules))
	for _, r := range rules {
		if r.Host == "any" {
			// ufw allow out <port>/<proto>
			cmds = append(cmds, []string{
				"ufw", "allow", "out", fmt.Sprintf("%s/%s", r.Port, r.Proto),
				"comment", r.Comment,
			})
		} else {
			// ufw allow out proto <proto> to <host> port <port>
			cmds = append(cmds, []string{
				"ufw", "allow", "out",
				"proto", r.Proto,
				"to", r.Host,
				"port", r.Port,
				"comment", r.Comment,
			})
		}
	}
	return cmds
}

// ApplyUpdateEgressRules applies the update-egress allowlist rules via ufw.
// Requires root privileges (or sudo). Returns an error if ufw is not available
// or any rule fails to apply.
//
// Rules are ufw-idempotent: inserting an existing rule is a no-op ("Skipping
// adding existing rule").
func ApplyUpdateEgressRules() error {
	backend := DetectBackend()
	if backend == BackendNone {
		return fmt.Errorf("no supported firewall backend found (ufw not installed)")
	}

	rules := UpdateEgressRules()
	cmds := UFWCommands(rules)

	var errs []string
	for _, args := range cmds {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput() //nolint:gosec
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v — %s", strings.Join(args, " "), err, strings.TrimSpace(string(out))))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("applying egress rules: %s", strings.Join(errs, "; "))
	}
	return nil
}

// StatusEntry describes one ufw rule relevant to update egress.
type StatusEntry struct {
	Rule    string
	Applied bool
}

// QueryUpdateEgressStatus returns whether each update-egress rule appears to be
// present in the current ufw ruleset. It shells out to `ufw status` and parses
// the output heuristically — treat this as informational, not authoritative.
func QueryUpdateEgressStatus() ([]StatusEntry, error) {
	out, err := exec.Command("ufw", "status").CombinedOutput() //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("ufw status: %w — %s", err, strings.TrimSpace(string(out)))
	}

	status := string(out)
	rules := UpdateEgressRules()
	entries := make([]StatusEntry, 0, len(rules))

	for _, r := range rules {
		// Heuristic: check if the host (or port) appears in the output.
		applied := false
		if r.Host == "any" {
			applied = strings.Contains(status, r.Port+"/"+r.Proto)
		} else {
			applied = strings.Contains(status, r.Host)
		}
		entries = append(entries, StatusEntry{
			Rule:    fmt.Sprintf("out %s/%s → %s:%s", r.Proto, r.Proto, r.Host, r.Port),
			Applied: applied,
		})
	}
	return entries, nil
}
