package tool

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/anwerj/gia-db/go"
	"github.com/mark3labs/mcp-go/mcp"
)

var errNilClient = errors.New("GiaDB client is nil")

type jsonTime struct {
	time.Time
}

func (t jsonTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.UTC().Format(time.RFC3339))
}

type rateJSON struct {
	Limit     int        `json:"limit"`
	Remaining int        `json:"remaining"`
	ResetAt   *time.Time `json:"reset_at"`
	RetryAt   *time.Time `json:"retry_at"`
}

type metaJSON struct {
	RequestID string   `json:"request_id"`
	ETag      string   `json:"etag"`
	Rate      rateJSON `json:"rate"`
}

type resourceJSON struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Labels    []string  `json:"labels"`
	State     gia.State `json:"state"`
	CreatedAt jsonTime  `json:"created_at"`
	UpdatedAt jsonTime  `json:"updated_at"`
	ETag      string    `json:"etag"`
}

type sectionJSON struct {
	ID        int64    `json:"id"`
	Body      string   `json:"body"`
	Author    string   `json:"author"`
	CreatedAt jsonTime `json:"created_at"`
	UpdatedAt jsonTime `json:"updated_at"`
}

type resourceDocument struct {
	Resource resourceJSON `json:"resource"`
	Meta     metaJSON     `json:"meta"`
}

type sectionDocument struct {
	Section sectionJSON `json:"section"`
	Meta    metaJSON    `json:"meta"`
}

type resourcePageDocument struct {
	Resources     []resourceJSON `json:"resources"`
	NextToken     string         `json:"next_token"`
	Exhausted     bool           `json:"exhausted"`
	ReportedTotal *int           `json:"reported_total"`
	Incomplete    bool           `json:"incomplete"`
	Truncated     bool           `json:"truncated"`
	Meta          metaJSON       `json:"meta"`
}

type sectionPageDocument struct {
	Sections  []sectionJSON `json:"sections"`
	NextToken string        `json:"next_token"`
	Exhausted bool          `json:"exhausted"`
	Meta      metaJSON      `json:"meta"`
}

type errorDocument struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      gia.ErrorCode `json:"code"`
	Message   string        `json:"message"`
	Outcome   gia.Outcome   `json:"outcome"`
	Status    int           `json:"status"`
	RequestID string        `json:"request_id"`
	Rate      rateJSON      `json:"rate"`
}

func resourceResult(resource gia.Resource, meta gia.ResponseMeta) (*mcp.CallToolResult, error) {
	return textJSON(resourceDocument{
		Resource: resourceJSONFrom(resource),
		Meta:     metaJSONFrom(meta),
	})
}

func sectionResult(section gia.Section, meta gia.ResponseMeta) (*mcp.CallToolResult, error) {
	return textJSON(sectionDocument{
		Section: sectionJSONFrom(section),
		Meta:    metaJSONFrom(meta),
	})
}

func resourcePageResult(page gia.ResourcePage) (*mcp.CallToolResult, error) {
	items := make([]resourceJSON, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, resourceJSONFrom(item))
	}
	return textJSON(resourcePageDocument{
		Resources:     items,
		NextToken:     page.NextToken,
		Exhausted:     page.Exhausted,
		ReportedTotal: page.ReportedTotal,
		Incomplete:    page.Incomplete,
		Truncated:     page.Truncated,
		Meta:          metaJSONFrom(page.Meta),
	})
}

func sectionPageResult(page gia.SectionPage) (*mcp.CallToolResult, error) {
	items := make([]sectionJSON, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, sectionJSONFrom(item))
	}
	return textJSON(sectionPageDocument{
		Sections:  items,
		NextToken: page.NextToken,
		Exhausted: page.Exhausted,
		Meta:      metaJSONFrom(page.Meta),
	})
}

func resourceJSONFrom(resource gia.Resource) resourceJSON {
	labels := resource.Labels
	if labels == nil {
		labels = []string{}
	}
	return resourceJSON{
		ID:        resource.ID,
		Title:     resource.Title,
		Body:      resource.Body,
		Labels:    labels,
		State:     resource.State,
		CreatedAt: jsonTime{resource.CreatedAt},
		UpdatedAt: jsonTime{resource.UpdatedAt},
		ETag:      resource.ETag,
	}
}

func sectionJSONFrom(section gia.Section) sectionJSON {
	return sectionJSON{
		ID:        section.ID,
		Body:      section.Body,
		Author:    section.Author,
		CreatedAt: jsonTime{section.CreatedAt},
		UpdatedAt: jsonTime{section.UpdatedAt},
	}
}

func metaJSONFrom(meta gia.ResponseMeta) metaJSON {
	return metaJSON{
		RequestID: meta.RequestID,
		ETag:      meta.ETag,
		Rate: rateJSON{
			Limit:     meta.Rate.Limit,
			Remaining: meta.Rate.Remaining,
			ResetAt:   timePtr(meta.Rate.ResetAt),
			RetryAt:   timePtr(meta.Rate.RetryAt),
		},
	}
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func textJSON(value any) (*mcp.CallToolResult, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(payload)), nil
}

func invalidArgument(message string) (*mcp.CallToolResult, error) {
	return fromSDK(&gia.Error{
		Code:    gia.CodeInvalidArgument,
		Message: message,
		Outcome: gia.OutcomeNotApplied,
	})
}

func fromSDK(err error) (*mcp.CallToolResult, error) {
	var apiErr *gia.Error
	if !errors.As(err, &apiErr) {
		return nil, err
	}
	payload, marshalErr := json.Marshal(errorDocument{Error: errorBody{
		Code:      apiErr.Code,
		Message:   apiErr.Message,
		Outcome:   apiErr.Outcome,
		Status:    apiErr.Status,
		RequestID: apiErr.RequestID,
		Rate: rateJSON{
			Limit:     apiErr.Rate.Limit,
			Remaining: apiErr.Rate.Remaining,
			ResetAt:   timePtr(apiErr.Rate.ResetAt),
			RetryAt:   timePtr(apiErr.Rate.RetryAt),
		},
	}})
	if marshalErr != nil {
		return nil, marshalErr
	}
	return mcp.NewToolResultError(string(payload)), nil
}

func arguments(request mcp.CallToolRequest) map[string]any {
	args := request.GetArguments()
	if args == nil {
		return map[string]any{}
	}
	return args
}

func requiredInt64(args map[string]any, key string) (int64, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return 0, fmt.Errorf("%s is required", key)
	}
	return int64Value(key, value)
}

func optionalInt(args map[string]any, key string) (int, bool, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return 0, false, nil
	}
	parsed, err := int64Value(key, value)
	if err != nil {
		return 0, true, err
	}
	if int64(int(parsed)) != parsed {
		return 0, true, fmt.Errorf("%s must be an integer", key)
	}
	return int(parsed), true, nil
}

func optionalInt64s(args map[string]any, key string) ([]int64, bool, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return nil, false, nil
	}
	switch typed := value.(type) {
	case []int64:
		return append([]int64(nil), typed...), true, nil
	case []int:
		out := make([]int64, len(typed))
		for i, item := range typed {
			out[i] = int64(item)
		}
		return out, true, nil
	case []any:
		out := make([]int64, len(typed))
		for i, item := range typed {
			parsed, err := int64Value(key, item)
			if err != nil {
				return nil, true, err
			}
			out[i] = parsed
		}
		return out, true, nil
	default:
		return nil, true, fmt.Errorf("%s must be an array of integers", key)
	}
}

func optionalString(args map[string]any, key string) (string, bool, error) {
	value, ok := args[key]
	if !ok {
		return "", false, nil
	}
	if value == nil {
		return "", true, fmt.Errorf("%s must be a string", key)
	}
	text, ok := value.(string)
	if !ok {
		return "", true, fmt.Errorf("%s must be a string", key)
	}
	return text, true, nil
}

func optionalStrings(args map[string]any, key string) ([]string, bool, error) {
	value, ok := args[key]
	if !ok {
		return nil, false, nil
	}
	if value == nil {
		return nil, true, fmt.Errorf("%s must be an array of strings", key)
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), true, nil
	case []any:
		out := make([]string, len(typed))
		for i, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, true, fmt.Errorf("%s must be an array of strings", key)
			}
			out[i] = text
		}
		return out, true, nil
	default:
		return nil, true, fmt.Errorf("%s must be an array of strings", key)
	}
}

func optionalTime(args map[string]any, key string) (*time.Time, error) {
	text, present, err := optionalString(args, key)
	if err != nil || !present {
		return nil, err
	}
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC3339", key)
	}
	utc := parsed.UTC()
	return &utc, nil
}

func int64Value(key string, value any) (int64, error) {
	switch typed := value.(type) {
	case int:
		return int64(typed), nil
	case int64:
		return typed, nil
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) || typed != math.Trunc(typed) {
			return 0, fmt.Errorf("%s must be an integer", key)
		}
		return int64(typed), nil
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer", key)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("%s must be an integer", key)
	}
}
