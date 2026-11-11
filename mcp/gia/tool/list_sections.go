package tool

import (
	"context"

	"github.com/anwerj/gia-db/go"
	"github.com/mark3labs/mcp-go/mcp"
)

// ListSectionsTool lists one page of sections for a resource.
type ListSectionsTool struct {
	DB *gia.Client
}

func (t *ListSectionsTool) Name() string {
	return "list_sections"
}

func (t *ListSectionsTool) Define(context.Context) mcp.Tool {
	return mcp.NewTool(
		t.Name(),
		mcp.WithDescription("List sections of one GiaDB resource. resource_id is the issue number. Each section id is a comment id. Pass next_token back unchanged."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithInteger("resource_id", mcp.Required(), mcp.Description("Resource ID. This is the GitHub issue number, not a comment id.")),
		mcp.WithInteger("page_size", mcp.Description("Page size from 1 to 100. Omit to use the SDK default.")),
		mcp.WithString("page_token", mcp.Description("Opaque continuation token from a previous page. Do not interpret or modify it.")),
	)
}

func (t *ListSectionsTool) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if t.DB == nil {
		return nil, errNilClient
	}
	args := arguments(request)
	id, err := requiredInt64(args, "resource_id")
	if err != nil {
		return invalidArgument(err.Error())
	}
	page := gia.PageOptions{}
	pageSize, present, err := optionalInt(args, "page_size")
	if err != nil {
		return invalidArgument(err.Error())
	}
	if present {
		page.PageSize = pageSize
	}
	pageToken, _, err := optionalString(args, "page_token")
	if err != nil {
		return invalidArgument(err.Error())
	}
	page.PageToken = pageToken
	result, err := t.DB.ListSections(ctx, id, page)
	if err != nil {
		return fromSDK(err)
	}
	return sectionPageResult(result)
}
