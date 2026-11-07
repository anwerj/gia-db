package tool

import (
	"context"

	"github.com/anwerj/gia-db/go"
	"github.com/mark3labs/mcp-go/mcp"
)

// ListResourcesTool lists one page of resources.
type ListResourcesTool struct {
	DB *gia.Client
}

func (t *ListResourcesTool) Name() string {
	return "list_resources"
}

func (t *ListResourcesTool) Define(context.Context) mcp.Tool {
	return mcp.NewTool(
		t.Name(),
		mcp.WithDescription("List GiaDB resources. Returns one page, an opaque next_token, and search metadata when the SDK used GitHub search. Pass next_token back unchanged. Resource id values are issue numbers."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithInteger("page_size", mcp.Description("Page size from 1 to 100. Omit to use the SDK default. Invalid sizes are rejected by the SDK.")),
		mcp.WithString("page_token", mcp.Description("Opaque continuation token from a previous page. Do not interpret or modify it.")),
		mcp.WithArray("ids", mcp.Description("Exact resource IDs, which are issue numbers."), mcp.WithIntegerItems()),
		mcp.WithString("title_match", mcp.Description("Title text matched through GitHub search.")),
		mcp.WithString("created_since", mcp.Description("RFC3339 lower bound for resource creation time.")),
		mcp.WithString("updated_since", mcp.Description("RFC3339 lower bound for resource update time.")),
		mcp.WithString("state", mcp.Description("Lifecycle filter."), mcp.Enum("active", "forgotten", "all")),
		mcp.WithArray("labels", mcp.Description("Labels that must all be present."), mcp.WithStringItems()),
		mcp.WithString("sort", mcp.Description("Sort field."), mcp.Enum("created", "updated")),
		mcp.WithString("direction", mcp.Description("Sort direction."), mcp.Enum("asc", "desc")),
	)
}

func (t *ListResourcesTool) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if t.DB == nil {
		return nil, errNilClient
	}
	args := arguments(request)
	options := gia.ListOptions{}
	pageSize, present, err := optionalInt(args, "page_size")
	if err != nil {
		return invalidArgument(err.Error())
	}
	if present {
		options.PageSize = pageSize
	}
	pageToken, _, err := optionalString(args, "page_token")
	if err != nil {
		return invalidArgument(err.Error())
	}
	options.PageToken = pageToken
	ids, present, err := optionalInt64s(args, "ids")
	if err != nil {
		return invalidArgument(err.Error())
	}
	if present {
		options.IDs = ids
	}
	titleMatch, _, err := optionalString(args, "title_match")
	if err != nil {
		return invalidArgument(err.Error())
	}
	options.TitleMatch = titleMatch
	createdSince, err := optionalTime(args, "created_since")
	if err != nil {
		return invalidArgument(err.Error())
	}
	options.CreatedSince = createdSince
	updatedSince, err := optionalTime(args, "updated_since")
	if err != nil {
		return invalidArgument(err.Error())
	}
	options.UpdatedSince = updatedSince
	state, _, err := optionalString(args, "state")
	if err != nil {
		return invalidArgument(err.Error())
	}
	options.State = gia.StateFilter(state)
	labels, present, err := optionalStrings(args, "labels")
	if err != nil {
		return invalidArgument(err.Error())
	}
	if present {
		options.AllLabels = labels
	}
	sort, _, err := optionalString(args, "sort")
	if err != nil {
		return invalidArgument(err.Error())
	}
	options.Sort = gia.SortField(sort)
	direction, _, err := optionalString(args, "direction")
	if err != nil {
		return invalidArgument(err.Error())
	}
	options.Direction = gia.SortDirection(direction)
	page, err := t.DB.List(ctx, options)
	if err != nil {
		return fromSDK(err)
	}
	return resourcePageResult(page)
}
