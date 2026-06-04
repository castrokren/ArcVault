package server

// PaginationMeta provides pagination metadata for list responses
type PaginationMeta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
	Pages int `json:"pages"`
}

// ErrorResponse defines the standard error response format
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse for simple success messages
type MessageResponse struct {
	Message string `json:"message"`
}
