// Package secrets provides secure credential management for Sovereign VMs
// TEAM_011: Interactive password prompting and secrets file generation
package secrets

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// SecretsFile is the path to the secrets file (not committed to git)
const SecretsFile = ".secrets"

// Credentials holds database credentials
type Credentials struct {
	DBUser     string
	DBPassword string
}

// PromptPassword prompts the user for a password securely (no echo)
func PromptPassword(prompt string) (string, error) {
	fmt.Print(prompt)

	// Check if stdin is a terminal
	if term.IsTerminal(int(os.Stdin.Fd())) {
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() // newline after password entry
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		return string(password), nil
	}

	// Fallback for non-terminal (CI/scripts)
	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	return strings.TrimSpace(password), nil
}

// PromptCredentials interactively prompts for database credentials
func PromptCredentials(dbUser string) (*Credentials, error) {
	fmt.Println("\n=== PostgreSQL Credentials Setup ===")
	fmt.Printf("Database user: %s\n", dbUser)

	password, err := PromptPassword("Enter database password: ")
	if err != nil {
		return nil, err
	}

	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	confirm, err := PromptPassword("Confirm password: ")
	if err != nil {
		return nil, err
	}

	if password != confirm {
		return nil, fmt.Errorf("passwords do not match")
	}

	return &Credentials{
		DBUser:     dbUser,
		DBPassword: password,
	}, nil
}

// GeneratePassword generates a secure random password
func GeneratePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// WriteSecretsFile writes credentials to the secrets file
// This file should be in .gitignore
func WriteSecretsFile(creds *Credentials) error {
	content := fmt.Sprintf(`# Sovereign Vault Secrets - DO NOT COMMIT
# Generated during build - keep secure
DB_USER=%s
DB_PASSWORD=%s
`, creds.DBUser, creds.DBPassword)

	if err := os.WriteFile(SecretsFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write secrets file: %w", err)
	}

	fmt.Printf("  ✓ Secrets written to %s (mode 0600)\n", SecretsFile)
	return nil
}

// LoadSecretsFile loads credentials from the secrets file
func LoadSecretsFile() (*Credentials, error) {
	data, err := os.ReadFile(SecretsFile)
	if err != nil {
		return nil, fmt.Errorf("secrets file not found - run 'sovereign build --sql' first")
	}

	creds := &Credentials{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		switch key {
		case "DB_USER":
			creds.DBUser = value
		case "DB_PASSWORD":
			creds.DBPassword = value
		}
	}

	if creds.DBUser == "" || creds.DBPassword == "" {
		return nil, fmt.Errorf("incomplete credentials in secrets file")
	}

	return creds, nil
}

// SecretsExist checks if the secrets file exists
func SecretsExist() bool {
	_, err := os.Stat(SecretsFile)
	return err == nil
}
