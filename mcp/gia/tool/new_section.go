package tool

import (
	"context"

	"github.com/anwerj/gia-db/go"
	"github.com/mark3labs/mcp-go/mcp"
)

// NewSectionTool appends one section to a resource.
type NewSectionTool struct {
	DB *gia.Client
}

func (t *NewSectionTool) Name() string {
	return "new_section"
}

func (t *NewSectionTool) Define(context.Context) mcp.Tool {
	return mcp.NewTool(
		t.Name(),
		mcp.WithDescription("Append one section to a GiaDB resource. resource_id is the issue number. The returned section id is the comment id. The body is stored exactly as supplied."),
		mcp.WithInteger("resource_id", mcp.Required(), mcp.Description("Resource ID. This is the GitHub issue number, not a comment id.")),
		mcp.WithString("body", mcp.Required(), mcp.Description("Section body, stored exactly.")),
	)
}

func (t *NewSectionTool) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if t.DB == nil {
		return nil, errNilClient
	}
	args := arguments(request)
	id, err := requiredInt64(args, "resource_id")
	if err != nil {
		return invalidArgument(err.Error())
	}
	body, present, err := optionalString(args, "body")
	if err != nil {
		return invalidArgument(err.Error())
	}
	if !present {
		return invalidArgument("body is required")
	}
	section, meta, err := t.DB.NewSection(ctx, id, body)
	if err != nil {
		return fromSDK(err)
	}
	return sectionResult(section, meta)
}
