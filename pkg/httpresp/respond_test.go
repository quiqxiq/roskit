package httpresp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	apperrors "github.com/quiqxiq/roskit/pkg/errors"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func decodeEnvelope(t *testing.T, body []byte) Envelope {
	t.Helper()
	var env Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	return env
}

func TestSuccess_DataPresent_ErrorNil(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, http.StatusOK, gin.H{"name": "alice"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error != nil {
		t.Fatalf("error should be nil, got %+v", env.Error)
	}
	if env.Data == nil {
		t.Fatalf("data should be non-nil")
	}
}

func TestError_ShapeMatches(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, http.StatusBadRequest, CodeInvalidInput, "name is required")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Data != nil {
		t.Fatalf("data should be nil on error, got %+v", env.Data)
	}
	if env.Error == nil {
		t.Fatalf("error should be present")
	}
	if env.Error.Code != CodeInvalidInput || env.Error.Message != "name is required" {
		t.Fatalf("error fields wrong: %+v", env.Error)
	}
	if env.Error.Details != nil {
		t.Fatalf("details should be omitted, got %+v", env.Error.Details)
	}
}

func TestErrorWithDetails_IncludesDetails(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ErrorWithDetails(c, http.StatusBadRequest, CodeInvalidInput, "validation failed",
		map[string]string{"name": "required"})

	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error == nil || env.Error.Details == nil {
		t.Fatalf("details should be populated, got %+v", env.Error)
	}
}

func TestFromAppError_KnownAppError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	appErr := apperrors.NewNotFound("user", "42")
	FromAppError(c, appErr)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error == nil || env.Error.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %+v", env.Error)
	}
}

func TestFromAppError_UnknownErrorMasksInternals(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	FromAppError(c, errors.New("raw db error: password=hunter2"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error == nil || env.Error.Code != CodeInternal {
		t.Fatalf("expected internal, got %+v", env.Error)
	}
	// Sensitive content from the wrapped error must not leak.
	if env.Error.Message == "raw db error: password=hunter2" {
		t.Fatalf("message leaked internal error: %q", env.Error.Message)
	}
}
