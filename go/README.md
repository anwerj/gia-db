# Go SDK

The [GiaDB v0.1](../db/SPEC.md) SDK for Go. Requires Go 1.26 or later. The module path is `github.com/anwerj/gia-db/go`. This repository has no Go release tag yet.

The caller supplies the Go context. This example uses `context.Background()`.

```go
import (
    "context"
    "log"
    "os"

    "github.com/anwerj/gia-db/go"
)

ctx := context.Background()

client, err := gia.NewClient(gia.Options{
    Owner:      "octocat",
    Repository: "knowledge",
    Token:      os.Getenv("GITHUB_TOKEN"),
})
if err != nil {
    log.Fatal(err)
}

// Create a document.
resource, _, err := client.Create(ctx, gia.CreateInput{
    Title:  "Concise technical tone",
    Body:   "Prefer direct language.",
    Labels: []string{"tone"},
})
if err != nil {
    log.Fatal(err)
}

// Read one document. Get does not load comments.
got, _, err := client.Get(ctx, resource.ID, gia.GetOptions{})
if err != nil {
    log.Fatal(err)
}

// List active documents with a label.
page, err := client.List(ctx, gia.ListOptions{
    PageOptions: gia.PageOptions{PageSize: 20},
    AllLabels:   []string{"tone"},
    State:       gia.StateFilterActive,
})
if err != nil {
    log.Fatal(err)
}
log.Println(len(page.Items), page.NextToken)

// Update the document. Nil fields are left unchanged.
title := "Direct technical tone"
updated, _, err := client.Update(ctx, got.ID, gia.UpdateInput{
    Title: &title,
})
if err != nil {
    log.Fatal(err)
}

// Append a comment, then page through comments.
_, _, err = client.NewSection(ctx, updated.ID, "Additional guidance.")
if err != nil {
    log.Fatal(err)
}
sections, err := client.ListSections(ctx, updated.ID, gia.PageOptions{PageSize: 20})
if err != nil {
    log.Fatal(err)
}
log.Println(len(sections.Items), sections.NextToken)

// Close the document, then open it again.
forgotten, _, err := client.Forget(ctx, updated.ID)
if err != nil {
    log.Fatal(err)
}
_, _, err = client.Reopen(ctx, forgotten.ID)
if err != nil {
    log.Fatal(err)
}
```

The SDK uses GitHub REST for issue reads, writes, comments, and search. It uses
a fixed GraphQL query for issue-only repository enumeration. GitHub remains the
source of truth; no local persistence or cache is required.

Important behavior:

- Resource IDs are repository-scoped issue numbers.
- Every non-pull-request issue in the repository is a GiaDB resource.
- The SDK passes issue and comment bodies through unchanged, without injecting GiaDB metadata or rewriting content.
- `Get` does not load comments. Use paginated `ListSections`.
- Paged methods return opaque continuation tokens.
- `List` is the only resource collection/query method. It supports IDs, title
  matching, state, labels, timestamps, sorting, and pagination.
- Search-backed `List` calls expose incomplete and 1,000-result truncation
  indicators.
- The SDK does not retry, pace, or limit concurrency. Throttled responses are returned as `rate_limited` with the GitHub rate-limit details.
- An ambiguous mutation transport failure is `outcome_unknown`.
- Shared issue field updates are last-write-wins.

Callers may supply an authenticated `http.Client` instead of `Token`. GitHub
App installation-token acquisition is intentionally outside the SDK.

The example above is a real method sequence for this module. It is not a complete program.

## Tests

From this directory:

```sh
go test -race ./...
```

Live tests call GitHub and are excluded unless `-tags=live` is set. See [`CONTRIBUTING.md`](../CONTRIBUTING.md).

## Release status

Not released from this repository. A release tag has the form `go/vX.Y.Z`. See [`RELEASING.md`](../RELEASING.md).
