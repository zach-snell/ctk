package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zach-snell/ctk/internal/confluence"
)

var quickSearch bool

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search Confluence with CQL or quick text search",
	Long: `Search Confluence using CQL (Confluence Query Language) or quick text search.

CQL Examples:
  ctk search "type=page AND space=DEV"
  ctk search "title~'architecture' AND type=page"
  ctk search "label='important' AND space=TEAM"

Quick text search:
  ctk search --quick "migration guide"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		query := args[0]

		var result *confluence.SearchResult
		var err error

		if quickSearch {
			result, err = client.QuickSearch(confluence.QuickSearchArgs{
				Query: query,
			})
		} else {
			result, err = client.CQLSearch(confluence.CQLSearchArgs{
				CQL: query,
			})
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Results) == 0 {
				fmt.Println("No results found.")
				return
			}

			flat := confluence.FlattenSearchResults(result)
			t := NewTable()
			t.Header("Type", "Space", "Title", "Excerpt")
			for _, r := range flat {
				excerpt := Truncate(strings.TrimSpace(r.Excerpt), 60)
				t.Row(r.Type, r.SpaceKey, Truncate(r.Title, 40), excerpt)
			}
			t.Flush()
			fmt.Printf("\nShowing %d of %d results\n", len(result.Results), result.TotalSize)
		})
	},
}

func init() {
	RootCmd.AddCommand(searchCmd)
	searchCmd.Flags().BoolVarP(&quickSearch, "quick", "q", false, "Use quick text search instead of CQL")
}
