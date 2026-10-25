package gia

import "time"

// StateFilter controls list lifecycle selection.
type StateFilter string

const (
	StateFilterDefault   StateFilter = ""
	StateFilterActive    StateFilter = "active"
	StateFilterForgotten StateFilter = "forgotten"
	StateFilterAll       StateFilter = "all"
)

type SortField string

const (
	SortCreated SortField = "created"
	SortUpdated SortField = "updated"
)

type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

type CreateInput struct {
	Title  string
	Body   string
	Labels []string
}

// UpdateInput is a patch. Nil fields are not changed. A body value replaces
// the stored issue body exactly. Labels replaces the complete label set.
type UpdateInput struct {
	Title  *string
	Body   *string
	Labels *[]string
}

type GetOptions struct {
	IfNoneMatch string
}

type PageOptions struct {
	PageSize  int
	PageToken string
}

type ListOptions struct {
	PageOptions
	IDs          []int64
	TitleMatch   string
	CreatedSince *time.Time
	UpdatedSince *time.Time
	State        StateFilter
	AllLabels    []string
	Sort         SortField
	Direction    SortDirection
}
