package response

// Error codes for the application.
// Error codes are organized by category:
// 0: Success
// 1xxx: Client errors (invalid input)
// 2xxx: Authentication errors
// 3xxx: Authorization errors
// 4xxx: Resource not found
// 5xxx: Server errors
const (
	// SuccessCode indicates a successful operation.
	SuccessCode = 0

	// Client error codes (1xxx)
	ErrInvalidParam      = 1001
	ErrMissingParam      = 1002
	ErrInvalidFormat     = 1003
	ErrInvalidRange      = 1004
	ErrDuplicateEntry    = 1005
	ErrRateLimited       = 1006
	ErrRequestTooLarge   = 1007
	ErrInvalidState      = 1008
	ErrValidationFailed  = 1009

	// Authentication error codes (2xxx)
	ErrUnauthenticated   = 2001
	ErrSessionExpired    = 2002

	// Authorization error codes (3xxx)
	ErrForbidden         = 3001
	ErrRoleRequired      = 3002

	// Resource error codes (4xxx)
	ErrResourceNotFound  = 4001
	ErrLogNotFound       = 4002
	ErrRuleNotFound      = 4003
	ErrAlertNotFound     = 4004
	ErrStoreNotFound     = 4005

	// Server error codes (5xxx)
	ErrInternal          = 5001
	ErrDatabaseError     = 5002
	ErrStorageFull       = 5003
	ErrQueueFull         = 5004
	ErrSchedulerError    = 5005
	ErrSerialization     = 5006
	ErrIOError           = 5007
)

// ErrorMessages maps error codes to their default messages.
var ErrorMessages = map[int]string{
	SuccessCode:         "success",
	ErrInvalidParam:     "invalid parameter",
	ErrMissingParam:     "missing required parameter",
	ErrInvalidFormat:    "invalid format",
	ErrInvalidRange:     "value out of allowed range",
	ErrDuplicateEntry:   "duplicate entry",
	ErrRateLimited:      "rate limit exceeded",
	ErrRequestTooLarge:  "request body too large",
	ErrInvalidState:     "invalid state transition",
	ErrValidationFailed: "validation failed",
	ErrUnauthenticated:  "authentication required",
	ErrSessionExpired:   "session expired",
	ErrForbidden:        "forbidden",
	ErrRoleRequired:     "role required",
	ErrResourceNotFound: "resource not found",
	ErrLogNotFound:      "log entry not found",
	ErrRuleNotFound:     "rule not found",
	ErrAlertNotFound:    "alert not found",
	ErrStoreNotFound:    "store not found",
	ErrInternal:         "internal server error",
	ErrDatabaseError:    "database error",
	ErrStorageFull:      "storage is full",
	ErrQueueFull:        "queue is full",
	ErrSchedulerError:   "scheduler error",
	ErrSerialization:    "serialization error",
	ErrIOError:          "io error",
}

// GetMessage returns the default message for an error code.
func GetMessage(code int) string {
	if msg, ok := ErrorMessages[code]; ok {
		return msg
	}
	return "unknown error"
}

// NewError creates a new error response with the given code.
func NewError(code int) *Response {
	return Error(code, GetMessage(code))
}

// NewErrorMsg creates a new error response with a custom message.
func NewErrorMsg(code int, message string) *Response {
	return Error(code, message)
}

// IsSuccess checks if a response indicates success.
func IsSuccess(r *Response) bool {
	return r.Code == SuccessCode
}

// IsClientError checks if a response indicates a client error.
func IsClientError(r *Response) bool {
	return r.Code >= 1001 && r.Code < 2000
}

// IsServerError checks if a response indicates a server error.
func IsServerError(r *Response) bool {
	return r.Code >= 5001 && r.Code < 6000
}
