package errors

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// Exit codes. See document/errors.md for the full convention.
//
// 0       Success
// 1       Unknown — uncategorized fallback, should not appear in practice
// 2       Usage — unknown subcommand / unknown flag / cobra usage error
// 3       InvalidArgument — semantic param error: missing required, bad value, file not found
// 4       Config — local config / disk problem
// 5       Auth — not logged in or token invalid (HTTP 401 / backend not-logged-in)
// 6       Permission — logged in but lacks permission (HTTP 403 / membership level)
// 7       QuotaExhausted — out of quota / credits (HTTP 402 / backend quota code)
// 8       NotFound — resource missing: task-id / file-key / record-id (HTTP 404)
// 9       Conflict — state conflict: task already running / completed (HTTP 409)
// 10      RateLimited — throttled (HTTP 429)
// 11      Network — connect/DNS/TLS/read failure
// 12      Timeout — client timeout
// 13      ServerError — HTTP 5xx
// 14      TaskFailed — async task reached terminal failed state
// 20      Internal — CLI bug: panic, marshal failure, unreachable branch
const (
	ExitSuccess         = 0
	ExitUnknown         = 1
	ExitUsage           = 2
	ExitInvalidArgument = 3
	ExitConfig          = 4
	ExitAuth            = 5
	ExitPermission      = 6
	ExitQuotaExhausted  = 7
	ExitNotFound        = 8
	ExitConflict        = 9
	ExitRateLimited     = 10
	ExitNetwork         = 11
	ExitTimeout         = 12
	ExitServerError     = 13
	ExitTaskFailed      = 14
	ExitInternal        = 20
)

// Legacy aliases. Kept so files still compiling against the old names build,
// but every call site is being migrated to the new constants.
const (
	ExitAPIError    = ExitUnknown
	ExitAuthError   = ExitAuth
	ExitParamError  = ExitInvalidArgument
	ExitConfigError = ExitConfig
	ExitNetError    = ExitNetwork
)

// Type strings used in the JSON error envelope. Stable contract for agents.
const (
	TypeUnknown         = "unknown"
	TypeUsage           = "usage"
	TypeInvalidArgument = "invalid_argument"
	TypeConfig          = "config"
	TypeAuth            = "auth"
	TypePermission      = "permission"
	TypeQuotaExhausted  = "quota_exhausted"
	TypeNotFound        = "not_found"
	TypeConflict        = "conflict"
	TypeRateLimited     = "rate_limited"
	TypeNetwork         = "network"
	TypeTimeout         = "timeout"
	TypeServerError     = "server_error"
	TypeTaskFailed      = "task_failed"
	TypeInternal        = "internal"
)

// CLIError is the structured error every command should bubble up. The fields
// match the JSON envelope written to stderr in --format json mode.
//
// Details carries optional context-specific key/value pairs that an agent
// might want to act on without parsing the human-readable Message — for
// example a query_key on a tools poll timeout. Keys should be snake_case
// and stable; values are stringified for transport.
type CLIError struct {
	Type        string            `json:"type"`
	Message     string            `json:"message"`
	Hint        string            `json:"hint,omitempty"`
	Code        int               `json:"exit_code"`
	Retryable   bool              `json:"retryable"`
	HTTPStatus  int               `json:"http_status,omitempty"`
	BackendCode string            `json:"backend_code,omitempty"`
	Details     map[string]string `json:"details,omitempty"`
}

func (e *CLIError) Error() string {
	return e.Message
}

// WithHTTP attaches an HTTP status code, returning the same error for chaining.
func (e *CLIError) WithHTTP(status int) *CLIError {
	e.HTTPStatus = status
	return e
}

// WithBackendCode attaches the backend business code.
func (e *CLIError) WithBackendCode(code string) *CLIError {
	e.BackendCode = code
	return e
}

// WithDetail attaches a structured key/value pair so agents can recover
// context (e.g. a query_key on a poll timeout) without parsing Message.
func (e *CLIError) WithDetail(key, value string) *CLIError {
	if e.Details == nil {
		e.Details = make(map[string]string, 1)
	}
	e.Details[key] = value
	return e
}

// New is the low-level constructor. Prefer the typed helpers below.
func New(code int, errType, message, hint string, retryable bool) *CLIError {
	return &CLIError{
		Type:      errType,
		Message:   message,
		Hint:      hint,
		Code:      code,
		Retryable: retryable,
	}
}

func UsageError(message, hint string) *CLIError {
	return New(ExitUsage, TypeUsage, message, hint, false)
}

func ParamError(message, hint string) *CLIError {
	return New(ExitInvalidArgument, TypeInvalidArgument, message, hint, false)
}

func ConfigError(message, hint string) *CLIError {
	return New(ExitConfig, TypeConfig, message, hint, false)
}

func AuthError(message, hint string) *CLIError {
	return New(ExitAuth, TypeAuth, message, hint, false)
}

func PermissionError(message, hint string) *CLIError {
	return New(ExitPermission, TypePermission, message, hint, false)
}

func QuotaError(message, hint string) *CLIError {
	return New(ExitQuotaExhausted, TypeQuotaExhausted, message, hint, false)
}

func NotFoundError(message, hint string) *CLIError {
	return New(ExitNotFound, TypeNotFound, message, hint, false)
}

func ConflictError(message, hint string) *CLIError {
	return New(ExitConflict, TypeConflict, message, hint, false)
}

func RateLimitedError(message, hint string) *CLIError {
	return New(ExitRateLimited, TypeRateLimited, message, hint, true)
}

func NetError(message, hint string) *CLIError {
	return New(ExitNetwork, TypeNetwork, message, hint, true)
}

func TimeoutError(message, hint string) *CLIError {
	return New(ExitTimeout, TypeTimeout, message, hint, true)
}

func ServerError(message, hint string) *CLIError {
	return New(ExitServerError, TypeServerError, message, hint, true)
}

func TaskFailedError(message, hint string) *CLIError {
	return New(ExitTaskFailed, TypeTaskFailed, message, hint, false)
}

// APIError is the legacy constructor. New code should pick a more specific
// helper above. Kept so the existing client.go and a few cmd files still build
// while we migrate.
func APIError(message, hint string) *CLIError {
	return New(ExitUnknown, TypeUnknown, message, hint, false)
}

// jsonMode is set by the root command when --format json is in effect.
// errors.Handle reads it to decide whether to write a JSON envelope or the
// human-readable form.
var jsonMode bool

// SetJSONMode toggles JSON envelope output for errors. Called from the root
// command's PersistentPreRun once flags are parsed.
func SetJSONMode(on bool) {
	jsonMode = on
}

// Handle terminates the process. err == nil is a no-op. Non-CLIError values
// are wrapped as Internal so the agent always gets a structured envelope.
func Handle(err error) {
	if err == nil {
		return
	}
	cliErr, ok := err.(*CLIError)
	if !ok {
		cliErr = New(ExitInternal, TypeInternal, err.Error(), "", false)
	}

	if jsonMode {
		writeJSON(cliErr)
	} else {
		writePretty(cliErr)
	}
	os.Exit(cliErr.Code)
}

func writeJSON(e *CLIError) {
	envelope := map[string]interface{}{
		"ok":        false,
		"exit_code": e.Code,
		"type":      e.Type,
		"message":   e.Message,
		"retryable": e.Retryable,
	}
	if e.Hint != "" {
		envelope["hint"] = e.Hint
	}
	if e.HTTPStatus != 0 {
		envelope["http_status"] = e.HTTPStatus
	}
	if e.BackendCode != "" {
		envelope["backend_code"] = e.BackendCode
	}
	if len(e.Details) > 0 {
		envelope["details"] = e.Details
	}
	out, mErr := json.Marshal(envelope)
	if mErr != nil {
		fmt.Fprintf(os.Stderr, `{"ok":false,"exit_code":20,"type":"internal","message":%q,"retryable":false}`+"\n", e.Message)
		return
	}
	fmt.Fprintln(os.Stderr, string(out))
}

func writePretty(e *CLIError) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", e.Message)
	if e.Hint != "" {
		fmt.Fprintf(os.Stderr, "Hint: %s\n", e.Hint)
	}
	// Machine-readable trailer so agents that forgot --format json can still
	// parse the structured signal with a single line regex.
	trailer := fmt.Sprintf("# pdf-cli error type=%s code=%d retryable=%t",
		e.Type, e.Code, e.Retryable)
	if e.HTTPStatus != 0 {
		trailer += fmt.Sprintf(" http_status=%d", e.HTTPStatus)
	}
	if e.BackendCode != "" {
		trailer += fmt.Sprintf(" backend_code=%s", e.BackendCode)
	}
	// Detail keys are emitted in sorted order for stable parsing.
	if len(e.Details) > 0 {
		keys := make([]string, 0, len(e.Details))
		for k := range e.Details {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			trailer += fmt.Sprintf(" %s=%s", k, e.Details[k])
		}
	}
	fmt.Fprintln(os.Stderr, trailer)
}
