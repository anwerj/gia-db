package tool

import (
	"context"

	"github.com/anwerj/gia-db/go"
	"github.com/mark3labs/mcp-go/mcp"
)

// ForgetResourceTool closes a resource without changing its body or labels.
type ForgetResourceTool struct {
	DB *gia.Client
}

func (t *ForgetResourceTool) Name() string {
	return "forget_resource"
}

func (t *ForgetResourceTool) Define(context.Context) mcp.Tool {
	return mcp.NewTool(
		t.Name(),
		mcp.WithDescription("Forget a GiaDB resource by closing its issue. This does not change the body or labels. id is the issue number."),
		mcp.WithInteger("id", mcp.Required(), mcp.Description("Resource ID. This is the GitHub issue number.")),
	)
}

func (t *ForgetResourceTool) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if t.DB == nil {
		return nil, errNilClient
	}
	id, err := requiredInt64(arguments(request), "id")
	if err != nil {
		return invalidArgument(err.Error())
	}
	resource, meta, err := t.DB.Forget(ctx, id)
	if err != nil {
		return fromSDK(err)
	}
	return resourceResult(resource, meta)
}
