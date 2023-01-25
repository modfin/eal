package eal

import (
	"database/sql"
	"net"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

func ExampleAddContextFields() {
	e := echo.New()

	// Initialize the logging middleware
	e.Use(CreateLoggerMiddleware())

	e.GET("/ping", func(c echo.Context) error {
		userID := c.FormValue("user-id")

		// Add "user-id" field to context, that will be included in the log entry generated by the middleware when
		// handler have returned.
		AddContextFields(c, Fields{"user-id": userID})

		return c.String(200, "")
	})
}

func ExampleCreateLoggerMiddleware() {
	e := echo.New()
	e.Use(CreateLoggerMiddleware())
}

func ExampleInhibitStacktraceForError_errorReference() {
	// Don't generate a stacktrace when Trace is called with a sql.ErrNoRows error.
	InhibitStacktraceForError(sql.ErrNoRows)
}

func ExampleInhibitStacktraceForError_errorType() {
	// Don't generate a stacktrace when Trace is called with a jwt.ValidationError error type.
	InhibitStacktraceForError((*jwt.ValidationError)(nil))
}

func ExampleRegisterErrorLogFunc_single() {
	RegisterErrorLogFunc(func(err error, fields Fields) {
		oe, ok := err.(*net.OpError)
		if !ok {
			return
		}
		fields["net_oper"] = oe.Op
		fields["net_addr"] = oe.Addr.String()
		fields["temporary"] = oe.Temporary()
		fields["timeout"] = oe.Timeout()
	}, (*net.OpError)(nil))
}

func ExampleRegisterErrorLogFunc_multiple() {
	errFmt := func(err error, fields Fields) {
		var i interface{} = err
		switch e := i.(type) {
		case *net.OpError:
			fields["net_oper"] = e.Op
			fields["net_addr"] = e.Addr.String()
			fields["temporary"] = e.Temporary()
			fields["timeout"] = e.Timeout()
		case *net.ParseError:
			fields["net_type"] = e.Type
			fields["net_text"] = e.Text
		}
	}
	RegisterErrorLogFunc(errFmt, (*net.OpError)(nil), (*net.ParseError)(nil))
}
