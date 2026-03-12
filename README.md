# ctk (Confluence CLI & MCP Server)

[![Documentation](https://img.shields.io/badge/docs-reference-blue)](https://zach-snell.github.io/ctk/)
[![Go Report Card](https://goreportcard.com/badge/github.com/zach-snell/ctk)](https://goreportcard.com/report/github.com/zach-snell/ctk)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A complete command-line interface and Model Context Protocol (MCP) server written in Go that provides programmatic integration with Confluence Cloud.

<p align="center">
  <img src="demo.gif" alt="ctk CLI demo" width="700" />
</p>

## Features

- **Dual Mode**: Run as a rich, interactive CLI tool for daily developer tasks, or as an MCP server for AI agents.
- **V2 API First**: Uses the modern Confluence V2 REST API with cursor-based pagination, falling back to V1 for CQL search.
- **Folder Support**: Full CRUD for Confluence folders — the only dedicated Confluence MCP with folder operations.
- **Page Diff**: LCS-based unified diff between any two page versions, rendered as standard diff output.
- **Write Gating**: Mutation tools are only registered when `CTK_ENABLE_WRITES=true`, providing safe read-only defaults for AI agents.
- **Token-Efficient**: Consolidated action-based tools minimize schema injection overhead. XHTML→Markdown converter strips storage format verbosity. ResponseFlattener removes bloated metadata.

## Installation

### From Source
```bash
# Clone the repository
git clone https://github.com/zach-snell/ctk.git
cd ctk

# Run the install script (builds and moves to ~/.local/bin)
./install.sh
```

Ensure `~/.local/bin` is added to your system `$PATH` for the executable to be universally available.

### From GitHub Releases
Download the appropriate binary for your system (Linux, macOS, Windows) from the [Releases](https://github.com/zach-snell/ctk/releases) page.

## CLI Usage

`ctk` provides a robust command-line interface with the following core modules:

```bash
# Authenticate (stores credentials in ~/.config/ctk/)
ctk auth

# Manage spaces
ctk spaces [list, get]

# Manage pages (full CRUD + versions + diff)
ctk pages [get, list, create, update, delete, move, versions, diff]

# Manage folders
ctk folders [list, get, create, update, delete]

# Search with CQL or quick text
ctk search --cql "type = page AND space = DEV"
```

## MCP Usage

The tool also serves as an MCP server. It supports two protocols: Stdio (default via `ctk mcp`) and the official Streamable Transport API over HTTP.

### Stdio Transport (Default)
If you intend to use this with an MCP client (such as Claude Desktop or Cursor), add it to your client's configuration file as a local command:

```json
{
  "mcpServers": {
    "confluence": {
      "command": "/absolute/path/to/ctk",
      "args": ["mcp"],
      "env": {
        "CONFLUENCE_DOMAIN": "your-domain.atlassian.net",
        "CONFLUENCE_EMAIL": "you@example.com",
        "CONFLUENCE_API_TOKEN": "your-api-token",
        "CTK_ENABLE_WRITES": "true"
      }
    }
  }
}
```

### Streamable Transport (HTTP)
You can run the server as a long-lived HTTP process serving the Streamable Transport API (which uses Server-Sent Events underneath). This is useful for remote network clients.

```bash
ctk mcp --port 8080
```

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `CONFLUENCE_DOMAIN` | Your Atlassian domain (e.g., `acme.atlassian.net`) | Yes |
| `CONFLUENCE_EMAIL` | Email associated with the API token | Yes |
| `CONFLUENCE_API_TOKEN` | An Atlassian API Token | Yes |
| `CTK_ENABLE_WRITES` | Set to `true` to enable mutation tools | No (defaults to read-only) |
| `CTK_DISABLED_TOOLS` | Comma-separated tool names to hide from AI agents | No |

### API Token Scopes

Create a **Confluence** app token at [id.atlassian.com](https://id.atlassian.com/manage-profile/security/api-tokens) with granular scopes. Run `ctk auth` to see the full recommended scope list, or use these:

**Read-only (8 scopes):**
```
read:space:confluence, read:page:confluence, read:folder:confluence,
read:hierarchical-content:confluence, read:comment:confluence,
read:label:confluence, read:attachment:confluence, search:confluence
```

**Full access (add these 6):**
```
write:page:confluence, write:folder:confluence, write:comment:confluence,
write:label:confluence, delete:page:confluence, delete:folder:confluence
```

### Write Gating & Security

**Two-layer safety model:**

1. **Token scopes** — granular Atlassian scopes control which APIs the token can call (403 if missing)
2. **Write gating** — mutation tools are only registered when `CTK_ENABLE_WRITES=true` is set (read-only by default)

**Explicit Tool Denial:** Even with writes enabled, you can explicitly deny the AI agent access to any tool:

```bash
export CTK_DISABLED_TOOLS="manage_folders,manage_labels"
```

## Tools Provided

| Tool | Description |
|------|-------------|
| `manage_spaces` | Space operations — list, get, list pages in space |
| `manage_pages` | Page operations — get, get by title, list, get children, get ancestors, list versions, diff, create, update, delete, move |
| `manage_search` | Search via CQL or quick text with cursor-based pagination |
| `manage_labels` | Label operations — list, add, remove |
| `manage_folders` | Folder operations — list, get, get children, create, update, delete |
| `manage_comments` | Comment operations — list footer, list inline, get replies, add footer, reply |
| `manage_attachments` | Attachment operations — list and download |

## Development

Requirements:
- Go 1.26+

```bash
# Run tests
go test ./...

# Run the linter
golangci-lint run ./...
```

## License

This project is licensed under the [Apache 2.0 License](LICENSE).
