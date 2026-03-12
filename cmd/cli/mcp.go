package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/zach-snell/ctk/internal/confluence"
	mcpserver "github.com/zach-snell/ctk/internal/mcp"
)

var port int

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Confluence MCP Server",
	Long: `Starts the Model Context Protocol (MCP) server for Confluence.
By default, this runs on stdio. You can provide a --port flag to
run it using the HTTP Streamable transport.`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func init() {
	RootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().IntVarP(&port, "port", "p", 0, "Port to listen on for HTTP Streamable transport")
}

func runServer() {
	// Priority: env vars > stored credentials
	domain := os.Getenv("CONFLUENCE_DOMAIN")
	email := os.Getenv("CONFLUENCE_EMAIL")
	apiToken := os.Getenv("CONFLUENCE_API_TOKEN")

	var s *mcp.Server

	if domain != "" && email != "" && apiToken != "" {
		s = mcpserver.New(domain, email, apiToken)
	} else {
		creds, err := confluence.LoadCredentials()
		if err != nil {
			fmt.Fprintf(os.Stderr, "No credentials found. Either:\n")
			fmt.Fprintf(os.Stderr, "  1. Run: ctk auth\n")
			fmt.Fprintf(os.Stderr, "  2. Set CONFLUENCE_DOMAIN + CONFLUENCE_EMAIL + CONFLUENCE_API_TOKEN env vars\n")
			os.Exit(1)
		}
		s = mcpserver.NewFromCredentials(creds)
	}

	if port != 0 {
		fmt.Printf("Starting Confluence MCP Server on :%d (HTTP Streamable)\n", port)
		handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
			return s
		}, &mcp.StreamableHTTPOptions{JSONResponse: false})

		srv := &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           handler,
			ReadHeaderTimeout: 3 * time.Second,
		}

		if err := srv.ListenAndServe(); err != nil {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}
}
