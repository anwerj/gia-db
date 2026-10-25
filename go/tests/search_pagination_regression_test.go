package tests

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/anwerj/gia-db/go"
)

// Regression tests for GitHub Search pagination in List.
//
// The removed bug only emitted a continuation token when response.NextPage <= 10.
// That matches the last readable page only for per_page=100 (10 × 100 = 1,000).
// With per_page=20 and 500 matches, callers stopped after page 10 (200 issues)
// while Truncated stayed false because total_count was 500, not above 1,000.

func TestRegressionListPerPage20RetrievesAll500AcrossTwentyFivePages(t *testing.T) {
	t.Parallel()

	server := newIssueSearchServer(t, issueSearchServerConfig{total: 500})
	client := newSearchListClient(t, server)
	walk := walkListSearch(t, client, 20)

	if len(walk.requestedPages) != 25 {
		t.Fatalf("requested %d pages, want 25 (500 matches ÷ 20 per page)", len(walk.requestedPages))
	}
	for i, page := range walk.requestedPages {
		if page != i+1 {
			t.Fatalf("request order = %v, want pages 1..25", walk.requestedPages)
		}
	}
	if len(walk.ids) != 500 {
		t.Fatalf("retrieved %d issues, want 500", len(walk.ids))
	}
	for i, id := range walk.ids {
		if id != int64(i+1) {
			t.Fatalf("id at index %d = %d, want %d", i, id, i+1)
		}
	}
	if walk.anyTruncated || walk.exhausted != true || walk.lastToken != "" {
		t.Fatalf("final page exhausted=%v truncated=%v token=%q", walk.exhausted, walk.anyTruncated, walk.lastToken)
	}
	if walk.reportedTotal != 500 {
		t.Fatalf("reported total = %d, want 500", walk.reportedTotal)
	}
}

func TestRegressionListContinuationTokenProgressionPerPage20(t *testing.T) {
	t.Parallel()

	server := newIssueSearchServer(t, issueSearchServerConfig{total: 500})
	client := newSearchListClient(t, server)
	walk := walkListSearch(t, client, 20)

	if len(walk.tokenTargetPages) != 24 {
		t.Fatalf("got %d continuation tokens, want 24 (pages 2..25)", len(walk.tokenTargetPages))
	}
	for i, target := range walk.tokenTargetPages {
		if target != i+2 {
			t.Fatalf("token %d targets page %d, want %d", i, target, i+2)
		}
	}
	for i, raw := range walk.rawTokens {
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil {
			t.Fatalf("token %d: %v", i, err)
		}
		want := "search:" + strconv.Itoa(i+2)
		if string(decoded) != want {
			t.Fatalf("token %d = %q, want %q", i, decoded, want)
		}
	}
}

func TestRegressionListPerPage20DoesNotStopAfterPageTen(t *testing.T) {
	t.Parallel()

	server := newIssueSearchServer(t, issueSearchServerConfig{total: 500})
	client := newSearchListClient(t, server)

	token := ""
	var ids int
	for pageNum := 1; pageNum <= 11; pageNum++ {
		page, err := client.List(context.Background(), gia.ListOptions{
			PageOptions: gia.PageOptions{PageSize: 20, PageToken: token},
			TitleMatch:  "needle",
		})
		if err != nil {
			t.Fatal(err)
		}
		ids += len(page.Items)
		if pageNum == 10 {
			if page.NextToken == "" {
				t.Fatal("page 10 returned no continuation token; the old NextPage<=10 cap stopped here with only 200 issues")
			}
			if ids != 200 {
				t.Fatalf("after page 10 got %d issues, want 200", ids)
			}
		}
		if pageNum == 11 {
			if ids != 220 {
				t.Fatalf("after page 11 got %d issues, want 220", ids)
			}
		}
		token = page.NextToken
	}
}

func TestRegressionListExact1000ResultCeilingFiftyPagesPerPage20(t *testing.T) {
	t.Parallel()

	server := newIssueSearchServer(t, issueSearchServerConfig{total: 1000})
	client := newSearchListClient(t, server)
	walk := walkListSearch(t, client, 20)

	if len(walk.requestedPages) != 50 {
		t.Fatalf("requested %d pages, want 50 (1,000 ÷ 20)", len(walk.requestedPages))
	}
	if len(walk.ids) != 1000 {
		t.Fatalf("retrieved %d issues, want 1,000", len(walk.ids))
	}
	if walk.anyTruncated || walk.lastToken != "" {
		t.Fatalf("at ceiling truncated=%v token=%q", walk.anyTruncated, walk.lastToken)
	}
	if walk.reportedTotal != 1000 {
		t.Fatalf("reported total = %d, want 1,000", walk.reportedTotal)
	}
}

func TestRegressionListStopsAt1000WhenReportedTotalExceedsCeiling(t *testing.T) {
	t.Parallel()

	server := newIssueSearchServer(t, issueSearchServerConfig{total: 1500})
	client := newSearchListClient(t, server)
	walk := walkListSearch(t, client, 20)

	if len(walk.ids) != 1000 {
		t.Fatalf("retrieved %d issues, want 1,000 (Search API ceiling)", len(walk.ids))
	}
	if len(walk.requestedPages) != 50 {
		t.Fatalf("requested %d pages, want 50", len(walk.requestedPages))
	}
	if !walk.everyTruncated {
		t.Fatal("Truncated was false on at least one page while total_count was 1,500")
	}
	if walk.lastToken != "" || !walk.exhausted {
		t.Fatalf("final exhausted=%v token=%q", walk.exhausted, walk.lastToken)
	}
	if walk.reportedTotal != 1500 {
		t.Fatalf("reported total = %d, want 1,500", walk.reportedTotal)
	}
	for _, page := range walk.requestedPages {
		if page > 50 {
			t.Fatalf("requested page %d past the readable window (page 50 is last for per_page=20)", page)
		}
	}
}

func TestRegressionListResultSetSmallerThanOnePage(t *testing.T) {
	t.Parallel()

	server := newIssueSearchServer(t, issueSearchServerConfig{total: 7})
	client := newSearchListClient(t, server)
	page, err := client.List(context.Background(), gia.ListOptions{
		PageOptions: gia.PageOptions{PageSize: 20},
		TitleMatch:  "needle",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 7 || page.NextToken != "" || !page.Exhausted || page.Truncated {
		t.Fatalf("page = items:%d token:%q exhausted:%v truncated:%v", len(page.Items), page.NextToken, page.Exhausted, page.Truncated)
	}
	for i, item := range page.Items {
		if item.ID != int64(i+1) {
			t.Fatalf("id %d = %d", i, item.ID)
		}
	}
}

func TestRegressionListEmptyResultSet(t *testing.T) {
	t.Parallel()

	server := newIssueSearchServer(t, issueSearchServerConfig{total: 0})
	client := newSearchListClient(t, server)
	page, err := client.List(context.Background(), gia.ListOptions{
		PageOptions: gia.PageOptions{PageSize: 20},
		TitleMatch:  "needle",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 0 || page.NextToken != "" || !page.Exhausted || page.Truncated {
		t.Fatalf("page = items:%d token:%q exhausted:%v truncated:%v", len(page.Items), page.NextToken, page.Exhausted, page.Truncated)
	}
	if page.ReportedTotal == nil || *page.ReportedTotal != 0 {
		t.Fatalf("reported total = %v", page.ReportedTotal)
	}
}

func TestRegressionListFinalPageWithoutLinkHeader(t *testing.T) {
	t.Parallel()

	server := newIssueSearchServer(t, issueSearchServerConfig{total: 40, omitLink: true})
	client := newSearchListClient(t, server)

	first, err := client.List(context.Background(), gia.ListOptions{
		PageOptions: gia.PageOptions{PageSize: 20},
		TitleMatch:  "needle",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.NextToken == "" {
		t.Fatal("expected a continuation token on page 1 without relying on Link alone")
	}

	second, err := client.List(context.Background(), gia.ListOptions{
		PageOptions: gia.PageOptions{PageSize: 20, PageToken: first.NextToken},
		TitleMatch:  "needle",
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.NextToken != "" || !second.Exhausted || second.Truncated || len(second.Items) != 20 {
		t.Fatalf("final page items:%d token:%q exhausted:%v", len(second.Items), second.NextToken, second.Exhausted)
	}
	if second.Items[0].ID != 21 || second.Items[19].ID != 40 {
		t.Fatalf("final IDs = %d..%d", second.Items[0].ID, second.Items[19].ID)
	}
}

func TestRegressionListMissingLinkHeaderStillPaginates(t *testing.T) {
	t.Parallel()

	server := newIssueSearchServer(t, issueSearchServerConfig{total: 500, omitLink: true})
	client := newSearchListClient(t, server)
	walk := walkListSearch(t, client, 20)

	assertListIDs(t, walk.ids, 1, 500)
	if walk.anyTruncated || walk.lastToken != "" || len(walk.requestedPages) != 25 {
		t.Fatalf("truncated=%v token=%q pages=%d", walk.anyTruncated, walk.lastToken, len(walk.requestedPages))
	}
}

func TestRegressionListInvalidContinuationToken(t *testing.T) {
	t.Parallel()

	server := newIssueSearchServer(t, issueSearchServerConfig{total: 10})
	client := newSearchListClient(t, server)
	_, err := client.List(context.Background(), gia.ListOptions{
		PageOptions: gia.PageOptions{PageSize: 20, PageToken: "not-a-valid-token"},
		TitleMatch:  "needle",
	})
	if err == nil {
		t.Fatal("expected error for invalid page token")
	}
	var apiErr *gia.Error
	if !errors.As(err, &apiErr) || apiErr.Code != gia.CodeInvalidArgument {
		t.Fatalf("error = %v", err)
	}
}

func TestRegressionListWrongKindContinuationToken(t *testing.T) {
	t.Parallel()

	server := newIssueSearchServer(t, issueSearchServerConfig{total: 10})
	client := newSearchListClient(t, server)
	wrongKind := base64.RawURLEncoding.EncodeToString([]byte("ids:2"))
	_, err := client.List(context.Background(), gia.ListOptions{
		PageOptions: gia.PageOptions{PageSize: 20, PageToken: wrongKind},
		TitleMatch:  "needle",
	})
	if err == nil {
		t.Fatal("expected error for token from another operation")
	}
	var apiErr *gia.Error
	if !errors.As(err, &apiErr) || apiErr.Code != gia.CodeInvalidArgument {
		t.Fatalf("error = %v", err)
	}
}

func TestRegressionListResumeNear1000ResultBoundary(t *testing.T) {
	t.Parallel()

	server := newIssueSearchServer(t, issueSearchServerConfig{total: 1200})
	client := newSearchListClient(t, server)

	full := walkListSearch(t, client, 20)
	assertListIDs(t, full.ids, 1, 1000)

	token := encodeSearchPageToken(49)
	var resumed []int64
	seen := map[int64]struct{}{}
	for step := 0; step < 3; step++ {
		page, err := client.List(context.Background(), gia.ListOptions{
			PageOptions: gia.PageOptions{PageSize: 20, PageToken: token},
			TitleMatch:  "needle",
		})
		if err != nil {
			t.Fatal(err)
		}
		if page.ReportedTotal == nil || *page.ReportedTotal != 1200 {
			t.Fatalf("reported total = %v", page.ReportedTotal)
		}
		if !page.Truncated {
			t.Fatal("expected Truncated while total_count exceeds the ceiling")
		}
		for _, item := range page.Items {
			if _, exists := seen[item.ID]; exists {
				t.Fatalf("duplicate issue %d after resume", item.ID)
			}
			seen[item.ID] = struct{}{}
			resumed = append(resumed, item.ID)
		}
		if page.NextToken == "" {
			if step != 1 {
				t.Fatalf("stopped after %d resumed pages, want 2 (pages 49 and 50)", step+1)
			}
			if !page.Exhausted {
				t.Fatal("final resumed page must be exhausted")
			}
			break
		}
		nextPage, err := decodeSearchTokenPage(page.NextToken)
		if err != nil {
			t.Fatal(err)
		}
		if step == 0 && nextPage != 50 {
			t.Fatalf("page 49 token targets page %d, want 50", nextPage)
		}
		token = page.NextToken
	}
	assertListIDs(t, resumed, 961, 1000)
	tail := full.ids[len(full.ids)-40:]
	if len(resumed) != len(tail) {
		t.Fatalf("resumed %d issues, full-walk tail has %d", len(resumed), len(tail))
	}
	for i := range tail {
		if resumed[i] != tail[i] {
			t.Fatalf("resumed[%d]=%d full[%d]=%d", i, resumed[i], i, tail[i])
		}
	}
}

func TestRegressionListTokenBeyondLastPageReturnsEmptyWithoutNext(t *testing.T) {
	t.Parallel()

	server := newIssueSearchServer(t, issueSearchServerConfig{total: 500})
	client := newSearchListClient(t, server)
	page, err := client.List(context.Background(), gia.ListOptions{
		PageOptions: gia.PageOptions{PageSize: 20, PageToken: encodeSearchPageToken(26)},
		TitleMatch:  "needle",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 0 || page.NextToken != "" || !page.Exhausted || page.Truncated {
		t.Fatalf("items:%d token:%q exhausted:%v truncated:%v", len(page.Items), page.NextToken, page.Exhausted, page.Truncated)
	}
}

func assertListIDs(t *testing.T, ids []int64, first, last int64) {
	t.Helper()
	if int64(len(ids)) != last-first+1 {
		t.Fatalf("got %d ids, want %d through %d", len(ids), first, last)
	}
	for i, id := range ids {
		want := first + int64(i)
		if id != want {
			t.Fatalf("id %d = %d, want %d", i, id, want)
		}
	}
}

func encodeSearchPageToken(page int) string {
	return base64.RawURLEncoding.EncodeToString([]byte("search:" + strconv.Itoa(page)))
}

type listSearchWalk struct {
	ids              []int64
	requestedPages   []int
	tokenTargetPages []int
	rawTokens        []string
	lastToken        string
	exhausted        bool
	anyTruncated     bool
	everyTruncated   bool
	reportedTotal    int
}

func walkListSearch(t *testing.T, client *gia.Client, pageSize int) listSearchWalk {
	t.Helper()
	var walk listSearchWalk
	walk.everyTruncated = true
	token := ""
	seen := map[int64]struct{}{}

	for guard := 0; guard < 60; guard++ {
		page, err := client.List(context.Background(), gia.ListOptions{
			PageOptions: gia.PageOptions{PageSize: pageSize, PageToken: token},
			TitleMatch:  "needle",
		})
		if err != nil {
			t.Fatal(err)
		}
		if page.ReportedTotal == nil {
			t.Fatal("missing reported total")
		}
		if walk.reportedTotal == 0 {
			walk.reportedTotal = *page.ReportedTotal
		} else if *page.ReportedTotal != walk.reportedTotal {
			t.Fatalf("reported total changed from %d to %d", walk.reportedTotal, *page.ReportedTotal)
		}
		if page.Truncated {
			walk.anyTruncated = true
		} else {
			walk.everyTruncated = false
		}
		if page.Exhausted != (page.NextToken == "") {
			t.Fatalf("exhausted=%v next token=%q", page.Exhausted, page.NextToken)
		}

		requested, err := decodeSearchTokenPage(token)
		if err != nil {
			t.Fatal(err)
		}
		walk.requestedPages = append(walk.requestedPages, requested)

		for _, item := range page.Items {
			if _, exists := seen[item.ID]; exists {
				t.Fatalf("duplicate issue %d on page %d", item.ID, requested)
			}
			seen[item.ID] = struct{}{}
			walk.ids = append(walk.ids, item.ID)
		}

		walk.lastToken = page.NextToken
		walk.exhausted = page.Exhausted
		if page.NextToken == "" {
			return walk
		}
		next, err := decodeSearchTokenPage(page.NextToken)
		if err != nil {
			t.Fatal(err)
		}
		if next != requested+1 {
			t.Fatalf("page %d token targets page %d", requested, next)
		}
		walk.tokenTargetPages = append(walk.tokenTargetPages, next)
		walk.rawTokens = append(walk.rawTokens, page.NextToken)
		token = page.NextToken
	}
	t.Fatal("pagination did not end")
	return walk
}

func newSearchListClient(t *testing.T, server *httptest.Server) *gia.Client {
	t.Helper()
	client, err := gia.NewClient(gia.Options{
		Owner:       "octocat",
		Repository:  "example",
		Token:       "secret",
		HTTPClient:  server.Client(),
		RESTBaseURL: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

type issueSearchServerConfig struct {
	total    int
	omitLink bool
}

func newIssueSearchServer(t *testing.T, cfg issueSearchServerConfig) *httptest.Server {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := 1
		if value := r.URL.Query().Get("page"); value != "" {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				http.Error(w, "bad page", http.StatusBadRequest)
				return
			}
			page = parsed
		}
		perPage := 100
		if value := r.URL.Query().Get("per_page"); value != "" {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				http.Error(w, "bad per_page", http.StatusBadRequest)
				return
			}
			perPage = parsed
		}
		start := (page - 1) * perPage
		count := cfg.total - start
		if count > perPage {
			count = perPage
		}
		if count < 0 {
			count = 0
		}
		items := make([]map[string]any, 0, count)
		for i := 0; i < count; i++ {
			items = append(items, map[string]any{
				"number": start + i + 1,
				"title":  "issue",
				"body":   "body",
				"state":  "open",
			})
		}
		if !cfg.omitLink && page*perPage < cfg.total {
			w.Header().Set("Link", `<`+server.URL+`/search/issues?page=`+strconv.Itoa(page+1)+`&per_page=`+strconv.Itoa(perPage)+`>; rel="next"`)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_count":        cfg.total,
			"incomplete_results": false,
			"items":              items,
		})
	}))
	t.Cleanup(server.Close)
	return server
}

func decodeSearchTokenPage(token string) (int, error) {
	if token == "" {
		return 1, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, err
	}
	const prefix = "search:"
	if !strings.HasPrefix(string(decoded), prefix) {
		return 0, errors.New("not a search token")
	}
	return strconv.Atoi(strings.TrimPrefix(string(decoded), prefix))
}
