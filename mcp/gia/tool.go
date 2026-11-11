package gia

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// Tool is the contract implemented by every GiaDB MCP tool.
type Tool interface {
	Name() string
	Define(ctx context.Context) mcp.Tool
	Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
}
