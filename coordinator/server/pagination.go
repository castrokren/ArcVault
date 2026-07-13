package server

import (
	"math"
	"net/http"
	"strconv"
)

const (
	DefaultLimit = 25
	MaxLimit     = 100
	MaxPage      = 10000
)

// PaginationParams holds the parsed page and limit values from query params.
// Defaults: DefaultLimit. Caps: MaxLimit (limit), MaxPage (page).
type PaginationParams struct {
	Page  int // 1-indexed, min 1, max MaxPage
	Limit int // default DefaultLimit, max MaxLimit
}

// PaginatedResponse is the standard envelope returned by paginated list endpoints.
type PaginatedResponse struct {
	Data  any `json:"data"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Pages int `json:"pages"`
	Limit int `json:"limit"`
}

// ParsePagination reads ?page= and ?limit= from the request query string and
// applies safe defaults:
//   - page < 1        → 1
//   - page > MaxPage  → MaxPage
//   - limit < 1       → DefaultLimit
//   - limit > MaxLimit → MaxLimit
//   - non-numeric     → defaults (page=1, limit=DefaultLimit)
func ParsePagination(r *http.Request) PaginationParams {
	q := r.URL.Query()

	page := 1
	if raw := q.Get("page"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			page = v
		}
	}
	if page < 1 {
		page = 1
	}
	if page > MaxPage {
		page = MaxPage
	}

	limit := DefaultLimit
	if raw := q.Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	if limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	return PaginationParams{Page: page, Limit: limit}
}

// NewPaginatedResponse assembles a PaginatedResponse.
// pages = ceil(total / limit). If limit is 0, pages is 0.
func NewPaginatedResponse(data any, total, page, limit int) PaginatedResponse {
	pages := 0
	if limit > 0 {
		pages = int(math.Ceil(float64(total) / float64(limit)))
	}
	return PaginatedResponse{
		Data:  data,
		Total: total,
		Page:  page,
		Pages: pages,
		Limit: limit,
	}
}
