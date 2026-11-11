package tool

import (
	"context"

	"github.com/anwerj/gia-db/go"
	"github.com/mark3labs/mcp-go/mcp"
)

// UpdateResourceTool patches one resource.
type UpdateResourceTool struct {
	DB *gia.Client
}

func (t *UpdateResourceTool) Name() string {
	return "update_resource"
}

func (t *UpdateResourceTool) Define(context.Context) mcp.Tool {
	return mcp.NewTool(
		t.Name(),
		mcp.WithDescription("Patch one GiaDB resource. Omitted title, body, and labels stay unchanged. A supplied body replaces the issue body exactly. A supplied labels array, including an empty array, replaces the complete label set. id is the issue number."),
		mcp.WithInteger("id", mcp.Required(), mcp.Description("Resource ID. This is the GitHub issue number.")),
		mcp.WithString("title", mcp.Description("Replacement title. Omit to leave the title unchanged.")),
		mcp.WithString("body", mcp.Description("Replacement issue body, stored exactly. Omit to leave the body unchanged. An empty string clears the body.")),
		mcp.WithArray("labels", mcp.Description("Replacement label set. Omit to leave labels unchanged. An empty array clears labels."), mcp.WithStringItems()),
	)
}

func (t *UpdateResourceTool) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if t.DB == nil {
		return nil, errNilClient
	}
	args := arguments(request)
	id, err := requiredInt64(args, "id")
	if err != nil {
		return invalidArgument(err.Error())
	}
	input := gia.UpdateInput{}
	title, present, err := optionalString(args, "title")
	if err != nil {
		return invalidArgument(err.Error())
	}
	if present {
		input.Title = &title
	}
	body, present, err := optionalString(args, "body")
	if err != nil {
		return invalidArgument(err.Error())
	}
	if present {
		input.Body = &body
	}
	labels, present, err := optionalStrings(args, "labels")
	if err != nil {
		return invalidArgument(err.Error())
	}
	if present {
		input.Labels = &labels
	}
	resource, meta, err := t.DB.Update(ctx, id, input)
	if err != nil {
		return fromSDK(err)
	}
	return resourceResult(resource, meta)
}
