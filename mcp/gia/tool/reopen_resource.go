package tool

import (
	"context"

	"github.com/anwerj/gia-db/go"
	"github.com/mark3labs/mcp-go/mcp"
)

// ReopenResourceTool reopens a forgotten resource without changing its body or labels.
type ReopenResourceTool struct {
	DB *gia.Client
}

func (t *ReopenResourceTool) Name() string {
	return "reopen_resource"
}

func (t *ReopenResourceTool) Define(context.Context) mcp.Tool {
	return mcp.NewTool(
		t.Name(),
		mcp.WithDescription("Reopen a forgotten GiaDB resource. This does not change the body or labels. id is the issue number."),
		mcp.WithInteger("id", mcp.Required(), mcp.Description("Resource ID. This is the GitHub issue number.")),
	)
}

func (t *ReopenResourceTool) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if t.DB == nil {
		return nil, errNilClient
	}
	id, err := requiredInt64(arguments(request), "id")
	if err != nil {
		return invalidArgument(err.Error())
	}
	resource, meta, err := t.DB.Reopen(ctx, id)
	if err != nil {
		return fromSDK(err)
	}
	return resourceResult(resource, meta)
}
