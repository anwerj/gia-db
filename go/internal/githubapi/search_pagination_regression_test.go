package githubapi

import (
	"context"
	"errors"
	"testing"
)

// Regression tests for ListMatching Search API pagination.
// See search_pagination_regression_test.go in package gia for the public List walk.

func TestRegressionListMatchingPerPage20FiveHundredTwentyFivePages(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 500, link: linkSequential})
	client := newSearchClient(t, server)
	walk := walkSearch(t, client, 20)

	if len(walk.requests) != 25 {
		t.Fatalf("requested %d pages, want 25", len(walk.requests))
	}
	assertIDs(t, walk.ids, 1, 500)
	if walk.truncated || walk.lastToken != "" {
		t.Fatalf("final truncated=%v token=%q", walk.truncated, walk.lastToken)
	}
}

func TestRegressionListMatchingPageTenStillEmitsPageElevenToken(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 500, link: linkSequential})
	client := newSearchClient(t, server)

	token := ""
	for pageNum := 1; pageNum <= 10; pageNum++ {
		page, err := client.ListMatching(context.Background(), ListMatchInput{
			Text:      "needle",
			PageSize:  20,
			PageToken: token,
		})
		if err != nil {
			t.Fatal(err)
		}
		if pageNum == 10 && page.NextToken == "" {
			t.Fatal("page 10 returned no token; old NextPage<=10 logic stopped pagination at 200 results")
		}
		token = page.NextToken
	}
	next, err := decodePageToken(token, "search")
	if err != nil {
		t.Fatal(err)
	}
	if next != 11 {
		t.Fatalf("page 10 token targets page %d, want 11", next)
	}
}

func TestRegressionListMatchingContinuationTokensAreSequential(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 500, link: linkSequential})
	client := newSearchClient(t, server)
	walk := walkSearch(t, client, 20)

	if len(walk.tokenPages) != 24 {
		t.Fatalf("token count = %d, want 24", len(walk.tokenPages))
	}
	for i, target := range walk.tokenPages {
		if target != i+2 {
			t.Fatalf("token %d -> page %d, want %d", i, target, i+2)
		}
	}
}

func TestRegressionListMatchingExact1000ResultsFiftyPages(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 1000, link: linkSequential})
	client := newSearchClient(t, server)
	walk := walkSearch(t, client, 20)

	if len(walk.requests) != 50 {
		t.Fatalf("requested %d pages, want 50", len(walk.requests))
	}
	assertIDs(t, walk.ids, 1, 1000)
	if walk.truncated || walk.lastToken != "" {
		t.Fatalf("truncated=%v token=%q", walk.truncated, walk.lastToken)
	}
}

func TestRegressionListMatchingInvalidContinuationToken(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 10, link: linkSequential})
	client := newSearchClient(t, server)
	_, err := client.ListMatching(context.Background(), ListMatchInput{
		Text:      "needle",
		PageSize:  20,
		PageToken: "not-valid",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Code != "invalid_argument" {
		t.Fatalf("error = %v", err)
	}
}

func TestRegressionListMatchingResumeNear1000Boundary(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 1200, link: linkSequential})
	client := newSearchClient(t, server)
	full := walkSearch(t, client, 20)
	assertIDs(t, full.ids, 1, 1000)

	token := encodePageToken("search", 49)
	var resumed []int64
	seen := map[int64]struct{}{}
	for step := 0; step < 3; step++ {
		page, err := client.ListMatching(context.Background(), ListMatchInput{
			Text:      "needle",
			PageSize:  20,
			PageToken: token,
		})
		if err != nil {
			t.Fatal(err)
		}
		if page.Total != 1200 || !page.Truncated {
			t.Fatalf("total=%d truncated=%v", page.Total, page.Truncated)
		}
		for _, item := range page.Items {
			if _, exists := seen[item.Number]; exists {
				t.Fatalf("duplicate %d", item.Number)
			}
			seen[item.Number] = struct{}{}
			resumed = append(resumed, item.Number)
		}
		if page.NextToken == "" {
			if step != 1 {
				t.Fatalf("stopped after %d pages", step+1)
			}
			break
		}
		next, err := decodePageToken(page.NextToken, "search")
		if err != nil {
			t.Fatal(err)
		}
		if step == 0 && next != 50 {
			t.Fatalf("next page = %d", next)
		}
		token = page.NextToken
	}
	assertIDs(t, resumed, 961, 1000)
}

func TestRegressionListMatchingCeilingWithTotalAbove1000(t *testing.T) {
	t.Parallel()

	server := newSearchServer(t, searchConfig{total: 1500, link: linkSequential})
	client := newSearchClient(t, server)
	walk := walkSearch(t, client, 20)

	assertIDs(t, walk.ids, 1, 1000)
	if len(walk.requests) != 50 || walk.anyPageAbove(50) {
		t.Fatalf("pages = %v", walk.requests)
	}
	if !walk.everyTruncated || walk.lastToken != "" {
		t.Fatalf("truncated=%v token=%q", walk.everyTruncated, walk.lastToken)
	}
}
