// Live HTTP server for manual checks against a real GitHub repository.
//
//	export GIADB_TEST_TOKEN="$MY_SKILL_REPO_TOKEN"
//	export GIADB_TEST_OWNER='anwerj'
//	export GIADB_TEST_REPOSITORY='my-skills-repo'
//	go run ./tests/live
//
// Listens on 127.0.0.1:8080 unless GIADB_TEST_ADDR is set. The process has no
// HTTP authentication. GIADB_TEST_TOKEN is sent only to GitHub.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/anwerj/gia-db/go"
)

const maxBodyBytes = 1 << 20

func main() {
	token := os.Getenv("GIADB_TEST_TOKEN")
	owner := os.Getenv("GIADB_TEST_OWNER")
	repository := os.Getenv("GIADB_TEST_REPOSITORY")
	if token == "" || owner == "" || repository == "" {
		fmt.Fprintln(os.Stderr, "GIADB_TEST_TOKEN, GIADB_TEST_OWNER, and GIADB_TEST_REPOSITORY are required")
		os.Exit(1)
	}
	addr := os.Getenv("GIADB_TEST_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8787"
	}
	client, err := gia.NewClient(gia.Options{
		Owner:      owner,
		Repository: repository,
		Token:      token,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           newServer(client).routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("gia live server listening on http://%s (%s/%s)", addr, owner, repository)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

type server struct {
	client *gia.Client
}

func newServer(client *gia.Client) *server {
	return &server{client: client}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /create", s.create)
	mux.HandleFunc("PUT /update", s.update)
	mux.HandleFunc("GET /get", s.get)
	mux.HandleFunc("GET /list", s.list)
	mux.HandleFunc("POST /forget", s.forget)
	mux.HandleFunc("POST /reopen", s.reopen)
	mux.HandleFunc("POST /section", s.section)
	mux.HandleFunc("GET /sections", s.sections)
	return mux
}

func (s *server) create(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	title := query.Get("title")
	if title == "" {
		badRequest(w, "title is required")
		return
	}
	labels, ok := stringList(w, query, "labels")
	if !ok {
		return
	}
	body, ok := readBody(w, r)
	if !ok {
		return
	}
	resource, _, err := s.client.Create(r.Context(), gia.CreateInput{
		Title:  title,
		Body:   body,
		Labels: labels,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resourceJSONFrom(resource))
}

func (s *server) update(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	id, ok := resourceID(w, query)
	if !ok {
		return
	}
	if !query.Has("update") {
		badRequest(w, "update is required")
		return
	}
	fields, err := jsonStringList(query.Get("update"))
	if err != nil {
		badRequest(w, "update must be a JSON array of strings")
		return
	}
	input := gia.UpdateInput{}
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if _, exists := seen[field]; exists {
			continue
		}
		seen[field] = struct{}{}
		switch field {
		case "title":
			if !query.Has("title") {
				badRequest(w, "title is required when listed in update")
				return
			}
			title := query.Get("title")
			input.Title = &title
		case "body":
			body, ok := readBody(w, r)
			if !ok {
				return
			}
			input.Body = &body
		case "labels":
			if !query.Has("labels") {
				badRequest(w, "labels is required when listed in update")
				return
			}
			labels, err := jsonStringList(query.Get("labels"))
			if err != nil {
				badRequest(w, "labels must be a JSON array of strings")
				return
			}
			input.Labels = &labels
		default:
			badRequest(w, "update contains an unknown field "+field)
			return
		}
	}
	resource, _, err := s.client.Update(r.Context(), id, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resourceJSONFrom(resource))
}

func (s *server) get(w http.ResponseWriter, r *http.Request) {
	id, ok := resourceID(w, r.URL.Query())
	if !ok {
		return
	}
	resource, _, err := s.client.Get(r.Context(), id, gia.GetOptions{
		IfNoneMatch: r.Header.Get("If-None-Match"),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resourceJSONFrom(resource))
}

func (s *server) list(w http.ResponseWriter, r *http.Request) {
	options, ok := listOptions(w, r)
	if !ok {
		return
	}
	page, err := s.client.List(r.Context(), options)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]resourceJSON, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, resourceJSONFrom(item))
	}
	writeJSON(w, http.StatusOK, resourcePageJSON{
		Items:         items,
		NextToken:     page.NextToken,
		Exhausted:     page.Exhausted,
		ReportedTotal: page.ReportedTotal,
		Incomplete:    page.Incomplete,
		Truncated:     page.Truncated,
	})
}

func (s *server) forget(w http.ResponseWriter, r *http.Request) {
	s.changeState(w, r, s.client.Forget)
}

func (s *server) reopen(w http.ResponseWriter, r *http.Request) {
	s.changeState(w, r, s.client.Reopen)
}

func (s *server) changeState(w http.ResponseWriter, r *http.Request, change func(ctx context.Context, id int64) (gia.Resource, gia.ResponseMeta, error)) {
	id, ok := resourceID(w, r.URL.Query())
	if !ok {
		return
	}
	resource, _, err := change(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resourceJSONFrom(resource))
}

func (s *server) section(w http.ResponseWriter, r *http.Request) {
	id, ok := resourceID(w, r.URL.Query())
	if !ok {
		return
	}
	body, ok := readBody(w, r)
	if !ok {
		return
	}
	section, _, err := s.client.NewSection(r.Context(), id, body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sectionJSONFrom(section))
}

func (s *server) sections(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	id, ok := resourceID(w, query)
	if !ok {
		return
	}
	page := gia.PageOptions{PageToken: query.Get("page_token")}
	if query.Has("page_size") {
		size, err := strconv.Atoi(query.Get("page_size"))
		if err != nil {
			badRequest(w, "page_size must be an integer")
			return
		}
		page.PageSize = size
	}
	result, err := s.client.ListSections(r.Context(), id, page)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]sectionJSON, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, sectionJSONFrom(item))
	}
	writeJSON(w, http.StatusOK, sectionPageJSON{
		Items:     items,
		NextToken: result.NextToken,
		Exhausted: result.Exhausted,
	})
}

type resourceJSON struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Labels    []string  `json:"labels"`
	State     gia.State `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ETag      string    `json:"etag"`
}

type resourcePageJSON struct {
	Items         []resourceJSON `json:"items"`
	NextToken     string         `json:"next_token"`
	Exhausted     bool           `json:"exhausted"`
	ReportedTotal *int           `json:"reported_total,omitempty"`
	Incomplete    bool           `json:"incomplete"`
	Truncated     bool           `json:"truncated"`
}

type sectionJSON struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type sectionPageJSON struct {
	Items     []sectionJSON `json:"items"`
	NextToken string        `json:"next_token"`
	Exhausted bool          `json:"exhausted"`
}

type errorJSON struct {
	Code    string `json:"code"`
	Message string `json:"message"`
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
		CreatedAt: resource.CreatedAt,
		UpdatedAt: resource.UpdatedAt,
		ETag:      resource.ETag,
	}
}

func sectionJSONFrom(section gia.Section) sectionJSON {
	return sectionJSON{
		ID:        section.ID,
		Body:      section.Body,
		Author:    section.Author,
		CreatedAt: section.CreatedAt,
		UpdatedAt: section.UpdatedAt,
	}
}

func listOptions(w http.ResponseWriter, r *http.Request) (gia.ListOptions, bool) {
	query := r.URL.Query()
	options := gia.ListOptions{
		PageOptions: gia.PageOptions{PageToken: query.Get("page_token")},
		TitleMatch:  query.Get("title"),
		State:       gia.StateFilter(query.Get("state")),
		Sort:        gia.SortField(query.Get("sort")),
		Direction:   gia.SortDirection(query.Get("direction")),
	}
	if query.Has("labels") {
		labels, err := jsonStringList(query.Get("labels"))
		if err != nil {
			badRequest(w, "labels must be a JSON array of strings")
			return gia.ListOptions{}, false
		}
		options.AllLabels = labels
	}
	if query.Has("ids") {
		ids, err := parseIDs(query.Get("ids"))
		if err != nil {
			badRequest(w, err.Error())
			return gia.ListOptions{}, false
		}
		options.IDs = ids
	}
	if query.Has("page_size") {
		size, err := strconv.Atoi(query.Get("page_size"))
		if err != nil {
			badRequest(w, "page_size must be an integer")
			return gia.ListOptions{}, false
		}
		options.PageSize = size
	}
	if query.Has("created_since") {
		created, err := time.Parse(time.RFC3339, query.Get("created_since"))
		if err != nil {
			badRequest(w, "created_since must be RFC3339")
			return gia.ListOptions{}, false
		}
		options.CreatedSince = &created
	}
	if query.Has("updated_since") {
		updated, err := time.Parse(time.RFC3339, query.Get("updated_since"))
		if err != nil {
			badRequest(w, "updated_since must be RFC3339")
			return gia.ListOptions{}, false
		}
		options.UpdatedSince = &updated
	}
	return options, true
}

func resourceID(w http.ResponseWriter, query url.Values) (int64, bool) {
	raw := query.Get("id")
	if raw == "" {
		badRequest(w, "id is required")
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		badRequest(w, "id must be a positive integer")
		return 0, false
	}
	return id, true
}

func stringList(w http.ResponseWriter, query url.Values, key string) ([]string, bool) {
	if !query.Has(key) {
		return nil, true
	}
	parsed, err := jsonStringList(query.Get(key))
	if err != nil {
		badRequest(w, key+" must be a JSON array of strings")
		return nil, false
	}
	return parsed, true
}

func jsonStringList(raw string) ([]string, error) {
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil || values == nil {
		return nil, fmt.Errorf("expected a JSON array of strings")
	}
	return values, nil
}

func parseIDs(raw string) ([]int64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("ids must be a comma-separated list of positive integers")
	}
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("ids must be a comma-separated list of positive integers")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func readBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	data, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil {
		badRequest(w, "read body")
		return "", false
	}
	if len(data) > maxBodyBytes {
		badRequest(w, "body exceeds 1 MiB")
		return "", false
	}
	return string(data), true
}

func badRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, errorJSON{
		Code:    string(gia.CodeInvalidArgument),
		Message: message,
	})
}

func writeError(w http.ResponseWriter, err error) {
	var giaErr *gia.Error
	if !errors.As(err, &giaErr) {
		writeJSON(w, http.StatusInternalServerError, errorJSON{Code: string(gia.CodeGitHub), Message: err.Error()})
		return
	}
	if giaErr.Code == gia.CodeNotModified {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	writeJSON(w, statusFor(giaErr.Code), errorJSON{Code: string(giaErr.Code), Message: giaErr.Message})
}

func statusFor(code gia.ErrorCode) int {
	switch code {
	case gia.CodeInvalidArgument, gia.CodeValidation:
		return http.StatusBadRequest
	case gia.CodeNotFound:
		return http.StatusNotFound
	case gia.CodeAuthentication:
		return http.StatusUnauthorized
	case gia.CodePermission:
		return http.StatusForbidden
	case gia.CodeInvalidResource:
		return http.StatusUnprocessableEntity
	case gia.CodeRateLimited:
		return http.StatusTooManyRequests
	case gia.CodeNotModified:
		return http.StatusNotModified
	case gia.CodeTransport, gia.CodeGitHub, gia.CodeOutcomeUnknown:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}
