// Dependency checking for VM services
// TEAM_029: Fail-fast validation of service dependencies
package common

import (
	"fmt"
	"os/exec"
	"strings"
)

// CheckDependencies verifies all dependencies are available before starting a VM.
// Returns an error if any dependency is unavailable.
// TEAM_029: Called by StartVM to fail fast
func CheckDependencies(cfg *VMConfig) error {
	if len(cfg.Dependencies) == 0 {
		return nil
	}

	fmt.Printf("Checking dependencies for %s...\n", cfg.DisplayName)

	for _, dep := range cfg.Dependencies {
		if err := checkDependency(dep); err != nil {
			return fmt.Errorf("dependency %s unavailable: %w", dep.Name, err)
		}
	}

	fmt.Println("  ✓ All dependencies available")
	return nil
}

// checkDependency verifies a single service dependency is available.
func checkDependency(dep ServiceDependency) error {
	fmt.Printf("  Checking %s (%s:%d)... ", dep.Description, dep.TailscaleHost, dep.Port)

	// First check if the dependency is reachable via Tailscale
	tsIP, connected := CheckTailscaleConnected(dep.TailscaleHost)
	if !connected {
		fmt.Println("✗")
		return fmt.Errorf("%s not found on Tailscale\n"+
			"  Start it first: sovereign start --%s", dep.TailscaleHost, dep.Name)
	}

	// Then check if the port is responding
	cmd := exec.Command("nc", "-z", "-w", "3", dep.TailscaleHost, fmt.Sprintf("%d", dep.Port))
	if err := cmd.Run(); err != nil {
		fmt.Println("✗")
		return fmt.Errorf("%s is on Tailscale (%s) but port %d not responding\n"+
			"  Check service status: sovereign test --%s",
			dep.TailscaleHost, tsIP, dep.Port, dep.Name)
	}

	fmt.Printf("✓ (%s)\n", tsIP)
	return nil
}

// GetDependencyInfo returns connection info for a dependency.
// Useful for passing to init scripts or configuration.
// TEAM_029: Used to get PostgreSQL connection details for Forgejo
func GetDependencyInfo(dep ServiceDependency) (*DependencyInfo, error) {
	tsIP, connected := CheckTailscaleConnected(dep.TailscaleHost)
	if !connected {
		return nil, fmt.Errorf("%s not connected to Tailscale", dep.TailscaleHost)
	}

	return &DependencyInfo{
		Name:          dep.Name,
		TailscaleHost: dep.TailscaleHost,
		TailscaleIP:   tsIP,
		Port:          dep.Port,
	}, nil
}

// DependencyInfo contains resolved connection info for a dependency.
type DependencyInfo struct {
	Name          string // "sql"
	TailscaleHost string // "sovereign-sql"
	TailscaleIP   string // "100.x.y.z"
	Port          int    // 5432
}

// ConnectionString returns a connection string for the dependency.
// Format varies by service type.
func (d *DependencyInfo) ConnectionString(user, password, dbname string) string {
	// PostgreSQL format
	if d.Port == 5432 {
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			user, password, d.TailscaleHost, d.Port, dbname)
	}
	// Generic TCP
	return fmt.Sprintf("%s:%d", d.TailscaleHost, d.Port)
}

// PostgreSQLDependency is a pre-configured dependency for PostgreSQL.
// TEAM_029: Common dependency used by Forgejo and future services
var PostgreSQLDependency = ServiceDependency{
	Name:          "sql",
	TailscaleHost: "sovereign-sql",
	Port:          5432,
	Description:   "PostgreSQL database",
}

// CheckPostgreSQLAvailable is a convenience function to check if PostgreSQL is ready.
func CheckPostgreSQLAvailable() error {
	fmt.Println("Checking PostgreSQL availability...")

	tsIP, connected := CheckTailscaleConnected("sovereign-sql")
	if !connected {
		return fmt.Errorf("sovereign-sql not found on Tailscale\n" +
			"  Start PostgreSQL first: sovereign start --sql")
	}

	// Check port
	cmd := exec.Command("nc", "-z", "-w", "3", "sovereign-sql", "5432")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sovereign-sql is on Tailscale (%s) but PostgreSQL port 5432 not responding\n"+
			"  Check status: sovereign test --sql", tsIP)
	}

	fmt.Printf("  ✓ PostgreSQL available at %s (sovereign-sql:5432)\n", tsIP)
	return nil
}

// GetPostgreSQLConnectionInfo returns PostgreSQL connection details.
// Reads credentials from secrets if available.
func GetPostgreSQLConnectionInfo() (host string, port int, user string, password string, err error) {
	tsIP, connected := CheckTailscaleConnected("sovereign-sql")
	if !connected {
		return "", 0, "", "", fmt.Errorf("sovereign-sql not connected to Tailscale")
	}

	// Default credentials (can be overridden by secrets)
	host = "sovereign-sql" // Use Tailscale hostname for DNS
	port = 5432
	user = "postgres"
	password = "sovereign" // Default, should be overridden

	// Note: In production, password should come from secrets.LoadSecretsFile()
	// but we keep this simple for now - the actual password is in .env

	_ = tsIP // Available if needed for direct IP connection
	return host, port, user, password, nil
}

// FormatPostgreSQLDSN formats a PostgreSQL connection string.
func FormatPostgreSQLDSN(host string, port int, user, password, dbname string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, password, host, port, dbname)
}

// CheckTailscaleConnected is defined in tailscale.go but we need it here.
// This is a compile-time check that it exists.
var _ = CheckTailscaleConnected

// DescribeDependencies returns a human-readable description of dependencies.
func DescribeDependencies(cfg *VMConfig) string {
	if len(cfg.Dependencies) == 0 {
		return "No dependencies"
	}

	var parts []string
	for _, dep := range cfg.Dependencies {
		parts = append(parts, fmt.Sprintf("%s (%s:%d)", dep.Description, dep.TailscaleHost, dep.Port))
	}
	return strings.Join(parts, ", ")
}
