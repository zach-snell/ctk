package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/ctk/internal/confluence"
)

var (
	pageSpace    string
	pageTitle    string
	pageBody     string
	pageParentID string
)

var pagesCmd = &cobra.Command{
	Use:   "pages",
	Short: "Manage Confluence pages",
}

var pagesGetCmd = &cobra.Command{
	Use:   "get [page-id]",
	Short: "Get a page by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		page, err := client.GetPage(confluence.GetPageArgs{
			PageID:      args[0],
			IncludeBody: true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, page, func() {
			flat := confluence.FlattenPage(page)
			fmt.Printf("Page: %s\n", flat.Title)
			KV("ID", flat.ID)
			KV("Space ID", flat.SpaceID)
			KV("Status", flat.Status)
			if flat.Version > 0 {
				KV("Version", fmt.Sprintf("%d", flat.Version))
			}
			if flat.ParentID != "" {
				KV("Parent ID", flat.ParentID)
			}
			if flat.Created != "" {
				KV("Created", flat.Created)
			}
			if flat.Updated != "" {
				KV("Updated", flat.Updated)
			}
			if len(flat.Labels) > 0 {
				KV("Labels", fmt.Sprintf("%v", flat.Labels))
			}
			if flat.Body != "" {
				fmt.Println("\n--- Body ---")
				fmt.Println(flat.Body)
			}
		})
	},
}

var pagesListCmd = &cobra.Command{
	Use:   "list [space-id]",
	Short: "List pages in a space",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		result, err := client.ListPages(confluence.ListPagesArgs{
			SpaceID: args[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Results) == 0 {
				fmt.Println("No pages found.")
				return
			}
			t := NewTable()
			t.Header("ID", "Title", "Status", "Version")
			for _, p := range result.Results {
				ver := "-"
				if p.Version != nil {
					ver = fmt.Sprintf("%d", p.Version.Number)
				}
				t.Row(p.ID, Truncate(p.Title, 60), p.Status, ver)
			}
			t.Flush()
			fmt.Printf("\nShowing %d pages\n", len(result.Results))
		})
	},
}

var pagesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new page",
	Run: func(cmd *cobra.Command, args []string) {
		if pageSpace == "" || pageTitle == "" {
			fmt.Fprintf(os.Stderr, "Error: --space and --title are required\n")
			os.Exit(1)
		}
		client := getClient()
		page, err := client.CreatePage(confluence.CreatePageArgs{
			SpaceID:  pageSpace,
			Title:    pageTitle,
			Body:     pageBody,
			ParentID: pageParentID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, page, func() {
			fmt.Printf("Created page: %s (ID: %s)\n", page.Title, page.ID)
		})
	},
}

var pagesUpdateCmd = &cobra.Command{
	Use:   "update [page-id]",
	Short: "Update an existing page",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if pageTitle == "" {
			fmt.Fprintf(os.Stderr, "Error: --title is required\n")
			os.Exit(1)
		}

		client := getClient()

		// Get current version first
		current, err := client.GetPage(confluence.GetPageArgs{PageID: args[0]})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting current page: %v\n", err)
			os.Exit(1)
		}

		nextVersion := 1
		if current.Version != nil {
			nextVersion = current.Version.Number + 1
		}

		page, err := client.UpdatePage(confluence.UpdatePageArgs{
			PageID:  args[0],
			Title:   pageTitle,
			Body:    pageBody,
			Version: nextVersion,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, page, func() {
			fmt.Printf("Updated page: %s (ID: %s, version: %d)\n", page.Title, page.ID, nextVersion)
		})
	},
}

var pagesVersionsCmd = &cobra.Command{
	Use:   "versions [page-id]",
	Short: "List version history for a page",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		result, err := client.ListPageVersions(args[0], 0, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Results) == 0 {
				fmt.Println("No versions found.")
				return
			}
			t := NewTable()
			t.Header("Version", "Author", "Message", "Minor", "Created")
			for _, v := range result.Results {
				minor := ""
				if v.MinorEdit {
					minor = "yes"
				}
				msg := Truncate(v.Message, 40)
				t.Row(
					fmt.Sprintf("%d", v.Number),
					v.AuthorID,
					msg,
					minor,
					FormatTime(v.CreatedAt),
				)
			}
			t.Flush()
			fmt.Printf("\nShowing %d versions\n", len(result.Results))
		})
	},
}

var (
	diffFrom int
	diffTo   int
)

var pagesCommentCmd = &cobra.Command{
	Use:   "comment [page-id] [body]",
	Short: "Add a footer comment to a page",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		comment, err := client.AddFooterComment(args[0], args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, comment, func() {
			flat := confluence.FlattenComment(comment)
			fmt.Printf("Comment added (ID: %s)\n", flat.ID)
			if flat.Body != "" {
				fmt.Println(flat.Body)
			}
		})
	},
}

var pagesDiffCmd = &cobra.Command{
	Use:   "diff [page-id]",
	Short: "Compare two versions of a page",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if diffFrom == 0 || diffTo == 0 {
			fmt.Fprintf(os.Stderr, "Error: --from and --to are required\n")
			os.Exit(1)
		}

		client := getClient()
		diff, err := client.DiffPageVersions(args[0], diffFrom, diffTo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, diff, func() {
			fmt.Printf("Diff: %s (v%d → v%d)\n\n", diff.Title, diff.FromVersion, diff.ToVersion)
			fmt.Println(diff.Diff)
		})
	},
}

func init() {
	RootCmd.AddCommand(pagesCmd)
	pagesCmd.AddCommand(pagesGetCmd)
	pagesCmd.AddCommand(pagesListCmd)
	pagesCmd.AddCommand(pagesCreateCmd)
	pagesCmd.AddCommand(pagesUpdateCmd)
	pagesCmd.AddCommand(pagesVersionsCmd)
	pagesCmd.AddCommand(pagesCommentCmd)
	pagesCmd.AddCommand(pagesDiffCmd)

	pagesCreateCmd.Flags().StringVarP(&pageSpace, "space", "s", "", "Space ID to create page in")
	pagesCreateCmd.Flags().StringVarP(&pageTitle, "title", "t", "", "Page title")
	pagesCreateCmd.Flags().StringVarP(&pageBody, "body", "b", "", "Page body (Confluence storage format)")
	pagesCreateCmd.Flags().StringVar(&pageParentID, "parent", "", "Parent page ID")

	pagesUpdateCmd.Flags().StringVarP(&pageTitle, "title", "t", "", "New page title")
	pagesUpdateCmd.Flags().StringVarP(&pageBody, "body", "b", "", "New page body (Confluence storage format)")

	pagesDiffCmd.Flags().IntVar(&diffFrom, "from", 0, "Starting version number")
	pagesDiffCmd.Flags().IntVar(&diffTo, "to", 0, "Ending version number")
}
