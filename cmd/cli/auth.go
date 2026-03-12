package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/ctk/internal/confluence"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Confluence Cloud",
	Long: `Set up credentials for accessing Confluence Cloud.

This stores your Confluence domain, email, and API token for use
by the CLI and MCP server.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := confluence.InteractiveLogin(); err != nil {
			fmt.Fprintf(os.Stderr, "auth failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	Run: func(cmd *cobra.Command, args []string) {
		runStatus()
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove stored credentials",
	Run: func(cmd *cobra.Command, args []string) {
		runLogout()
	},
}

func init() {
	RootCmd.AddCommand(authCmd)
	RootCmd.AddCommand(statusCmd)
	RootCmd.AddCommand(logoutCmd)
}

func runStatus() {
	// Check environment variables first
	if envCreds := confluence.LoadCredentialsFromEnv(); envCreds != nil {
		fmt.Println("Authenticated via environment variables")
		fmt.Printf("  Domain:  %s.atlassian.net\n", envCreds.Domain)
		fmt.Printf("  Email:   %s\n", envCreds.Email)
		return
	}

	creds, err := confluence.LoadCredentials()
	if err != nil {
		fmt.Println("Not authenticated. Run: ctk auth")
		return
	}

	path, _ := confluence.CredentialsPath()

	fmt.Println("Authenticated via stored credentials")
	fmt.Printf("  Domain:  %s.atlassian.net\n", creds.Domain)
	fmt.Printf("  Email:   %s\n", creds.Email)
	if len(creds.APIToken) > 8 {
		fmt.Printf("  Token:   %s...%s\n", creds.APIToken[:4], creds.APIToken[len(creds.APIToken)-4:])
	} else {
		fmt.Println("  Token:   ****")
	}
	fmt.Printf("  Stored:  %s\n", creds.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  File:    %s\n", path)
}

func runLogout() {
	if err := confluence.RemoveCredentials(); err != nil {
		fmt.Fprintf(os.Stderr, "error removing credentials: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Logged out. Credentials removed.")
}
