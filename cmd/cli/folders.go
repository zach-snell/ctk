package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/ctk/internal/confluence"
)

var foldersCmd = &cobra.Command{
	Use:   "folders",
	Short: "Manage Confluence folders",
}

var foldersListCmd = &cobra.Command{
	Use:   "list [space-id]",
	Short: "List folders in a space",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		result, err := client.ListFolders(args[0], 0, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Results) == 0 {
				fmt.Println("No folders found.")
				return
			}
			t := NewTable()
			t.Header("ID", "Title", "Status", "Version")
			for _, f := range result.Results {
				ver := "-"
				if f.Version != nil {
					ver = fmt.Sprintf("%d", f.Version.Number)
				}
				t.Row(f.ID, Truncate(f.Title, 60), f.Status, ver)
			}
			t.Flush()
			fmt.Printf("\nShowing %d folders\n", len(result.Results))
		})
	},
}

var foldersGetCmd = &cobra.Command{
	Use:   "get [folder-id]",
	Short: "Get folder details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		folder, err := client.GetFolder(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, folder, func() {
			flat := confluence.FlattenFolder(folder)
			fmt.Printf("Folder: %s\n", flat.Title)
			KV("ID", flat.ID)
			if flat.SpaceID != "" {
				KV("Space ID", flat.SpaceID)
			}
			if flat.ParentID != "" {
				KV("Parent ID", flat.ParentID)
			}
			KV("Status", flat.Status)
			if flat.Version > 0 {
				KV("Version", fmt.Sprintf("%d", flat.Version))
			}
			if flat.Created != "" {
				KV("Created", flat.Created)
			}
			if flat.Updated != "" {
				KV("Updated", flat.Updated)
			}
		})
	},
}

var foldersChildrenCmd = &cobra.Command{
	Use:   "children [folder-id]",
	Short: "Get child folders",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		result, err := client.GetFolderChildren(args[0], 0, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Results) == 0 {
				fmt.Println("No child folders found.")
				return
			}
			t := NewTable()
			t.Header("ID", "Title", "Status", "Version")
			for _, f := range result.Results {
				ver := "-"
				if f.Version != nil {
					ver = fmt.Sprintf("%d", f.Version.Number)
				}
				t.Row(f.ID, Truncate(f.Title, 60), f.Status, ver)
			}
			t.Flush()
			fmt.Printf("\nShowing %d child folders\n", len(result.Results))
		})
	},
}

func init() {
	RootCmd.AddCommand(foldersCmd)
	foldersCmd.AddCommand(foldersListCmd)
	foldersCmd.AddCommand(foldersGetCmd)
	foldersCmd.AddCommand(foldersChildrenCmd)
}
