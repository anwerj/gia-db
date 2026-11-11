package tool

import (
	"context"

	"github.com/anwerj/gia-db/go"
	"github.com/mark3labs/mcp-go/mcp"
)

// CreateResourceTool creates one active resource.
type CreateResourceTool struct {
	DB *gia.Client
}

func (t *CreateResourceTool) Name() string {
	return "create_resource"
}

func (t *CreateResourceTool) Define(context.Context) mcp.Tool {
	return mcp.NewTool(
		t.Name(),
		mcp.WithDescription("Create one active GiaDB resource. The body is stored exactly as supplied. Labels are optional."),
		mcp.WithString("title", mcp.Required(), mcp.Description("Resource title.")),
		mcp.WithString("body", mcp.Description("Issue body, stored exactly. Omit to store an empty body.")),
		mcp.WithArray("labels", mcp.Description("Optional label names. Omit to create the resource without labels."), mcp.WithStringItems()),
	)
}

func (t *CreateResourceTool) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if t.DB == nil {
		return nil, errNilClient
	}
	args := arguments(request)
	title, present, err := optionalString(args, "title")
	if err != nil {
		return invalidArgument(err.Error())
	}
	if !present {
		return invalidArgument("title is required")
	}
	body, _, err := optionalString(args, "body")
	if err != nil {
		return invalidArgument(err.Error())
	}
	labels, _, err := optionalStrings(args, "labels")
	if err != nil {
		return invalidArgument(err.Error())
	}
	resource, meta, err := t.DB.Create(ctx, gia.CreateInput{
		Title:  title,
		Body:   body,
		Labels: labels,
	})
	if err != nil {
		return fromSDK(err)
	}
	return resourceResult(resource, meta)
}
