package eal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

var (
	ErrExpiredToken = NewHTTPError(nil, http.StatusBadRequest, "expired token")
	ErrTest         = errors.New("generic error")
)

func TestNewHTTPError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		code          int
		msg           string
		wantCode      int
		wantMsg       string
		wantInnerCode int
		wantInnerMsg  string
	}{
		{
			name:          "only_status_code",
			err:           nil,
			code:          500,
			msg:           "",
			wantCode:      http.StatusInternalServerError,
			wantMsg:       http.StatusText(http.StatusInternalServerError),
			wantInnerCode: http.StatusInternalServerError,
			wantInnerMsg:  http.StatusText(http.StatusInternalServerError),
		},
		{
			name:          "status_code_and_message",
			err:           nil,
			code:          500,
			msg:           "some message",
			wantCode:      http.StatusInternalServerError,
			wantMsg:       "some message",
			wantInnerCode: http.StatusInternalServerError,
			wantInnerMsg:  "some message",
		},
		{
			name:          "generic_error",
			err:           ErrTest,
			code:          500,
			msg:           "some message",
			wantCode:      http.StatusInternalServerError,
			wantMsg:       "some message",
			wantInnerCode: http.StatusInternalServerError,
			wantInnerMsg:  "some message",
		},
		{
			name:          "ErrorHTTPResponse",
			err:           ErrExpiredToken,
			code:          500,
			msg:           "some message",
			wantCode:      http.StatusInternalServerError,
			wantMsg:       "some message",
			wantInnerCode: http.StatusBadRequest,
			wantInnerMsg:  "expired token",
		},
		{
			name:          "wrapped_ErrorHTTPResponse",
			err:           fmt.Errorf("wrapped error message: %w", Trace(ErrExpiredToken)),
			code:          500,
			msg:           "some message",
			wantCode:      http.StatusInternalServerError,
			wantMsg:       "some message",
			wantInnerCode: http.StatusBadRequest,
			wantInnerMsg:  "expired token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got error
			if tt.msg != "" {
				got = NewHTTPError(tt.err, tt.code, tt.msg)
			} else {
				got = NewHTTPError(tt.err, tt.code)
			}

			if got == nil {
				t.Error("got nil, want echo.HTTPError")
			}

			var errMsg *echo.HTTPError
			if !errors.As(got, &errMsg) {
				t.Errorf("got error type: %T, want echo.HTTPError", got)
			}
			if errMsg.Code != tt.wantCode {
				t.Errorf("got HTTP code: %d, want: %d", errMsg.Code, tt.wantCode)
			}
			msg, ok := errMsg.Message.(string)
			if !ok {
				t.Errorf("got message type: %T, want string", errMsg.Message)
			}
			if msg != tt.wantMsg {
				t.Errorf("got HTTP message: %s, want: %s", msg, tt.wantMsg)
			}

			innerErr := GetInnerHTTPError(got)
			if innerErr.Code != tt.wantInnerCode {
				t.Errorf("got inner HTTP code: %d, want: %d", innerErr.Code, tt.wantCode)
			}
			msg, ok = innerErr.Message.(string)
			if !ok {
				t.Errorf("got inner message type: %T, want string", innerErr.Message)
			}
			if msg != tt.wantInnerMsg {
				t.Errorf("got inner HTTP message: %s, want: %s", msg, tt.wantInnerMsg)
			}
		})
	}
}

type testErr struct{ e error }

func (t testErr) Error() string   { return "testErr" }
func (t testErr) Temporary() bool { return false }
func (t testErr) Unwrap() error   { return t.e }

type testSetLogFieldsErr struct{ e error }

func (t testSetLogFieldsErr) Error() string                         { return "testErr" }
func (t testSetLogFieldsErr) SetLogFields(f map[string]interface{}) { f["set_log_fields"] = true }
func (t testSetLogFieldsErr) Unwrap() error                         { return t.e }

// nonComparableError is an error where the struct contain a slice and can't be used as a key in a map.
type nonComparableError struct {
	lines []string
}

func (e nonComparableError) Error() string {
	return strings.Join(e.lines, ",")
}

var statTestErr = errors.New("test error")

func TestUnwrapError(t *testing.T) {
	elf := func(err error, fields Fields) {
		fields["registeredErrorLogFunctions"] = true
		fields["type_"+reflect.TypeOf(err).String()] = true
		if f, ok := err.(interface{ Timeout() bool }); ok {
			fields["timeout"] = f.Timeout()
		}
		if f, ok := err.(interface{ Temporary() bool }); ok {
			fields["temporary"] = f.Temporary()
		}
	}
	RegisterErrorLogFunc(elf, statTestErr, context.DeadlineExceeded, (*testErr)(nil), (*nonComparableError)(nil))
	for _, tt := range []struct {
		name string
		err  error
		want map[string]interface{}
	}{
		{
			name: "nil",
			err:  nil,
		},
		{
			name: "static_error",
			err:  statTestErr,
			want: map[string]interface{}{"error_message": "test error", "registeredErrorLogFunctions": true, "type_*errors.errorString": true},
		},
		{
			name: "testErr-struct",
			err:  testErr{},
			want: map[string]interface{}{"error_message": "testErr"},
		},
		{
			name: "testErr-pointer",
			err:  &testErr{},
			want: map[string]interface{}{"error_message": "testErr", "registeredErrorLogFunctions": true, "temporary": false, "type_*eal.testErr": true},
		},
		{
			name: "context.DeadlineExceeded",
			err:  fmt.Errorf("test: %w", context.DeadlineExceeded),
			want: map[string]interface{}{"error_message": "test: context deadline exceeded", "registeredErrorLogFunctions": true, "timeout": true, "temporary": true, "type_context.deadlineExceededError": true},
		},
		{
			name: "context.DeadlineExceeded_wrapped_in_testErr",
			err:  &testErr{e: context.DeadlineExceeded},
			want: map[string]interface{}{"error_message": "testErr", "registeredErrorLogFunctions": true, "timeout": true, "temporary": true, "type_*eal.testErr": true, "type_context.deadlineExceededError": true},
		},
		{
			name: "testSetLogFieldsErr-struct",
			err:  testSetLogFieldsErr{},
			want: map[string]interface{}{"error_message": "testErr", "set_log_fields": true},
		},
		{
			name: "testSetLogFieldsErr-pointer",
			err:  &testSetLogFieldsErr{},
			want: map[string]interface{}{"error_message": "testErr", "set_log_fields": true},
		},
		{
			name: "testSetLogFieldsErr_wrapped_in_testErr",
			err:  &testErr{e: testSetLogFieldsErr{}},
			want: map[string]interface{}{"error_message": "testErr", "registeredErrorLogFunctions": true, "set_log_fields": true, "temporary": false, "type_*eal.testErr": true},
		},
		{
			name: "nonComparableError-pointer",
			err:  &nonComparableError{lines: []string{"test", "lines"}},
			want: map[string]interface{}{"error_message": "test,lines", "registeredErrorLogFunctions": true, "type_*eal.nonComparableError": true},
		},
		{
			name: "nonComparableError-struct",
			err:  nonComparableError{lines: []string{"test", "lines"}},
			want: map[string]interface{}{"error_message": "test,lines"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := make(map[string]any)
			UnwrapError(tt.err, got)
			if tt.err == nil && len(got) == len(tt.want) {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\n got: %v,\nwant: %v", got, tt.want)
			}
		})
	}
}
