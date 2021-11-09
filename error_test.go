package eal

import (
	"errors"
	"fmt"
	"net/http"
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
