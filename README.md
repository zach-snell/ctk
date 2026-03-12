# ctk — Confluence Toolkit

[![CI](https://github.com/zach-snell/ctk/actions/workflows/ci.yml/badge.svg)](https://github.com/zach-snell/ctk/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/zach-snell/ctk)](https://goreportcard.com/report/github.com/zach-snell/ctk)
[![Documentation](https://img.shields.io/badge/docs-reference-blue)](https://zach-snell.github.io/ctk/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

The most comprehensive dedicated Confluence MCP server in the open-source ecosystem. A dual-mode Go binary that works as both a rich CLI tool and an MCP server for AI agents.

**8 MCP tools · Full CLI · Write gating · Single binary · Zero dependencies**

## Why ctk?

| | ctk | mcp-atlassian (Python) |
|---|---|---|
| **Language** | Go (single ~15MB binary) | Python (pip install + deps) |
| **Confluence tools** | 8 dedicated tools | ~30 mixed Jira+Confluence |
| **Startup** | ~50ms | ~2s |
| **Folder support** | Full CRUD | None |
| **Page diff** | LCS-based unified diff | None |
| **Space create** | Yes (V1 stable API) | None |
| **Write gating** | `CTK_ENABLE_WRITES` env var | None |
| **Auth** | Classic + scoped tokens | Classic only |

## Features

- **Dual Mode** — CLI for humans, MCP server for AI agents, same binary
- **V2 API First** — Modern Confluence V2 REST API with cursor-based pagination, V1 fallback for CQL search and space create
- **Folder Support** — Full CRUD for Confluence folders — the only dedicated Confluence MCP with folder operations
- **Page Diff** — LCS-based unified diff between any two page versions
- **Write Gating** — Mutation tools only registered when `CTK_ENABLE_WRITES=true`, safe read-only defaults
- **Token-Efficient** — Consolidated action-based tools minimize schema overhead. XHTML↔Markdown conversion. ResponseFlattener strips metadata bloat
- **Markdown Interface** — Agents send/receive markdown, ctk converts to/from Confluence XHTML storage format internally

## Installation

```bash
# From source
git clone https://github.com/zach-snell/ctk.git && cd ctk
./install.sh  # builds and copies to ~/.local/bin

# Or build manually
go build -o ctk ./cmd/ctk
```

Pre-built binaries available on the [Releases](https://github.com/zach-snell/ctk/releases) page.

## Quick Start

```bash
# Authenticate
ctk auth

# List spaces
ctk spaces list

# Get a page
ctk pages get 12345

# Search with CQL
ctk search --cql "type = page AND space = DEV AND title ~ 'architecture'"

# Create a page (writes enabled)
ctk pages create --space-id 12345 --title "My Page" --body "# Hello World"
```

## CLI Commands

```
ctk auth                    Authenticate with Confluence Cloud
ctk spaces                  List, get, create spaces
ctk pages                   Page CRUD, versions, diff, move
ctk folders                 Folder CRUD, children
ctk search                  CQL and quick text search
```

## MCP Server

### Stdio Transport (Claude Desktop, Cursor, OpenCode, etc.)

```json
{
  "mcpServers": {
    "confluence": {
      "command": "/path/to/ctk",
      "args": ["mcp"],
      "env": {
        "CONFLUENCE_DOMAIN": "your-domain",
        "CONFLUENCE_EMAIL": "you@example.com",
        "CONFLUENCE_API_TOKEN": "your-api-token",
        "CTK_ENABLE_WRITES": "true"
      }
    }
  }
}
```

### Streamable HTTP Transport

```bash
ctk mcp --port 8080
```

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `CONFLUENCE_DOMAIN` | Atlassian domain (e.g., `acme` for acme.atlassian.net) | Yes |
| `CONFLUENCE_EMAIL` | Email for the API token | Yes |
| `CONFLUENCE_API_TOKEN` | Atlassian API token | Yes |
| `CONFLUENCE_TOKEN_TYPE` | `classic` or `scoped` (auto-detected if omitted) | No |
| `CTK_ENABLE_WRITES` | Set to `true` to enable mutation tools | No |
| `CTK_DISABLED_TOOLS` | Comma-separated tool names to hide | No |

## MCP Tools (8)

| Tool | Actions |
|------|---------|
| `manage_spaces` | list, get, get_by_key, create |
| `manage_pages` | get, get_by_title, list, get_children, get_ancestors, list_versions, diff, create, update, delete, move |
| `manage_search` | cql, quick |
| `manage_labels` | list, add, remove |
| `manage_folders` | list, get, get_children, create, update, delete |
| `manage_comments` | list_footer, list_inline, get_replies, add_footer, reply |
| `manage_attachments` | list, download, upload, delete |
| `manage_users` | get_current, search |

## Security

**Three-layer safety model:**

1. **Token scopes** — Atlassian scopes control which APIs the token can call (403 if missing)
2. **Write gating** — Mutation tools only registered when `CTK_ENABLE_WRITES=true` (read-only by default)
3. **Tool denial** — Explicitly hide tools: `CTK_DISABLED_TOOLS="manage_folders,manage_labels"`

## Development

```bash
go test -race ./...          # Run tests
golangci-lint run ./...      # Lint
go build -o ctk ./cmd/ctk   # Build
```

## License

[Apache 2.0](LICENSE)
