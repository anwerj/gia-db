package gia

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/anwerj/gia-db/go/internal/githubapi"
)

type Options struct {
	Owner       string
	Repository  string
	Token       string
	HTTPClient  *http.Client
	RESTBaseURL string
	GraphQLURL  string
}

type Client struct {
	api *githubapi.Client
}

func NewClient(options Options) (*Client, error) {
	if strings.TrimSpace(options.Owner) == "" {
		return nil, invalidArgument("owner is required")
	}
	if strings.TrimSpace(options.Repository) == "" {
		return nil, invalidArgument("repository is required")
	}
	api, err := githubapi.New(githubapi.Config{
		Owner:       options.Owner,
		Repository:  options.Repository,
		Token:       options.Token,
		HTTPClient:  options.HTTPClient,
		RESTBaseURL: options.RESTBaseURL,
		GraphQLURL:  options.GraphQLURL,
	})
	if err != nil {
		return nil, invalidArgument(err.Error())
	}
	return &Client{api: api}, nil
}

func (c *Client) Create(ctx context.Context, input CreateInput) (Resource, ResponseMeta, error) {
	if strings.TrimSpace(input.Title) == "" {
		return Resource{}, ResponseMeta{}, invalidArgument("title is required")
	}
	issue, meta, err := c.api.Create(ctx, input.Title, input.Body, dedupe(input.Labels))
	if err != nil {
		return Resource{}, fromMeta(meta), fromAPIError(err)
	}
	resource, err := resourceFromMutation(issue)
	return resource, fromMeta(meta), err
}

func (c *Client) Get(ctx context.Context, id int64, options GetOptions) (Resource, ResponseMeta, error) {
	if id <= 0 {
		return Resource{}, ResponseMeta{}, invalidArgument("resource ID must be positive")
	}
	issue, meta, err := c.api.Get(ctx, id, options.IfNoneMatch)
	if err != nil {
		return Resource{}, fromMeta(meta), fromAPIError(err)
	}
	resource, err := resourceFromIssue(issue)
	if errors.Is(err, errNonResource) {
		err = &Error{Code: CodeNotFound, Message: "pull request is not a GiaDB resource", Outcome: OutcomeNotApplied}
	}
	return resource, fromMeta(meta), err
}

func (c *Client) Update(ctx context.Context, id int64, input UpdateInput) (Resource, ResponseMeta, error) {
	if id <= 0 {
		return Resource{}, ResponseMeta{}, invalidArgument("resource ID must be positive")
	}
	if input.Title == nil && input.Body == nil && input.Labels == nil {
		return Resource{}, ResponseMeta{}, invalidArgument("update contains no fields")
	}
	if _, _, err := c.Get(ctx, id, GetOptions{}); err != nil {
		return Resource{}, ResponseMeta{}, err
	}

	var body *string
	if input.Body != nil {
		body = input.Body
	}
	var labels *[]string
	if input.Labels != nil {
		replaced := dedupe(*input.Labels)
		labels = &replaced
	}

	issue, meta, err := c.api.Update(ctx, id, input.Title, body, labels)
	if err != nil {
		return Resource{}, fromMeta(meta), fromAPIError(err)
	}
	resource, err := resourceFromMutation(issue)
	return resource, fromMeta(meta), err
}

func (c *Client) NewSection(ctx context.Context, id int64, body string) (Section, ResponseMeta, error) {
	if id <= 0 {
		return Section{}, ResponseMeta{}, invalidArgument("resource ID must be positive")
	}
	if _, meta, err := c.Get(ctx, id, GetOptions{}); err != nil {
		return Section{}, meta, err
	}
	comment, meta, err := c.api.AppendComment(ctx, id, body)
	if err != nil {
		return Section{}, fromMeta(meta), fromAPIError(err)
	}
	return sectionFromComment(comment), fromMeta(meta), nil
}

func (c *Client) ListSections(ctx context.Context, id int64, page PageOptions) (SectionPage, error) {
	if id <= 0 {
		return SectionPage{}, invalidArgument("resource ID must be positive")
	}
	if err := validatePage(page); err != nil {
		return SectionPage{}, err
	}
	if _, _, err := c.Get(ctx, id, GetOptions{}); err != nil {
		return SectionPage{}, err
	}
	result, err := c.api.ListComments(ctx, id, page.PageSize, page.PageToken)
	if err != nil {
		return SectionPage{}, fromAPIError(err)
	}
	items := make([]Section, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, sectionFromComment(item))
	}
	return SectionPage{
		Items:     items,
		NextToken: result.NextToken,
		Exhausted: result.NextToken == "",
		Meta:      fromMeta(result.Meta),
	}, nil
}

func (c *Client) List(ctx context.Context, options ListOptions) (ResourcePage, error) {
	if err := validateList(options); err != nil {
		return ResourcePage{}, err
	}
	input := githubapi.ListInput{
		PageSize:  options.PageSize,
		PageToken: options.PageToken,
		State:     string(normalizeState(options.State)),
		Labels:    options.AllLabels,
		Sort:      string(normalizeSort(options.Sort)),
		Direction: string(normalizeDirection(options.Direction)),
	}

	var (
		result githubapi.IssuePage
		err    error
	)
	switch {
	case len(options.IDs) != 0:
		result, err = c.api.ListByIDs(ctx, options.IDs, input)
	case options.TitleMatch != "" || options.CreatedSince != nil || options.UpdatedSince != nil:
		result, err = c.api.ListMatching(ctx, githubapi.ListMatchInput{
			PageSize:     options.PageSize,
			PageToken:    options.PageToken,
			Text:         options.TitleMatch,
			In:           "title",
			CreatedSince: options.CreatedSince,
			UpdatedSince: options.UpdatedSince,
			State:        input.State,
			Labels:       input.Labels,
			Sort:         input.Sort,
			Direction:    input.Direction,
		})
	default:
		result, err = c.api.ListIssues(ctx, input)
	}
	if err != nil {
		return ResourcePage{}, fromAPIError(err)
	}
	items, err := resourcesFromIssues(result.Items)
	if err != nil {
		return ResourcePage{}, err
	}
	if len(options.IDs) != 0 {
		items = filterResources(items, options)
	}
	var reportedTotal *int
	if result.Searched {
		total := result.Total
		reportedTotal = &total
	}
	return ResourcePage{
		Items:         items,
		NextToken:     result.NextToken,
		Exhausted:     result.NextToken == "",
		Meta:          fromMeta(result.Meta),
		ReportedTotal: reportedTotal,
		Incomplete:    result.Incomplete,
		Truncated:     result.Truncated,
	}, nil
}

func (c *Client) Forget(ctx context.Context, id int64) (Resource, ResponseMeta, error) {
	return c.setState(ctx, id, "closed")
}

func (c *Client) Reopen(ctx context.Context, id int64) (Resource, ResponseMeta, error) {
	return c.setState(ctx, id, "open")
}

func (c *Client) setState(ctx context.Context, id int64, state string) (Resource, ResponseMeta, error) {
	if id <= 0 {
		return Resource{}, ResponseMeta{}, invalidArgument("resource ID must be positive")
	}
	if _, _, err := c.Get(ctx, id, GetOptions{}); err != nil {
		return Resource{}, ResponseMeta{}, err
	}
	issue, meta, err := c.api.SetState(ctx, id, state)
	if err != nil {
		return Resource{}, fromMeta(meta), fromAPIError(err)
	}
	resource, err := resourceFromMutation(issue)
	return resource, fromMeta(meta), err
}

var errNonResource = errors.New("not a GiaDB resource")

func resourceFromMutation(issue githubapi.Issue) (Resource, error) {
	resource, err := resourceFromIssue(issue)
	if errors.Is(err, errNonResource) {
		return Resource{}, &Error{
			Code:    CodeNotFound,
			Message: "pull request is not a GiaDB resource",
			Outcome: OutcomeNotApplied,
		}
	}
	return resource, err
}

func resourceFromIssue(issue githubapi.Issue) (Resource, error) {
	if issue.IsPR {
		return Resource{}, errNonResource
	}
	state := StateActive
	if issue.State == "closed" {
		state = StateForgotten
	}
	return Resource{
		ID:        issue.Number,
		Title:     issue.Title,
		Body:      issue.Body,
		Labels:    slices.Clone(issue.Labels),
		State:     state,
		CreatedAt: issue.CreatedAt,
		UpdatedAt: issue.UpdatedAt,
		ETag:      issue.ETag,
	}, nil
}

func resourcesFromIssues(issues []githubapi.Issue) ([]Resource, error) {
	result := make([]Resource, 0, len(issues))
	for _, issue := range issues {
		resource, err := resourceFromIssue(issue)
		if errors.Is(err, errNonResource) {
			continue
		}
		if err != nil {
			return nil, err
		}
		result = append(result, resource)
	}
	return result, nil
}

func sectionFromComment(comment githubapi.Comment) Section {
	return Section{
		ID:        comment.ID,
		Body:      comment.Body,
		Author:    comment.Author,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}
}

func fromMeta(meta githubapi.Meta) ResponseMeta {
	return ResponseMeta{
		RequestID: meta.RequestID,
		ETag:      meta.ETag,
		Rate: Rate{
			Limit:     meta.Rate.Limit,
			Remaining: meta.Rate.Remaining,
			ResetAt:   meta.Rate.ResetAt,
			RetryAt:   meta.Rate.RetryAt,
		},
	}
}

func fromAPIError(err error) error {
	var apiErr *githubapi.Error
	if !errors.As(err, &apiErr) {
		return &Error{Code: CodeGitHub, Message: err.Error(), Outcome: OutcomeNotApplied, Cause: err}
	}
	code := ErrorCode(apiErr.Code)
	outcome := OutcomeNotApplied
	if apiErr.Unknown {
		outcome = OutcomeUnknown
	}
	return &Error{
		Code:      code,
		Message:   apiErr.Message,
		Outcome:   outcome,
		Status:    apiErr.Status,
		RequestID: apiErr.RequestID,
		Rate: Rate{
			Limit:     apiErr.Rate.Limit,
			Remaining: apiErr.Rate.Remaining,
			ResetAt:   apiErr.Rate.ResetAt,
			RetryAt:   apiErr.Rate.RetryAt,
		},
		Cause: apiErr.Cause,
	}
}

func dedupe(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func validatePage(page PageOptions) error {
	if page.PageSize < 0 || page.PageSize > 100 {
		return invalidArgument("page size must be between 1 and 100")
	}
	return nil
}

func validateList(options ListOptions) error {
	if err := validatePage(options.PageOptions); err != nil {
		return err
	}
	seenIDs := make(map[int64]struct{}, len(options.IDs))
	for _, id := range options.IDs {
		if id <= 0 {
			return invalidArgument("resource IDs must be positive")
		}
		if _, exists := seenIDs[id]; exists {
			return invalidArgument("resource IDs must be unique")
		}
		seenIDs[id] = struct{}{}
	}
	if !validState(options.State) {
		return invalidArgument("invalid state filter")
	}
	if !validSort(options.Sort) || !validDirection(options.Direction) {
		return invalidArgument("invalid sort option")
	}
	if len(options.IDs) != 0 && (options.Sort != "" || options.Direction != "") {
		return invalidArgument("ID-filtered lists preserve ID order and cannot be sorted")
	}
	return nil
}

func filterResources(resources []Resource, options ListOptions) []Resource {
	filtered := make([]Resource, 0, len(resources))
	title := strings.ToLower(options.TitleMatch)
	state := normalizeState(options.State)
	for _, resource := range resources {
		if title != "" && !strings.Contains(strings.ToLower(resource.Title), title) {
			continue
		}
		if options.CreatedSince != nil && resource.CreatedAt.Before(*options.CreatedSince) {
			continue
		}
		if options.UpdatedSince != nil && resource.UpdatedAt.Before(*options.UpdatedSince) {
			continue
		}
		if state == StateFilterActive && resource.State != StateActive {
			continue
		}
		if state == StateFilterForgotten && resource.State != StateForgotten {
			continue
		}
		matchesLabels := true
		for _, label := range options.AllLabels {
			if !slices.Contains(resource.Labels, label) {
				matchesLabels = false
				break
			}
		}
		if matchesLabels {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

func validState(state StateFilter) bool {
	switch state {
	case StateFilterDefault, StateFilterActive, StateFilterForgotten, StateFilterAll:
		return true
	default:
		return false
	}
}

func validSort(sort SortField) bool {
	return sort == "" || sort == SortCreated || sort == SortUpdated
}

func validDirection(direction SortDirection) bool {
	return direction == "" || direction == SortAscending || direction == SortDescending
}

func normalizeState(state StateFilter) StateFilter {
	if state == "" {
		return StateFilterActive
	}
	return state
}

func normalizeSort(sort SortField) SortField {
	if sort == "" {
		return SortCreated
	}
	return sort
}

func normalizeDirection(direction SortDirection) SortDirection {
	if direction == "" {
		return SortDescending
	}
	return direction
}

func ensureClient(c *Client) error {
	if c == nil || c.api == nil {
		return fmt.Errorf("GiaDB: nil client")
	}
	return nil
}
