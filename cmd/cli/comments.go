package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/ctk/internal/confluence"
)

var commentsCmd = &cobra.Command{
	Use:   "comments",
	Short: "Manage page comments",
}

var commentsListCmd = &cobra.Command{
	Use:   "list [page-id]",
	Short: "List footer comments on a page",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		result, err := client.ListComments(confluence.ListCommentsArgs{
			PageID: args[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Results) == 0 {
				fmt.Println("No comments found.")
				return
			}
			t := NewTable()
			t.Header("ID", "Author", "Status", "Created", "Body")
			for _, c := range result.Results {
				flat := confluence.FlattenComment(&c)
				body := Truncate(flat.Body, 50)
				t.Row(flat.ID, flat.AuthorID, flat.Status, flat.Created, body)
			}
			t.Flush()
			fmt.Printf("\nShowing %d comments\n", len(result.Results))
		})
	},
}

func init() {
	RootCmd.AddCommand(commentsCmd)
	commentsCmd.AddCommand(commentsListCmd)
}
