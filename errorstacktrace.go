package mflogger

import (
	"errors"
	"reflect"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

// ErrorStackTrace is created by the Trace function and hold a stacktrace to where Trace where first called.
// The error message returned by Error isn't changed from the original error message. To retrieve the recorded
// callstack, the Stack function can be used, the callstack is also logged so the only way to retrieve
// the callstack, is to either walk the chain of errors
type ErrorStackTrace struct {
	err   error
	stack string
}

var (
	// LogCallStackDirectly control if an error message should be logged immediately with the callstack
	// when the Trace method is called. If there is a chance that the error that is returned by the Trace
	// method is thrown away before it's logged, LogCallStackDirectly can be set to true to log the callstack
	// immediately.
	LogCallStackDirectly bool

	inhibitStacktraceForError = make(map[interface{}]struct{})
)

// InhibitStacktraceForError will inhibit/block/skip stacktrace generation for certain error types/instances when
// Trace is called. If a stacktrace isn't generated, Trace will instead return the supplied error.
//
// To block Trace from generating stacktraces for sql.ErrNoRows or jwt.ValidationError types, use:
//  InhibitStacktraceForError(sql.ErrNoRows, (*jwt.ValidationError)(nil), (*echo.HTTPError)(nil))
func InhibitStacktraceForError(err ...error) {
	for _, errItem := range err {
		t := reflect.ValueOf(errItem)
		if t.Kind() == reflect.Ptr && t.IsNil() {
			inhibitStacktraceForError[reflect.TypeOf(errItem)] = struct{}{}
		} else {
			inhibitStacktraceForError[errItem] = struct{}{}
		}
	}
}

// Error return the wrapped errors message, the ErrorStackTrace type don't add the stacktrace information to the
// error string. The stacktrace can be accessed by calling Stack, or through SetLogFields.
func (st *ErrorStackTrace) Error() string {
	return st.err.Error()
}

// SetLogFields is used by Entry.WithError to populate log fields.
func (st *ErrorStackTrace) SetLogFields(logFields map[string]interface{}) {
	logFields[errorStack] = st.stack
}

// Unwrap return the wrapped error.
func (st *ErrorStackTrace) Unwrap() error {
	return st.err
}

// Stack return the stacktrace to where the ErrorStackTrace first were inserted in the error chain.
func (st *ErrorStackTrace) Stack() string {
	return st.stack
}

// TypeName return the name of the wrapped error struct.
func (st *ErrorStackTrace) TypeName() string {
	return reflect.TypeOf(st.err).String()
}

// Trace return an error that gather the callstack and wraps the provided error. If the provided error already is, or
// contain a wrapped ErrorStackTrace error, the provided error is returned without modification.
func Trace(err error) error {
	if err == nil {
		return nil
	}

	// Edge case: if we receive an interface that have a non nil type, but a nil value (interfaces is a tuple with a type pointer and a value pointer)
	t := reflect.ValueOf(err)
	if t.Kind() == reflect.Ptr && t.IsNil() {
		logrus.WithField(errorStack, string(debug.Stack())).Errorf("# NON NIL INTERFACE TYPE DETECTED (error value is nil, error type is %T) #", err)

		// Since this probably isn't an error per se, we return nil, instead of returning a non nil interface type.
		return nil
	}

	if _, ok := inhibitStacktraceForError[err]; ok {
		// Return the supplied error since we shouldn't generate a stacktrace for this error instance
		return err
	}

	if _, ok := inhibitStacktraceForError[reflect.TypeOf(err)]; ok {
		// Return the supplied error since we shouldn't generate a stacktrace for this error type
		return err
	}

	// Check if we already have a wrapped ErrorStackTrace
	var st *ErrorStackTrace
	if errors.As(err, &st) {
		return err
	}

	trace := string(debug.Stack())
	if LogCallStackDirectly {
		logrus.WithFields(logrus.Fields{errorMessage: err.Error(), errorStack: trace}).Error("ERROR")
	}

	return &ErrorStackTrace{
		err:   err,
		stack: trace,
	}
}

// GetErrorStackTrace check if the provided error is, or have a wrapped ErrorStackTrace, and if there is one, it's returned.
func GetErrorStackTrace(err error) (st *ErrorStackTrace, ok bool) {
	return st, errors.As(err, &st)
}
