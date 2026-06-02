package server

import (
	"math"
	"net/http"
	"strconv"
)

// PaginationParams holds the parsed page and limit values from query params.
type PaginationParams struct {
	Page  int // 1-indexed, min 1
	Limit int // default 25, max 100
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
//   - limit < 1       → 25
//   - limit > 100     → 100
//   - non-numeric     → defaults (page=1, limit=25)
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

	limit := 25
	if raw := q.Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	if limit < 1 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
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
