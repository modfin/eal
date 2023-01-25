package eal

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/golang-jwt/jwt/v4"
)

type (
	testError struct {
		msg string
	}
)

const testErrorMessage = "test error 1"

var (
	errTest1 = errors.New(testErrorMessage)
	errTest2 = base64.CorruptInputError(42)
)

func (e *testError) Error() string {
	return e.msg
}

func TestTrace(t *testing.T) {
	// Don't generate stack-traces for sql.ErrNoRows, or for jwt.ValidationError error types
	InhibitStacktraceForError(sql.ErrNoRows, (*jwt.ValidationError)(nil))

	for _, tt := range []struct {
		name           string
		err            error
		wantNilError   bool
		wantErrorType  string
		wantStackTrace bool
	}{
		{name: "nil", err: nil, wantNilError: true},
		{name: "test1", err: errTest1, wantErrorType: "*eal.ErrorStackTrace", wantStackTrace: true},
		{name: "wrapped", err: fmt.Errorf("wrapped test error: %w", Trace(errTest2)), wantErrorType: "*fmt.wrapError", wantStackTrace: true},
		{name: "sql_ErrNoRows", err: sql.ErrNoRows, wantErrorType: "*errors.errorString"},
		{name: "jwt_ValidationError", err: &jwt.ValidationError{}, wantErrorType: "*jwt.ValidationError"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := Trace(tt.err)
			if tt.wantNilError {
				if err != nil {
					t.Errorf("got err: %v, want: nil", err)
				}
				return
			}

			et := reflect.TypeOf(err)
			if et.String() != tt.wantErrorType {
				t.Errorf("got error type: %v, want: %v", et.String(), tt.wantErrorType)
			}

			var est *ErrorStackTrace
			estOK := errors.As(err, &est)
			if estOK != tt.wantStackTrace {
				t.Errorf("got stack-trace: %v , want: %v", estOK, tt.wantStackTrace)
			}
		})
	}
}

func TestTraceEdgeCase(t *testing.T) {
	// Test interface type edge case, where we have a non nil type pointer with a nil value pointer.
	var err error
	var te *testError
	err = te // cast the nil *testError variable, to an error interface, creating an error that isn't nil.
	if err == nil {
		// This should never happen with the current implementation regarding interfaces and nil-checking in GO,
		// but since the test relies on (*type)(nil) != nil, we check that this is also true with future GO versions
		// used to compile/run the test.
		t.Error("typed nil interface were nil ((*type)(nil) == nil)")
	}
	err = Trace(te)
	if err != nil {
		t.Errorf("got interface: (%[1]T)(%[1]v), want (%[2]T)(%[2]v)", err, nil)
	}
}

func TestGetErrorStackTrace(t *testing.T) {
	est := Trace(errTest1)
	wrappedErr := fmt.Errorf("wrapped test error: %w", Trace(errTest1))

	for n, tt := range []struct {
		err    error
		wantOk bool
	}{
		{err: errTest1, wantOk: false},
		{err: est, wantOk: true},
		{err: wrappedErr, wantOk: true},
	} {
		t.Run(fmt.Sprintf("%d", n), func(t *testing.T) {
			err, ok := GetErrorStackTrace(tt.err)
			if ok != tt.wantOk {
				t.Errorf("got ok: %v, want: %v", ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if err == nil {
				t.Fatalf("Returned ErrorStackTrace is nil")
			}

			if err.Error() != testErrorMessage {
				t.Errorf("got error message: %s, want: %s", err.Error(), testErrorMessage)
			}
			if err.TypeName() != "*errors.errorString" {
				t.Errorf("got err.TypeName(): %s want: *errors.errorString", err.TypeName())
			}
			if err.Stack() == "" {
				t.Error("got empty err.Stack(), want non empty call stack")
			}

			lf := make(map[string]interface{})
			err.SetLogFields(lf)
			st, ok := lf[errorStack]
			if !ok {
				t.Errorf("SetLogFields() didn't set the %s field", errorStack)
			} else if st == "" {
				t.Errorf("got an empty %s field, want a callstack", errorStack)
			}

			uwErr := err.Unwrap()
			if uwErr == nil {
				t.Fatal("got err.Unwrap() = nil, want non nil")
			}
			if !errors.Is(uwErr, errTest1) {
				t.Errorf("err.Unwrap() want 'errTest1', got [%T, %[1]v]", uwErr)
			}
		})
	}
}
