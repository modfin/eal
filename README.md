# eal
Extended Access Logging
> Simplifies access and error logging of Labstack/echo HTTP servers

## Setup echo access/error logging
To get started with access and error logging, it's enough to call `eal.Init`, `eal.InitDefaultErrorLogging` (both are optional),
and then add the middleware returned by `eal.CreateLoggerMiddleware()` to the echo server.

```go
package main

import (
  "github.com/labstack/echo/v4"
  "github.com/modfin/eal"
)

func main() {
  // Initialize logrus JSON logger.
  eal.Init(false)

  // Initialize eal default error logging for echo.HTTPError and jwt.ValidationError error types.
  eal.InitDefaultErrorLogging()

  // Create echo instance and set up the access logging middleware.
  e := echo.New()
  e.Use(eal.CreateLoggerMiddleware())

  // Setup endpoints and start echo server the usual way...
  // ...
```

## Add information to access/error log entry
To extend the log entry that is going to be written when the endpoint is about to return, one can use the `AddContextFields` method.
```go
  e.POST("/user", func(c echo.Context) error {
    userID := c.FormValue("user-id")

    // Add "user-id" field to context, that will be included in the log entry generated by the middleware when
    // handler have returned.
    eal.AddContextFields(c, Fields{"user-id": userID})

    // ...
  })
```

## Add stacktrace information to logged errors
To generate a stacktrace, the `Trace` method can be used. `Trace` takes an error and wrap it in a new error that contain a stacktrace. 
It is possible to configure what errors and error types that shouldn't generate a stacktrace (see `InhibitStacktraceForError` for more information). 
If the error provided to `Trace` already is, or contain, a wrapped stacktrace-error, the original error will be returned unmodified.

There is a global parameter that can be set that affect when the stacktrace is first logged: `LogCallStackDirectly`. If it's
`true`, `Trace` will write a new log entry directly after the new stacktrace error have been created. This can be useful if there is a chance
that the error returned by `Trace` isn't wrapped and returned to the middleware logger.

```go
  if err != nil {
    // Wrap the original error in a stacktrace, before wrapping it in a new error with more information (GO 1.13 and later)
    return fmt.Errorf("encode: %v: %w", data, eal.Trace(err))
  }
```

## Add more error information to the log event
Some error types may have more information than what's shown in the `Error()` string, or if it's desirable to have some error information
logged as a separate field in the log. The `RegisterErrorLogFunc` method can be used to extend the log entry with specific error information.

See `InitDefaultErrorLogging()` for an example of how to use `RegisterErrorLogFunc`.

## Send Error information to caller
Normally echo will send back a HTTP status 500 when an error is returned from the echo handlerFunc, unless the error is a echo.HTTPError.
When the `eal.CreateLoggerMiddleware` is used, it will look for the earliest echo.HTTPError if can find in the returned error, and return
that to echo, if the returned error don't contain a wrapped echoHTTPError, the error will be passed on to echo unmodified.

```go
var errNope error = echo.NewHTTPError(http.StatusNotFound, "Nope") // Returns 404 {"message":"Nope"}, to caller

...

  e.GET("/droids", func(c echo.Context) error {
    return errNope
  })

```

or if the error information that we want to send back is caused by an error, eal implement a `NewHTTPError` method that wrap an error in a
echo.HTTPError


```go
func errNope(err error) error {
  // Wrap the error in a stacktrace, and then wrap it in a echo.HTTPError
  return eal.NewHTTPError(eal.Trace(err), http.StatusNotFound, "Nope") // Return 404 {"message":"Nope"}, to caller
}

...

  e.GET("/droids", func(c echo.Context) error {
    d, err := getDroids()
    if err != nil {
      return errNope(err)
    }
    return c.JSON(http.StatusOK, d)
  })

```

it's also possible to send back a custom JSON message to the caller by using a struct as a parameter in the echo.HTTPError

```go
type ErrorMessage struct {
  ErrorCode    int    `json:"error_code"`
  ErrorMessage string `json:"error_message"`
}

var ErrSomeMessage error = echo.NewHTTPError(http.StatusNotFound, &ErrorMessage{ErrorCode: 42, ErrorMessage: "common.error.some_message"})
```