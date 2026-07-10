# Error Handling Pattern

This project uses a standardized error pattern across all layers (core, API, client, TUI).

## Core: `core/errors`

All application errors originate from the `core/errors` package.

### AppError

```go
type AppError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Err     error  `json:"-"`
}
```

- **Code**: Machine-readable identifier (e.g., `TASK_NOT_FOUND`, `VALIDATION_REQUIRED`)
- **Message**: Human-readable description
- **Err**: Optional wrapped error (accessible via `Unwrap()`)

### Sentinel Errors

```go
ErrTaskNotFound  // CODE: TASK_NOT_FOUND
ErrRunNotFound   // CODE: RUN_NOT_FOUND
ErrAgentNotFound // CODE: AGENT_NOT_FOUND
ErrConflict      // CODE: CONFLICT
ErrValidation    // CODE: VALIDATION
ErrInternal      // CODE: INTERNAL
ErrUnauthorized  // CODE: UNAUTHORIZED
```

### Helper Functions

```go
apperrors.Required("taskName") // → VALIDATION_REQUIRED, "taskName is required"
apperrors.StatusFor(err)      // → HTTP status code (400, 404, etc.)
apperrors.Is(err, target)     // → re-export of errors.Is
apperrors.As(err, &target)    // → re-export of errors.As
```

### Repositories

Repositories return `(nil, ErrXxxNotFound)` instead of `(nil, nil)` for not-found cases:

```go
// Before:
func (r *TaskRepository) GetByID(id string) (*Task, error) {
    // returned (nil, nil) for not found
}

// After:
func (r *TaskRepository) GetByID(id string) (*Task, error) {
    // returns (nil, apperrors.ErrTaskNotFound) for not found
}
```

## API: `cli/server`

### Response Format

All error responses use JSON with `code` and `message`:

```json
{"code": "TASK_NOT_FOUND", "message": "task not found"}
```

### Handler Pattern

Use `handleError(w, err)` instead of `writeError(w, status, msg)`:

```go
// Before:
writeError(w, http.StatusNotFound, "task not found")

// After:
handleError(w, apperrors.ErrTaskNotFound)
// or
handleError(w, apperrors.Required("taskName"))
// or for unexpected errors:
handleError(w, err) // logs internally, returns 500 INTERNAL
```

`handleError` automatically maps `AppError.Code` → HTTP status via `CodeToStatus`.

### Auth Middleware

Uses the same `handleError` for JSON error responses (not `http.Error` plain text).

## Client: `cli/client`

### APIError

```go
type APIError struct {
    StatusCode int
    Code       string
    Message    string
}
```

The client parses `{"code","message"}` responses into `APIError`. Callers can inspect:

```go
var apiErr *client.APIError
if errors.As(err, &apiErr) {
    // apiErr.Code == "TASK_NOT_FOUND"
    // apiErr.StatusCode == 404
}
```

## TUI: `cli/tui`

### Visual Differentiation

- **API errors** (client.APIError): Red ✗ prefix
- **Validation/local errors**: Yellow ⚠ prefix

```go
func formatError(err error) string {
    var apiErr *client.APIError
    if errors.As(err, &apiErr) {
        return errorStyle.Render("  ✗ " + err.Error())
    }
    return warnStyle.Render("  ⚠ " + err.Error())
}
```

## Code-to-Status Mapping

| Code                  | HTTP Status |
|-----------------------|-------------|
| TASK_NOT_FOUND        | 404         |
| RUN_NOT_FOUND         | 404         |
| AGENT_NOT_FOUND       | 404         |
| VALIDATION            | 400         |
| VALIDATION_REQUIRED   | 400         |
| CONFLICT              | 409         |
| INTERNAL              | 500         |
| UNAUTHORIZED          | 401         |

## Adding New Errors

1. Define a sentinel in `core/errors/errors.go`
2. Add the code-to-status mapping in `CodeToStatus`
3. Use `handleError(w, err)` in handlers
4. The client will automatically parse the structured response
