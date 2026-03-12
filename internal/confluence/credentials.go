package confluence

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Credentials holds persisted authentication data for Confluence Cloud.
type Credentials struct {
	Domain    string    `json:"domain"`     // e.g., "mycompany" (for mycompany.atlassian.net)
	Email     string    `json:"email"`      // Atlassian account email
	APIToken  string    `json:"api_token"`  // Atlassian API token
	CreatedAt time.Time `json:"created_at"` // When credentials were stored
}

// CredentialsPath returns the path to the credentials file.
func CredentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}
	return filepath.Join(home, ".config", "ctk", "credentials.json"), nil
}

// SaveCredentials persists credentials to disk.
func SaveCredentials(creds *Credentials) error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing credentials file: %w", err)
	}

	return nil
}

// LoadCredentials reads stored credentials from disk.
func LoadCredentials() (*Credentials, error) {
	path, err := CredentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading credentials file: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}

	return &creds, nil
}

// RemoveCredentials deletes the stored credentials file.
func RemoveCredentials() error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// LoadCredentialsFromEnv attempts to load credentials from environment variables.
// Returns nil if the env vars are not set.
func LoadCredentialsFromEnv() *Credentials {
	domain := os.Getenv("CONFLUENCE_DOMAIN")
	email := os.Getenv("CONFLUENCE_EMAIL")
	apiToken := os.Getenv("CONFLUENCE_API_TOKEN")

	if domain == "" || email == "" || apiToken == "" {
		return nil
	}

	return &Credentials{
		Domain:   domain,
		Email:    email,
		APIToken: apiToken,
	}
}

// ScopeReadOnly contains the granular scopes required for read-only access.
var ScopeReadOnly = []string{
	"read:space:confluence",
	"read:page:confluence",
	"read:folder:confluence",
	"read:hierarchical-content:confluence",
	"read:comment:confluence",
	"read:label:confluence",
	"read:attachment:confluence",
	"search:confluence",
}

// ScopeWrite contains the additional granular scopes for write operations.
var ScopeWrite = []string{
	"write:page:confluence",
	"write:folder:confluence",
	"write:comment:confluence",
	"write:label:confluence",
	"delete:page:confluence",
	"delete:folder:confluence",
}

// InteractiveLogin prompts the user for credentials and stores them.
func InteractiveLogin() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("Confluence Cloud Authentication")
	fmt.Println("===============================")
	fmt.Println()
	fmt.Println("Create an API Token at:")
	fmt.Println("  https://id.atlassian.com/manage-profile/security/api-tokens")
	fmt.Println()
	fmt.Println("Select \"Confluence\" as the app, then add these scopes:")
	fmt.Println()
	fmt.Println("  Read-only (8 scopes):")
	for _, s := range ScopeReadOnly {
		fmt.Printf("    %s\n", s)
	}
	fmt.Println()
	fmt.Println("  Write access (add these 6 for full access):")
	for _, s := range ScopeWrite {
		fmt.Printf("    %s\n", s)
	}
	fmt.Println()
	fmt.Println("Write tools are only registered when CTK_ENABLE_WRITES=true is set.")
	fmt.Println("To explicitly deny tools, use:")
	fmt.Println("  export CTK_DISABLED_TOOLS=\"manage_folders,manage_labels\"")
	fmt.Println()

	fmt.Print("Confluence domain (e.g., 'mycompany' for mycompany.atlassian.net): ")
	domain, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading domain: %w", err)
	}
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return fmt.Errorf("domain is required")
	}
	// Strip .atlassian.net if user provided full domain
	domain = strings.TrimSuffix(domain, ".atlassian.net")

	fmt.Print("Atlassian email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading email: %w", err)
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}

	fmt.Print("API Token: ")
	token, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading API token: %w", err)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("API token is required")
	}

	// Verify credentials by hitting the spaces API
	fmt.Println("\nVerifying credentials...")
	client := NewClient(domain, email, token)
	_, err = client.Get("/wiki/api/v2/spaces?limit=1")
	if err != nil {
		return fmt.Errorf("credential verification failed: %w\n\nCheck that your domain, email, and API token are correct", err)
	}

	fmt.Println("Credentials verified successfully!")

	creds := &Credentials{
		Domain:    domain,
		Email:     email,
		APIToken:  token,
		CreatedAt: time.Now(),
	}

	if err := SaveCredentials(creds); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	path, _ := CredentialsPath()
	fmt.Printf("\nCredentials saved to: %s\n", path)
	fmt.Println("You can now use the Confluence MCP server.")
	return nil
}
