package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/ctk/internal/confluence"
	"github.com/zach-snell/ctk/internal/version"
)

// New creates and configures the Confluence MCP server with all tools registered.
func New(domain, email, token string) *mcp.Server {
	client := confluence.NewClient(domain, email, token)
	return newServer(client)
}

// NewFromCredentials creates the MCP server from stored credentials.
func NewFromCredentials(creds *confluence.Credentials) *mcp.Server {
	client := confluence.NewClientFromCredentials(creds)
	return newServer(client)
}

func newServer(client *confluence.Client) *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "ctk",
			Version: version.Version,
		},
		nil,
	)

	registerTools(s, client)
	return s
}

// addTool is a helper function to conditionally register a generic tool handler.
func addTool[In any](s *mcp.Server, disabled map[string]bool, tool mcp.Tool, handler func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, any, error)) {
	if disabled[tool.Name] {
		return
	}
	mcp.AddTool(s, &tool, handler)
}

func registerTools(s *mcp.Server, c *confluence.Client) {
	disabledToolsEnv := os.Getenv("CTK_DISABLED_TOOLS")
	disabled := make(map[string]bool)
	if disabledToolsEnv != "" {
		for _, t := range strings.Split(disabledToolsEnv, ",") {
			disabled[strings.TrimSpace(t)] = true
		}
	}

	canWrite := os.Getenv("CTK_ENABLE_WRITES") == "true"

	// ─── Spaces ──────────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_spaces",
		Description: fmt.Sprintf("Unified tool for listing and getting Confluence spaces. Write operations: %v", canWrite),
	}, ManageSpacesHandler(c))

	// ─── Pages ───────────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_pages",
		Description: fmt.Sprintf("Unified tool for Confluence page operations (get, get_by_title, list, get_children, get_ancestors, list_versions, diff%s)", writeActions(canWrite, ", create, update, delete, move")),
	}, ManagePagesHandler(c, canWrite))

	// ─── Search ──────────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_search",
		Description: "Unified tool for Confluence search (CQL and quick text search)",
	}, ManageSearchHandler(c))

	// ─── Labels ──────────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_labels",
		Description: fmt.Sprintf("Unified tool for managing page labels (list%s)", writeActions(canWrite, ", add, remove")),
	}, ManageLabelsHandler(c, canWrite))

	// ─── Folders ─────────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_folders",
		Description: fmt.Sprintf("Unified tool for Confluence folder operations (list, get, get_children%s)", writeActions(canWrite, ", create, update, delete")),
	}, ManageFoldersHandler(c, canWrite))

	// ─── Comments ────────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_comments",
		Description: fmt.Sprintf("Unified tool for Confluence comments (list_footer, list_inline, get_replies%s)", writeActions(canWrite, ", add_footer, reply")),
	}, ManageCommentsHandler(c, canWrite))

	// ─── Attachments ─────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_attachments",
		Description: "Unified tool for Confluence attachment operations (list, download)",
	}, ManageAttachmentsHandler(c))
}

func writeActions(canWrite bool, actions string) string {
	if canWrite {
		return actions
	}
	return ""
}
