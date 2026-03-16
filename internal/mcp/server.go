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

// New creates and configures the Confluence MCP server with a classic-auth client.
// Prefer NewFromCredentials for automatic token type detection.
func New(domain, email, token string) *mcp.Server {
	client := confluence.NewClient(domain, email, token)
	return newServer(client)
}

// NewFromCredentials creates the MCP server from stored credentials.
func NewFromCredentials(creds *confluence.Credentials) (*mcp.Server, error) {
	client, err := confluence.NewClientFromCredentials(creds)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	return newServer(client), nil
}

// NewUnauthenticated creates an MCP server that registers all tools but returns
// auth-required errors when any tool is called. Used for inspection/listing (e.g., Glama Docker builds).
func NewUnauthenticated() *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "ctk",
			Version: version.Version,
		},
		nil,
	)
	registerToolsUnauthenticated(s)
	return s
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
	spaceActions := "'list', 'get', 'get_by_key'"
	if canWrite {
		spaceActions += ", 'create'"
	}
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_spaces",
		Description: fmt.Sprintf("Unified tool for listing and getting Confluence spaces. Actions: %s. Write operations: %v", spaceActions, canWrite),
	}, ManageSpacesHandler(c, canWrite))

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
		Description: fmt.Sprintf("Unified tool for Confluence attachment operations (list, download%s)", writeActions(canWrite, ", upload, delete")),
	}, ManageAttachmentsHandler(c, canWrite))

	// ─── Users ───────────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_users",
		Description: "Search and get Confluence users. Actions: 'get_current', 'search'",
	}, ManageUsersHandler(c))
}

func writeActions(canWrite bool, actions string) string {
	if canWrite {
		return actions
	}
	return ""
}

const authRequiredMsg = `Authentication required. Configure credentials via:
  1. Run: ctk auth
  2. Set environment variables: CONFLUENCE_DOMAIN + CONFLUENCE_EMAIL + CONFLUENCE_API_TOKEN
  3. Config file: ~/.config/ctk/credentials.json`

// addUnauthenticatedTool registers a tool that always returns an auth-required message.
func addUnauthenticatedTool[In any](s *mcp.Server, tool mcp.Tool) {
	mcp.AddTool(s, &tool, func(_ context.Context, _ *mcp.CallToolRequest, _ In) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: authRequiredMsg},
			},
		}, nil, nil
	})
}

func registerToolsUnauthenticated(s *mcp.Server) {
	addUnauthenticatedTool[ManageSpacesArgs](s, mcp.Tool{
		Name:        "manage_spaces",
		Description: "Unified tool for listing and getting Confluence spaces. Actions: 'list', 'get', 'get_by_key', 'create'. Write operations: false",
	})

	addUnauthenticatedTool[ManagePagesArgs](s, mcp.Tool{
		Name:        "manage_pages",
		Description: "Unified tool for Confluence page operations (get, get_by_title, list, get_children, get_ancestors, list_versions, diff)",
	})

	addUnauthenticatedTool[ManageSearchArgs](s, mcp.Tool{
		Name:        "manage_search",
		Description: "Unified tool for Confluence search (CQL and quick text search)",
	})

	addUnauthenticatedTool[ManageLabelsArgs](s, mcp.Tool{
		Name:        "manage_labels",
		Description: "Unified tool for managing page labels (list)",
	})

	addUnauthenticatedTool[ManageFoldersArgs](s, mcp.Tool{
		Name:        "manage_folders",
		Description: "Unified tool for Confluence folder operations (list, get, get_children)",
	})

	addUnauthenticatedTool[ManageCommentsArgs](s, mcp.Tool{
		Name:        "manage_comments",
		Description: "Unified tool for Confluence comments (list_footer, list_inline, get_replies)",
	})

	addUnauthenticatedTool[ManageAttachmentsArgs](s, mcp.Tool{
		Name:        "manage_attachments",
		Description: "Unified tool for Confluence attachment operations (list, download)",
	})

	addUnauthenticatedTool[ManageUsersArgs](s, mcp.Tool{
		Name:        "manage_users",
		Description: "Search and get Confluence users. Actions: 'get_current', 'search'",
	})
}
