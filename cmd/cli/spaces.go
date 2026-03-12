package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/ctk/internal/confluence"
)

var spacesCmd = &cobra.Command{
	Use:   "spaces",
	Short: "Manage Confluence spaces",
}

var spacesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List spaces the authenticated user has access to",
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		result, err := client.ListSpaces(confluence.ListSpacesArgs{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Results) == 0 {
				fmt.Println("No spaces found.")
				return
			}
			t := NewTable()
			t.Header("ID", "Key", "Name", "Type", "Status")
			for _, s := range result.Results {
				t.Row(s.ID, s.Key, s.Name, s.Type, s.Status)
			}
			t.Flush()
			fmt.Printf("\nShowing %d spaces\n", len(result.Results))
		})
	},
}

var spacesGetCmd = &cobra.Command{
	Use:   "get [space-key]",
	Short: "Get details for a specific space by key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		space, err := client.GetSpaceByKey(confluence.GetSpaceByKeyArgs{Key: args[0]})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, space, func() {
			fmt.Printf("Space: %s\n", space.Name)
			KV("ID", space.ID)
			KV("Key", space.Key)
			KV("Type", space.Type)
			KV("Status", space.Status)
			if space.HomepageID != "" {
				KV("Homepage ID", space.HomepageID)
			}
		})
	},
}

func init() {
	RootCmd.AddCommand(spacesCmd)
	spacesCmd.AddCommand(spacesListCmd)
	spacesCmd.AddCommand(spacesGetCmd)
}

// getClient is a helper to instantiate the core Confluence API client.
func getClient() *confluence.Client {
	// Check environment variables first
	if envCreds := confluence.LoadCredentialsFromEnv(); envCreds != nil {
		return confluence.NewClientFromCredentials(envCreds)
	}

	creds, err := confluence.LoadCredentials()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Not authenticated. Run 'ctk auth' first.\n")
		os.Exit(1)
	}

	return confluence.NewClientFromCredentials(creds)
}
