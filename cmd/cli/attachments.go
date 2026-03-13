package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/ctk/internal/confluence"
)

var attachmentsCmd = &cobra.Command{
	Use:   "attachments",
	Short: "Manage page attachments",
}

var attachmentsListCmd = &cobra.Command{
	Use:   "list [page-id]",
	Short: "List attachments on a page",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		result, err := client.ListAttachments(confluence.ListAttachmentsArgs{
			PageID: args[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Results) == 0 {
				fmt.Println("No attachments found.")
				return
			}
			t := NewTable()
			t.Header("ID", "Title", "Media Type", "Size", "Created")
			for _, a := range result.Results {
				size := fmt.Sprintf("%d", a.FileSize)
				t.Row(a.ID, Truncate(a.Title, 40), a.MediaType, size, FormatTime(a.CreatedAt))
			}
			t.Flush()
			fmt.Printf("\nShowing %d attachments\n", len(result.Results))
		})
	},
}

var attachmentsDownloadCmd = &cobra.Command{
	Use:   "download [attachment-id]",
	Short: "Download an attachment to a local file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()

		// First get the attachment metadata to find the download URL
		att, err := client.GetAttachment(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting attachment: %v\n", err)
			os.Exit(1)
		}

		if att.DownloadURL == "" {
			fmt.Fprintf(os.Stderr, "Error: attachment has no download URL\n")
			os.Exit(1)
		}

		data, mediaType, err := client.DownloadAttachment(att.DownloadURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error downloading: %v\n", err)
			os.Exit(1)
		}

		outFile, _ := cmd.Flags().GetString("output")
		if outFile == "" {
			outFile = att.Title
		}

		if err := os.WriteFile(outFile, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, map[string]string{
			"id":        att.ID,
			"title":     att.Title,
			"mediaType": mediaType,
			"file":      outFile,
			"size":      fmt.Sprintf("%d", len(data)),
		}, func() {
			fmt.Printf("Downloaded %s (%s, %d bytes) → %s\n", att.Title, mediaType, len(data), outFile)
		})
	},
}

func init() {
	RootCmd.AddCommand(attachmentsCmd)
	attachmentsCmd.AddCommand(attachmentsListCmd)
	attachmentsCmd.AddCommand(attachmentsDownloadCmd)

	attachmentsDownloadCmd.Flags().StringP("output", "o", "", "Output filename (defaults to attachment title)")
}
