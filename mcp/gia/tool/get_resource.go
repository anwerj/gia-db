package tool

import (
	"context"

	"github.com/anwerj/gia-db/go"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetResourceTool reads one resource by issue number.
type GetResourceTool struct {
	DB *gia.Client
}

func (t *GetResourceTool) Name() string {
	return "get_resource"
}

func (t *GetResourceTool) Define(context.Context) mcp.Tool {
	return mcp.NewTool(
		t.Name(),
		mcp.WithDescription("Get one GiaDB resource by issue number, whether it is active or forgotten. id is the issue number, not a comment id."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithInteger("id", mcp.Required(), mcp.Description("Resource ID. This is the GitHub issue number.")),
		mcp.WithString("if_none_match", mcp.Description("Optional ETag for a conditional read. A matching resource returns not_modified.")),
	)
}

func (t *GetResourceTool) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if t.DB == nil {
		return nil, errNilClient
	}
	args := arguments(request)
	id, err := requiredInt64(args, "id")
	if err != nil {
		return invalidArgument(err.Error())
	}
	ifNoneMatch, _, err := optionalString(args, "if_none_match")
	if err != nil {
		return invalidArgument(err.Error())
	}
	resource, meta, err := t.DB.Get(ctx, id, gia.GetOptions{IfNoneMatch: ifNoneMatch})
	if err != nil {
		return fromSDK(err)
	}
	return resourceResult(resource, meta)
}
