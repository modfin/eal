package eal

import (
	"errors"
	"reflect"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type (
	// Entry extend the logrus.Entry type, with additional convenience methods: WithCtx, WithError and WithFields,
	// to simplify logging.
	Entry struct {
		logrus.Entry
	}
)

const (
	errorMessage = "error_message"
	errorStack   = "error_stack"
	errorType    = "error_type"
)

// NewEntry return an Entry instance to be used for creating a log entry.
// For example:
//  eal.NewEntry().Info("App started")
func NewEntry() *Entry {
	return &Entry{Entry: *logrus.WithFields(logrus.Fields{})}
}

// WithFields adds custom fields (key/value) to the log entry.
// For example:
//  eal.NewEntry().WithFields(eal.Fields{"time": time.Since(start)}).Info("Work completed")
func (e *Entry) WithFields(f map[string]interface{}) *Entry {
	for k, v := range f {
		if !strings.HasPrefix(k, "_") {
			e.Entry.Data[k] = v
		}
	}
	return e
}

// WithError uses UnwrapError internally to extract more information from the error and add it to the log entry fields.
//
// See UnwrapError and RegisterErrorLogFunc methods for more information about how to extend the log entry fields.
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
