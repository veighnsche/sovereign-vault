// Vault VM verification (Test command and Tailscale checks)
// TEAM_034: Based on forge/verify.go pattern
package vault

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/anthropics/sovereign/internal/vm/common"
)

// Test delegates to common.RunVMTests with Vault-specific custom tests
func (v *VM) Test() error {
	return common.RunVMTests(VaultConfig, vaultCustomTests)
}

// Vault-specific tests
// TEAM_035: Use dynamic FQDN detection for HTTPS tests
var vaultCustomTests = []common.TestFunc{
	testVaultwardenWebUI,
	testVaultwardenAPI,
}

func testVaultwardenWebUI(cfg *common.VMConfig) common.TestResult {
	// TEAM_035: Get actual FQDN from tailscale status (may be sovereign-vault-1, -2, etc.)
	fqdn := common.GetTailscaleFQDN(cfg)
	if fqdn == "" {
		return common.TestResult{Name: "Vaultwarden web UI (via Tailscale)", Passed: false, Message: "cannot determine Tailscale FQDN"}
	}
	cmd := exec.Command("curl", "-sk", "-o", "/dev/null", "-w", "%{http_code}",
		"--connect-timeout", "5", fmt.Sprintf("https://%s", fqdn))
	output, _ := cmd.Output()
	httpCode := strings.TrimSpace(string(output))
	if httpCode == "200" {
		return common.TestResult{Name: "Vaultwarden web UI (via Tailscale)", Passed: true, Message: fqdn}
	}
	return common.TestResult{Name: "Vaultwarden web UI (via Tailscale)", Passed: false, Message: fmt.Sprintf("HTTP %s from %s", httpCode, fqdn)}
}

func testVaultwardenAPI(cfg *common.VMConfig) common.TestResult {
	// TEAM_035: Get actual FQDN from tailscale status
	fqdn := common.GetTailscaleFQDN(cfg)
	if fqdn == "" {
		return common.TestResult{Name: "Vaultwarden API (via Tailscale)", Passed: false, Message: "cannot determine Tailscale FQDN"}
	}
	cmd := exec.Command("curl", "-sk", "-o", "/dev/null", "-w", "%{http_code}",
		"--connect-timeout", "5", fmt.Sprintf("https://%s/api/config", fqdn))
	output, _ := cmd.Output()
	httpCode := strings.TrimSpace(string(output))
	if httpCode == "200" {
		return common.TestResult{Name: "Vaultwarden API (via Tailscale)", Passed: true}
	}
	return common.TestResult{Name: "Vaultwarden API (via Tailscale)", Passed: false, Message: fmt.Sprintf("HTTP %s", httpCode)}
}

// RemoveTailscaleRegistrations delegates to common package
func RemoveTailscaleRegistrations() error {
	return common.RemoveTailscaleRegistrations("sovereign-vault")
}
