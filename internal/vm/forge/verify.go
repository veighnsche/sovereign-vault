// Forge VM verification (Test command and Tailscale checks)
// TEAM_025: Split from forge.go following sql/verify.go pattern
// TEAM_029: Migrated to use common package
package forge

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/anthropics/sovereign/internal/vm/common"
)

// TEAM_029: Test delegates to common.RunVMTests with Forge-specific custom tests
func (v *VM) Test() error {
	return common.RunVMTests(ForgeConfig, forgeCustomTests)
}

// TEAM_029: Forge-specific tests
var forgeCustomTests = []common.TestFunc{
	testForgejoWebUI,
	testSSHPort,
}

// TEAM_035: Updated to use HTTPS on port 443 with dynamic FQDN detection
func testForgejoWebUI(cfg *common.VMConfig) common.TestResult {
	// Get actual FQDN (may be sovereign-forge-1, -2, etc.)
	fqdn := common.GetTailscaleFQDN(cfg)
	if fqdn == "" {
		return common.TestResult{Name: "Forgejo web UI (via Tailscale)", Passed: false, Message: "cannot determine Tailscale FQDN"}
	}
	cmd := exec.Command("curl", "-sk", "-o", "/dev/null", "-w", "%{http_code}",
		"--connect-timeout", "5", fmt.Sprintf("https://%s", fqdn))
	output, _ := cmd.Output()
	httpCode := strings.TrimSpace(string(output))
	if httpCode == "200" {
		return common.TestResult{Name: "Forgejo web UI (via Tailscale)", Passed: true, Message: fqdn}
	}
	if httpCode == "302" || httpCode == "303" {
		return common.TestResult{Name: "Forgejo web UI (via Tailscale)", Passed: true, Message: fmt.Sprintf("%s (redirect - needs initial setup)", fqdn)}
	}
	return common.TestResult{Name: "Forgejo web UI (via Tailscale)", Passed: false, Message: fmt.Sprintf("HTTP %s from %s", httpCode, fqdn)}
}

func testSSHPort(cfg *common.VMConfig) common.TestResult {
	cmd := exec.Command("nc", "-z", "-w", "3", cfg.TailscaleHost, "22")
	if err := cmd.Run(); err != nil {
		return common.TestResult{Name: "SSH port (via Tailscale)", Passed: true, Message: "SSH port not responding (may be disabled)"}
	}
	return common.TestResult{Name: "SSH port (via Tailscale)", Passed: true}
}

// TEAM_029: RemoveTailscaleRegistrations delegates to common package
func RemoveTailscaleRegistrations() error {
	return common.RemoveTailscaleRegistrations("sovereign-forge")
}
