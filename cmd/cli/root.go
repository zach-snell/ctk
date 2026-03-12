package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/ctk/internal/version"
)

// RootCmd represents the base command when called without any subcommands.
var RootCmd = &cobra.Command{
	Use:     "ctk",
	Version: version.Version,
	Short:   "A unified CLI and MCP server for Confluence Cloud",
	Long: `ctk is a complete command-line interface and Model Context Protocol
server for Confluence Cloud.

It allows you to manage spaces, pages, and search directly from your
terminal, or expose these capabilities to your AI agents via the MCP
protocol.

Try running 'ctk auth' to get started!`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().Bool("json", false, "Output raw JSON instead of formatted tables")
}
