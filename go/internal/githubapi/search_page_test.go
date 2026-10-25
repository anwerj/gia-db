package githubapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
)

func TestNextSearchPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		current    int
		pageSize   int
		itemCount  int
		headerNext int
		total      int
		totalKnown bool
		wantNext   int
		wantCut    bool
	}{
		{
			name:    "page size 100 stops before the unreadable eleventh page",
			current: 10, pageSize: 100, itemCount: 100, headerNext: 11,
			total: 1200, totalKnown: true, wantNext: 0, wantCut: true,
		},
		{
			name:    "page size 20 continues past page 10",
			current: 10, pageSize: 20, itemCount: 20, headerNext: 11,
			total: 500, totalKnown: true, wantNext: 11, wantCut: false,
		},
		{
			name:    "known total inside the window ends on the last page",
			current: 25, pageSize: 20, itemCount: 20, headerNext: 0,
			total: 500, totalKnown: true, wantNext: 0, wantCut: false,
		},
		{
			name:    "missing link continues while the total is unread",
			current: 1, pageSize: 20, itemCount: 20, headerNext: 0,
			total: 500, totalKnown: true, wantNext: 2, wantCut: false,
		},
		{
			name:    "missing total on a full page continues",
			current: 1, pageSize: 20, itemCount: 20, headerNext: 0,
			wantNext: 2, wantCut: false,
		},
		{
			name:    "missing total on a short page is the end",
			current: 1, pageSize: 20, itemCount: 5, headerNext: 0,
			wantNext: 0, wantCut: false,
		},
		{
			name:    "missing total at the window is truncated",
			current: 50, pageSize: 20, itemCount: 20, headerNext: 0,
			wantNext: 0, wantCut: true,
		},
		{
			name:    "short page at the window with a next link is truncated",
			current: 50, pageSize: 20, itemCount: 5, headerNext: 51,
			wantNext: 0, wantCut: true,
		},
		{
			name:    "exact window with a next link is complete",
			current: 50, pageSize: 20, itemCount: 20, headerNext: 51,
			total: 1000, totalKnown: true, wantNext: 0, wantCut: false,
		},
		{
			name:    "jumped next link resumes at the next page",
			current: 1, pageSize: 20, itemCount: 20, headerNext: 5,
			total: 100, totalKnown: true, wantNext: 2, wantCut: false,
		},
		{
			name:    "total above the window stays truncated while pages remain",
			current: 1, pageSize: 100, itemCount: 100, headerNext: 2,
			total: 1200, totalKnown: true, wantNext: 2, wantCut: true,
		},
		{
			name:    "empty page continues while the known total is unread",
			current: 2, pageSize: 20, itemCount: 0, headerNext: 0,
			total: 500, totalKnown: true, wantNext: 3, wantCut: false,
		},
		{
			name:    "empty page without a total does not invent a successor",
			current: 2, pageSize: 20, itemCount: 0, headerNext: 0,
			wantNext: 0, wantCut: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			next, truncated := nextSearchPage(
				test.current, test.pageSize, test.itemCount,
				test.headerNext, test.total, test.totalKnown,
			)
			if next != test.wantNext || truncated != test.wantCut {
				t.Fatalf("next=%d truncated=%v, want next=%d truncated=%v", next, truncated, test.wantNext, test.wantCut)
			}
		})
	}
}

func TestSearchPerPage20ReturnsAll500(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 500, link: linkSequential})
	client := newSearchClient(t, server)
	pages := walkSearch(t, client, 20)

	if got := len(pages.ids); got != 500 {
		t.Fatalf("retrieved %d issues, want 500", got)
	}
	assertIDs(t, pages.ids, 1, 500)
	if pages.requests[len(pages.requests)-1] != 25 || len(pages.requests) != 25 {
		t.Fatalf("requested pages = %v, want 1..25", pages.requests)
	}
	if pages.truncated || pages.lastToken != "" {
		t.Fatalf("last page truncated=%v token=%q", pages.truncated, pages.lastToken)
	}
	for i, tokenPage := range pages.tokenPages {
		if tokenPage != i+2 {
			t.Fatalf("token %d points at page %d, want %d", i, tokenPage, i+2)
		}
	}
	if len(pages.tokenPages) != 24 {
		t.Fatalf("continuation tokens = %d, want 24", len(pages.tokenPages))
	}
}

func TestSearchContinuationDoesNotSkipOrRepeat(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 60, link: linkSequential})
	client := newSearchClient(t, server)
	pages := walkSearch(t, client, 20)

	assertIDs(t, pages.ids, 1, 60)
	if len(pages.tokenPages) != 2 || pages.tokenPages[0] != 2 || pages.tokenPages[1] != 3 {
		t.Fatalf("token pages = %v, want [2 3]", pages.tokenPages)
	}
	if pages.lastToken != "" || pages.truncated {
		t.Fatalf("final page token=%q truncated=%v", pages.lastToken, pages.truncated)
	}
	if !sameInts(pages.requests, []int{1, 2, 3}) {
		t.Fatalf("requested pages = %v", pages.requests)
	}
}

func TestSearchReachesExact1000ResultCeiling(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 1000, link: linkSequential})
	client := newSearchClient(t, server)
	pages := walkSearch(t, client, 20)

	assertIDs(t, pages.ids, 1, 1000)
	if pages.truncated || pages.lastToken != "" {
		t.Fatalf("ceiling page truncated=%v token=%q", pages.truncated, pages.lastToken)
	}
	if len(pages.requests) != 50 || pages.anyPageAbove(50) {
		t.Fatalf("requested pages = %v", pages.requests)
	}
}

func TestSearchTotalAbove1000IsTruncated(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 1500, link: linkSequential})
	client := newSearchClient(t, server)
	pages := walkSearch(t, client, 20)

	assertIDs(t, pages.ids, 1, 1000)
	if !pages.truncated || pages.lastToken != "" {
		t.Fatalf("final truncated=%v token=%q", pages.truncated, pages.lastToken)
	}
	if !pages.everyTruncated {
		t.Fatal("a page inside an over-limit result set was reported as complete")
	}
	if pages.anyPageAbove(50) {
		t.Fatalf("requested a page past the window: %v", pages.requests)
	}
}

func TestSearchPageSize100StopsAtTenPages(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 1200, link: linkSequential})
	client := newSearchClient(t, server)
	pages := walkSearch(t, client, 100)

	assertIDs(t, pages.ids, 1, 1000)
	if !pages.everyTruncated || pages.lastToken != "" || pages.anyPageAbove(10) {
		t.Fatalf("truncated=%v token=%q pages=%v", pages.everyTruncated, pages.lastToken, pages.requests)
	}
}

func TestSearchFinalPageHasNoContinuationToken(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 40, link: linkSequential})
	client := newSearchClient(t, server)

	first, err := client.ListMatching(context.Background(), ListMatchInput{Text: "needle", PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if first.NextToken == "" || first.Truncated || len(first.Items) != 20 {
		t.Fatalf("first page = %+v", first)
	}
	second, err := client.ListMatching(context.Background(), ListMatchInput{
		Text: "needle", PageSize: 20, PageToken: first.NextToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.NextToken != "" || second.Truncated || len(second.Items) != 20 {
		t.Fatalf("final page = %+v", second)
	}
	if second.Items[0].Number != 21 || second.Items[19].Number != 40 {
		t.Fatalf("final IDs = %d..%d", second.Items[0].Number, second.Items[19].Number)
	}
}

func TestSearchMissingLinkStillReturnsRemainingResults(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 500, link: linkNone})
	client := newSearchClient(t, server)
	pages := walkSearch(t, client, 20)

	assertIDs(t, pages.ids, 1, 500)
	if pages.truncated || pages.lastToken != "" || len(pages.requests) != 25 {
		t.Fatalf("truncated=%v token=%q pages=%v", pages.truncated, pages.lastToken, pages.requests)
	}
}

func TestSearchMissingTotalOnAFullPageContinues(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 40, omitTotal: true, link: linkNone})
	client := newSearchClient(t, server)
	page, err := client.ListMatching(context.Background(), ListMatchInput{Text: "needle", PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	next, err := decodePageToken(page.NextToken, "search")
	if err != nil {
		t.Fatal(err)
	}
	if next != 2 || page.Truncated || page.Total != 0 {
		t.Fatalf("page token=%d truncated=%v total=%d", next, page.Truncated, page.Total)
	}
}

func TestSearchMissingTotalOnAShortPageEnds(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 5, omitTotal: true, link: linkNone})
	client := newSearchClient(t, server)
	page, err := client.ListMatching(context.Background(), ListMatchInput{Text: "needle", PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if page.NextToken != "" || page.Truncated || len(page.Items) != 5 {
		t.Fatalf("page items=%d token=%q truncated=%v", len(page.Items), page.NextToken, page.Truncated)
	}
}

func TestSearchMissingTotalAtCeilingIsTruncated(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 2000, omitTotal: true, link: linkNone})
	client := newSearchClient(t, server)
	pages := walkSearch(t, client, 20)

	assertIDs(t, pages.ids, 1, 1000)
	if !pages.truncated || pages.lastToken != "" || pages.anyPageAbove(50) {
		t.Fatalf("truncated=%v token=%q pages=%v", pages.truncated, pages.lastToken, pages.requests)
	}
}

func TestSearchUnexpectedNextPageDoesNotSkip(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 100, link: linkJump})
	client := newSearchClient(t, server)
	pages := walkSearch(t, client, 20)

	assertIDs(t, pages.ids, 1, 100)
	if !sameInts(pages.requests, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("requested pages = %v, want 1..5", pages.requests)
	}
	if pages.truncated || pages.lastToken != "" {
		t.Fatalf("truncated=%v token=%q", pages.truncated, pages.lastToken)
	}
}

func TestSearchEmptyPageDoesNotHideRemainingResults(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 60, link: linkNone, skipPage: 2})
	client := newSearchClient(t, server)
	pages := walkSearch(t, client, 20)

	if !sameInts(pages.requests, []int{1, 2, 3}) {
		t.Fatalf("requested pages = %v, want 1, 2, 3", pages.requests)
	}
	want := make([]int64, 0, 40)
	for id := int64(1); id <= 20; id++ {
		want = append(want, id)
	}
	for id := int64(41); id <= 60; id++ {
		want = append(want, id)
	}
	if len(pages.ids) != len(want) {
		t.Fatalf("ids = %v, want %v", pages.ids, want)
	}
	for i := range want {
		if pages.ids[i] != want[i] {
			t.Fatalf("ids = %v, want %v", pages.ids, want)
		}
	}
	if pages.truncated || pages.lastToken != "" {
		t.Fatalf("truncated=%v token=%q", pages.truncated, pages.lastToken)
	}
}

func TestSearchPullRequestDoesNotEndAFullPage(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 40, omitTotal: true, link: linkNone, pullFirst: true})
	client := newSearchClient(t, server)
	first, err := client.ListMatching(context.Background(), ListMatchInput{Text: "needle", PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	next, err := decodePageToken(first.NextToken, "search")
	if err != nil {
		t.Fatal(err)
	}
	if next != 2 || first.Truncated || len(first.Items) != 19 {
		t.Fatalf("first page items=%d token page=%d truncated=%v", len(first.Items), next, first.Truncated)
	}
	for _, item := range first.Items {
		if item.IsPR || item.Number == 1 {
			t.Fatalf("pull request was returned: %+v", item)
		}
	}
}

type searchLink int

const (
	linkSequential searchLink = iota
	linkNone
	linkJump
)

type searchConfig struct {
	total     int
	omitTotal bool
	link      searchLink
	pullFirst bool
	skipPage  int
}

func newSearchServer(t *testing.T, cfg searchConfig) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/issues" {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		requests++
		if requests > 80 {
			mu.Unlock()
			http.Error(w, "too many pages", http.StatusInternalServerError)
			return
		}
		mu.Unlock()

		page := queryInt(r, "page", 1)
		perPage := queryInt(r, "per_page", 100)
		if page < 1 || perPage < 1 {
			http.Error(w, "bad page", http.StatusBadRequest)
			return
		}
		start := (page - 1) * perPage
		count := cfg.total - start
		if cfg.skipPage == page {
			count = 0
		}
		if count < 0 {
			count = 0
		}
		if count > perPage {
			count = perPage
		}
		items := make([]map[string]any, 0, count)
		for i := 0; i < count; i++ {
			number := start + i + 1
			item := map[string]any{
				"number": number,
				"title":  "issue",
				"body":   "body",
				"state":  "open",
			}
			if cfg.pullFirst && page == 1 && i == 0 {
				item["pull_request"] = map[string]any{}
			}
			items = append(items, item)
		}
		payload := map[string]any{
			"incomplete_results": false,
			"items":              items,
		}
		if !cfg.omitTotal {
			payload["total_count"] = cfg.total
		}
		if next := cfg.nextPage(page, perPage); next > 0 {
			w.Header().Set("Link", `<`+serverURL(r)+`/search/issues?page=`+strconv.Itoa(next)+`&per_page=`+strconv.Itoa(perPage)+`>; rel="next"`)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Errorf("encode search page: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func (cfg searchConfig) nextPage(page, perPage int) int {
	switch cfg.link {
	case linkNone:
		return 0
	case linkJump:
		if page == 1 {
			return 5
		}
	}
	if page*perPage < cfg.total {
		return page + 1
	}
	return 0
}

func serverURL(r *http.Request) string {
	return "http://" + r.Host
}

func queryInt(r *http.Request, key string, fallback int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func newSearchClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client, err := New(Config{
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

type searchWalk struct {
	ids            []int64
	requests       []int
	tokenPages     []int
	lastToken      string
	truncated      bool
	everyTruncated bool
}

func (walk searchWalk) anyPageAbove(page int) bool {
	for _, requested := range walk.requests {
		if requested > page {
			return true
		}
	}
	return false
}

func walkSearch(t *testing.T, client *Client, pageSize int) searchWalk {
	t.Helper()
	var walk searchWalk
	walk.everyTruncated = true
	token := ""
	seen := map[int64]struct{}{}
	for guard := 0; guard < 60; guard++ {
		page, err := client.ListMatching(context.Background(), ListMatchInput{
			Text:      "needle",
			PageSize:  pageSize,
			PageToken: token,
		})
		if err != nil {
			t.Fatal(err)
		}
		requested, err := decodePageToken(token, "search")
		if err != nil {
			t.Fatal(err)
		}
		walk.requests = append(walk.requests, requested)
		if !page.Truncated {
			walk.everyTruncated = false
		}
		for _, item := range page.Items {
			if _, exists := seen[item.Number]; exists {
				t.Fatalf("duplicate issue %d on page %d", item.Number, requested)
			}
			seen[item.Number] = struct{}{}
			walk.ids = append(walk.ids, item.Number)
		}
		walk.truncated = page.Truncated
		walk.lastToken = page.NextToken
		if page.NextToken == "" {
			return walk
		}
		next, err := decodePageToken(page.NextToken, "search")
		if err != nil {
			t.Fatal(err)
		}
		if next != requested+1 {
			t.Fatalf("page %d continued at page %d", requested, next)
		}
		walk.tokenPages = append(walk.tokenPages, next)
		token = page.NextToken
	}
	t.Fatal("pagination did not end")
	return walk
}

func assertIDs(t *testing.T, ids []int64, first, last int64) {
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

func sameInts(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
