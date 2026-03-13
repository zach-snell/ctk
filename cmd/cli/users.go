package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/ctk/internal/confluence"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage Confluence users",
}

var usersMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the currently authenticated user",
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		user, err := client.GetCurrentUser()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, user, func() {
			fmt.Printf("User: %s\n", user.DisplayName)
			KV("Account ID", user.AccountID)
			if user.Email != "" {
				KV("Email", user.Email)
			}
			KV("Type", user.Type)
		})
	},
}

var usersSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for users by name or email",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		result, err := client.SearchUsers(confluence.SearchUsersArgs{
			Query: args[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Results) == 0 {
				fmt.Println("No users found.")
				return
			}
			t := NewTable()
			t.Header("Title", "Type", "URL")
			for _, r := range result.Results {
				t.Row(Truncate(r.Title, 40), r.EntityType, Truncate(r.URL, 60))
			}
			t.Flush()
			fmt.Printf("\nShowing %d results\n", len(result.Results))
		})
	},
}

func init() {
	RootCmd.AddCommand(usersCmd)
	usersCmd.AddCommand(usersMeCmd)
	usersCmd.AddCommand(usersSearchCmd)
}
