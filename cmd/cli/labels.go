package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/ctk/internal/confluence"
)

var labelsCmd = &cobra.Command{
	Use:   "labels",
	Short: "Manage page labels",
}

var labelsListCmd = &cobra.Command{
	Use:   "list [page-id]",
	Short: "List labels on a page",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		result, err := client.ListLabels(confluence.ListLabelsArgs{
			PageID: args[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Results) == 0 {
				fmt.Println("No labels found.")
				return
			}
			t := NewTable()
			t.Header("ID", "Name", "Prefix")
			for _, l := range result.Results {
				t.Row(l.ID, l.Name, l.Prefix)
			}
			t.Flush()
			fmt.Printf("\nShowing %d labels\n", len(result.Results))
		})
	},
}

var labelsAddCmd = &cobra.Command{
	Use:   "add [page-id] [label]",
	Short: "Add a label to a page",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		label, err := client.AddLabel(confluence.AddLabelArgs{
			PageID: args[0],
			Name:   args[1],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, label, func() {
			fmt.Printf("Added label: %s\n", label.Name)
			KV("ID", label.ID)
			KV("Prefix", label.Prefix)
		})
	},
}

var labelsRemoveCmd = &cobra.Command{
	Use:   "remove [page-id] [label]",
	Short: "Remove a label from a page",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		err := client.RemoveLabel(confluence.RemoveLabelArgs{
			PageID: args[0],
			Label:  args[1],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, map[string]string{"status": "removed", "label": args[1]}, func() {
			fmt.Printf("Removed label %q from page %s\n", args[1], args[0])
		})
	},
}

func init() {
	RootCmd.AddCommand(labelsCmd)
	labelsCmd.AddCommand(labelsListCmd)
	labelsCmd.AddCommand(labelsAddCmd)
	labelsCmd.AddCommand(labelsRemoveCmd)
}
