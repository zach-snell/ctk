package confluence

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TokenType indicates the kind of Atlassian API token in use.
type TokenType string

const (
	// TokenTypeClassic is a classic API token — uses Basic Auth against the site URL.
	TokenTypeClassic TokenType = "classic"
	// TokenTypeScoped is a fine-grained/scoped API token — uses Bearer auth against the gateway URL.
	TokenTypeScoped TokenType = "scoped"
)

// Credentials holds persisted authentication data for Confluence Cloud.
type Credentials struct {
	Domain    string    `json:"domain"`               // e.g., "mycompany" (for mycompany.atlassian.net)
	Email     string    `json:"email"`                // Atlassian account email
	APIToken  string    `json:"api_token"`            // Atlassian API token
	CloudID   string    `json:"cloud_id,omitempty"`   // Atlassian Cloud ID (required for scoped tokens)
	Type      TokenType `json:"token_type,omitempty"` // "classic" or "scoped"
	CreatedAt time.Time `json:"created_at"`           // When credentials were stored
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

// ProbeTokenType determines whether a token is classic or scoped by testing both
// auth methods against the Confluence API. It tries basic auth first (classic),
// and falls back to Bearer auth via the gateway (scoped).
// Returns the detected token type and cloudID (empty for classic tokens).
func ProbeTokenType(domain, email, token string) (TokenType, string, error) {
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// 1. Try classic: Basic Auth against direct site URL
	classicURL := fmt.Sprintf("https://%s.atlassian.net/wiki/api/v2/spaces?limit=1", domain)
	req, err := http.NewRequest(http.MethodGet, classicURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("creating probe request: %w", err)
	}
	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("probing classic auth: %w", err)
	}
	resp.Body.Close()

	// Only 401 means "auth credentials rejected" — 403/404/etc mean auth worked
	// but the user lacks permissions or the resource doesn't exist.
	if resp.StatusCode != http.StatusUnauthorized {
		return TokenTypeClassic, "", nil
	}

	// 2. Classic rejected (401) — try scoped: Bearer Auth against gateway URL
	cloudID, err := FetchCloudID(domain)
	if err != nil {
		return "", "", fmt.Errorf("classic auth rejected and could not fetch Cloud ID for scoped fallback: %w", err)
	}

	gatewayURL := fmt.Sprintf("https://api.atlassian.com/ex/confluence/%s/wiki/api/v2/spaces?limit=1", cloudID)
	req, err = http.NewRequest(http.MethodGet, gatewayURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("creating gateway probe request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err = httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("probing scoped auth: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		return TokenTypeScoped, cloudID, nil
	}

	return "", "", fmt.Errorf("authentication failed with both classic (Basic Auth) and scoped (Bearer) methods. Verify your domain, email, and API token are correct")
}

// FetchCloudID retrieves the Atlassian Cloud ID for a given domain.
// This endpoint is public and requires no authentication.
func FetchCloudID(domain string) (string, error) {
	url := fmt.Sprintf("https://%s.atlassian.net/_edge/tenant_info", domain)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url) //nolint:gosec // URL is constructed from user-provided domain
	if err != nil {
		return "", fmt.Errorf("fetching tenant info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tenant info returned status %d (is '%s' a valid Atlassian domain?)", resp.StatusCode, domain)
	}

	var info struct {
		CloudID string `json:"cloudId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("parsing tenant info: %w", err)
	}
	if info.CloudID == "" {
		return "", fmt.Errorf("no cloudId found in tenant info for domain '%s'", domain)
	}

	return info.CloudID, nil
}

// LoadCredentialsFromEnv attempts to load credentials from environment variables.
// Returns nil if the env vars are not set.
// Token type can be explicitly set via CONFLUENCE_TOKEN_TYPE ("classic" or "scoped").
// If not set, it will be auto-detected during client creation.
func LoadCredentialsFromEnv() *Credentials {
	domain := os.Getenv("CONFLUENCE_DOMAIN")
	email := os.Getenv("CONFLUENCE_EMAIL")
	apiToken := os.Getenv("CONFLUENCE_API_TOKEN")

	if domain == "" || email == "" || apiToken == "" {
		return nil
	}

	creds := &Credentials{
		Domain:   domain,
		Email:    email,
		APIToken: apiToken,
		Type:     TokenType(os.Getenv("CONFLUENCE_TOKEN_TYPE")), // empty string = auto-detect
		CloudID:  os.Getenv("CONFLUENCE_CLOUD_ID"),
	}

	return creds
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

	fmt.Println("\nVerifying credentials (trying classic auth, then scoped)...")

	tokenType, cloudID, probeErr := ProbeTokenType(domain, email, token)
	if probeErr != nil {
		return fmt.Errorf("credential verification failed: %w", probeErr)
	}

	switch tokenType {
	case TokenTypeScoped:
		fmt.Printf("Authenticated via scoped token (Bearer, Cloud ID: %s)\n", cloudID)
	default:
		fmt.Println("Authenticated via classic token (Basic Auth)")
	}

	fmt.Println("Credentials verified successfully!")

	creds := &Credentials{
		Domain:    domain,
		Email:     email,
		APIToken:  token,
		CloudID:   cloudID,
		Type:      tokenType,
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
