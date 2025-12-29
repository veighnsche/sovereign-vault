// Tailscale registration management for VMs
// TEAM_029: Extracted from sql/verify.go and forge/verify.go
package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// RemoveTailscaleRegistrations removes existing Tailscale registrations
// for the given hostname prefix (e.g., "sovereign-sql", "sovereign-forge").
// TEAM_029: Extracted from sql/verify.go RemoveTailscaleRegistrations
func RemoveTailscaleRegistrations(hostnamePrefix string) error {
	fmt.Println("Checking for existing Tailscale registrations...")

	out, err := exec.Command("tailscale", "status", "--json").Output()
	if err != nil {
		fmt.Println("  ⚠ Cannot check Tailscale (CLI not available)")
		return nil
	}

	var status struct {
		Peer map[string]struct {
			HostName string `json:"HostName"`
			ID       string `json:"ID"`
			Online   bool   `json:"Online"`
		} `json:"Peer"`
	}

	if err := json.Unmarshal(out, &status); err != nil {
		fmt.Printf("  ⚠ Cannot parse Tailscale status: %v\n", err)
		return nil
	}

	var toDelete []struct {
		ID   string
		Name string
	}
	for _, peer := range status.Peer {
		if strings.HasPrefix(peer.HostName, hostnamePrefix) {
			toDelete = append(toDelete, struct {
				ID   string
				Name string
			}{ID: peer.ID, Name: peer.HostName})
		}
	}

	if len(toDelete) == 0 {
		fmt.Printf("  ✓ No existing %s registrations found\n", hostnamePrefix)
		return nil
	}

	fmt.Printf("  Found %d %s registration(s) to delete\n", len(toDelete), hostnamePrefix)

	apiKey := getAPIKey()
	if apiKey == "" {
		fmt.Println("  ⚠ TAILSCALE_API_KEY not set - cannot auto-delete")
		fmt.Println("  Please delete manually at: https://login.tailscale.com/admin/machines")
		for _, d := range toDelete {
			fmt.Printf("    - %s (ID: %s)\n", d.Name, d.ID)
		}
		return fmt.Errorf("found %d existing registrations - delete manually or set TAILSCALE_API_KEY", len(toDelete))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	var deleted int
	for _, d := range toDelete {
		url := fmt.Sprintf("https://api.tailscale.com/api/v2/device/%s", d.ID)
		req, _ := http.NewRequest("DELETE", url, nil)
		req.SetBasicAuth(apiKey, "")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  ⚠ Failed to delete %s: %v\n", d.Name, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == 200 || resp.StatusCode == 204 {
			fmt.Printf("  ✓ Deleted %s\n", d.Name)
			deleted++
		} else {
			fmt.Printf("  ⚠ Failed to delete %s: HTTP %d\n", d.Name, resp.StatusCode)
		}
	}

	if deleted == len(toDelete) {
		fmt.Printf("  ✓ Successfully deleted all %d registration(s)\n", deleted)
	} else {
		fmt.Printf("  ⚠ Deleted %d of %d registration(s)\n", deleted, len(toDelete))
	}

	return nil
}

// getAPIKey retrieves the Tailscale API key from environment or .env files.
// TEAM_029: Extracted from sql/verify.go
func getAPIKey() string {
	apiKey := os.Getenv("TAILSCALE_API_KEY")
	if apiKey != "" {
		return apiKey
	}

	envPaths := []string{
		".env",
		"sovereign/.env",
		"../sovereign/.env",
		os.Getenv("HOME") + "/Projects/android/kernel/sovereign/.env",
	}
	for _, envPath := range envPaths {
		if envData, err := os.ReadFile(envPath); err == nil {
			for _, line := range strings.Split(string(envData), "\n") {
				if strings.HasPrefix(line, "TAILSCALE_API_KEY=") {
					apiKey = strings.TrimPrefix(line, "TAILSCALE_API_KEY=")
					apiKey = strings.Trim(apiKey, "\"'")
					if apiKey != "" {
						fmt.Printf("  Found API key in %s\n", envPath)
						return apiKey
					}
				}
			}
		}
	}

	return ""
}

// CheckTailscaleConnected checks if a machine is connected via Tailscale.
// Returns the Tailscale IP if connected, empty string otherwise.
// TEAM_029: Extracted from sql/verify.go Test()
func CheckTailscaleConnected(hostnamePrefix string) (string, bool) {
	out, err := exec.Command("tailscale", "status").Output()
	if err != nil {
		return "", false
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, hostnamePrefix) && !strings.Contains(line, "offline") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[0], true
			}
		}
	}
	return "", false
}
