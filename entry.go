package mflogger

import (
	"errors"
	"reflect"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type (
	// Entry extend logrus.Entry with additional convenience methods, to be easier to implement structured logging
	// of errors.
	Entry struct {
		logrus.Entry
	}
)

const (
	errorMessage = "error-message"
	errorStack   = "error-stack"
	errorType    = "error-type"
)

func NewEntry() *Entry {
	return &Entry{Entry: *logrus.WithFields(logrus.Fields{})}
}

// WithFields adds custom fields (key/value) to the log entry.
func (e *Entry) WithFields(f map[string]interface{}) *Entry {
	for k, v := range f {
		if !strings.HasPrefix(k, "_") {
			e.Entry.Data[k] = v
		}
	}
	return e
}

// WithError adds error information to the log entry.
func (e *Entry) WithError(err error) *Entry {
	if err == nil {
		return e
	}

	var innerErr = err
	for errors.Unwrap(innerErr) != nil {
		innerErr = errors.Unwrap(innerErr)
	}
	e.Entry.Data[errorType] = reflect.TypeOf(innerErr).String()

	UnwrapError(err, e.Entry.Data)

	return e
}

// WithCtx add fields from the context, to the log entry.
func (e *Entry) WithCtx(c echo.Context) *Entry {
	if c == nil {
		return e
	}

	// ContextLogFields are setup by the CreateLoggerMiddleware function.
	contextLogFields := c.Get(contextName)
	if contextLogFields == nil {
		return e
	}

	logFields, ok := contextLogFields.(map[string]interface{})
	if !ok {
		return e
	}

	e.WithFields(logFields)
	return e
}
