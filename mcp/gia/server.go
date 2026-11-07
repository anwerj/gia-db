package gia

import (
	"context"
	"errors"
	"fmt"

	giadb "github.com/anwerj/gia-db/go"
	"github.com/anwerj/gia-db/mcp/gia/tool"
	"github.com/mark3labs/mcp-go/server"
)

// Build constructs the GiaDB MCP server and registers every concrete tool.
func Build(ctx context.Context, db *giadb.Client) (*server.MCPServer, error) {
	if db == nil {
		return nil, errors.New("GiaDB client is nil")
	}
	mcpServer := server.NewMCPServer(
		"GiaDB",
		Version,
		server.WithToolCapabilities(true),
	)
	if err := Register(ctx, mcpServer, registeredTools(db)); err != nil {
		return nil, err
	}
	return mcpServer, nil
}

func registeredTools(db *giadb.Client) []Tool {
	return []Tool{
		&tool.CreateResourceTool{DB: db},
		&tool.GetResourceTool{DB: db},
		&tool.UpdateResourceTool{DB: db},
		&tool.ListResourcesTool{DB: db},
		&tool.NewSectionTool{DB: db},
		&tool.ListSectionsTool{DB: db},
		&tool.ForgetResourceTool{DB: db},
		&tool.ReopenResourceTool{DB: db},
	}
}

// Register adds tools to the server. Each name must be unique, and Name must
// match the name returned by Define.
func Register(ctx context.Context, mcpServer *server.MCPServer, tools []Tool) error {
	seen := make(map[string]struct{}, len(tools))
	for _, item := range tools {
		name := item.Name()
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate tool name %q", name)
		}
		defined := item.Define(ctx)
		if defined.Name != name {
			return fmt.Errorf("tool %q Define name is %q", name, defined.Name)
		}
		seen[name] = struct{}{}
		mcpServer.AddTool(defined, item.Handle)
	}
	return nil
}
