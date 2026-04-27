package errors

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestEnvelopeWithDetails(t *testing.T) {
	e := TaskFailedError("翻译任务终止: status=fail reason=ocr_error", "查看 history").
		WithDetail("task_id", "abc-123").
		WithDetail("status", "fail").
		WithDetail("fail_reason", "ocr_error")

	out := captureStderr(t, true, func() {
		writeJSON(e)
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("envelope is not valid JSON: %v\nraw: %s", err, out)
	}

	if envelope["exit_code"].(float64) != 14 {
		t.Errorf("exit_code: got %v, want 14", envelope["exit_code"])
	}
	if envelope["type"] != "task_failed" {
		t.Errorf("type: got %v, want task_failed", envelope["type"])
	}
	details, ok := envelope["details"].(map[string]interface{})
	if !ok {
		t.Fatalf("details missing or wrong type: %v", envelope["details"])
	}
	for k, want := range map[string]string{
		"task_id":     "abc-123",
		"status":      "fail",
		"fail_reason": "ocr_error",
	} {
		if details[k] != want {
			t.Errorf("details[%q]: got %v, want %s", k, details[k], want)
		}
	}
}

func TestWithDetailLazyAlloc(t *testing.T) {
	e := AuthError("token 失效", "重新登录")
	if e.Details != nil {
		t.Fatalf("Details should be nil before WithDetail")
	}
	e.WithDetail("k", "v")
	if e.Details["k"] != "v" {
		t.Fatalf("WithDetail did not set the key")
	}
}

func TestPrettyTrailerSortedDeterministic(t *testing.T) {
	// Detail keys must be emitted in sorted order so agents can split on
	// space and parse without surprises. Insertion order is intentionally
	// scrambled here.
	e := TimeoutError("超时", "see hint").
		WithDetail("zzz", "last").
		WithDetail("aaa", "first").
		WithDetail("mmm", "middle").
		WithHTTP(408).
		WithBackendCode("1234")

	out := captureStderr(t, false, func() {
		writePretty(e)
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected 3 stderr lines, got %d:\n%s", len(lines), out)
	}
	trailer := lines[len(lines)-1]
	want := "# pdf-cli error type=timeout code=12 retryable=true http_status=408 backend_code=1234 aaa=first mmm=middle zzz=last"
	if trailer != want {
		t.Errorf("trailer mismatch:\n  got:  %s\n  want: %s", trailer, want)
	}
}

// captureStderr runs fn with os.Stderr redirected to a pipe and returns the
// captured output. jsonMode is set/restored around the call so writeJSON and
// writePretty pick up the right toggle.
func captureStderr(t *testing.T, json bool, fn func()) string {
	t.Helper()
	prev := jsonMode
	jsonMode = json
	defer func() { jsonMode = prev }()

	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fn()
	w.Close()
	<-done
	return buf.String()
}
