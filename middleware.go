package eal

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	uuid "github.com/nu7hatch/gouuid"
)

const (
	contextName = "mfContextLogFields"
)

// CreateLoggerMiddleware return an echo middleware method that handle access and error logging of the call.
//
// If an error is returned from the handlerFunc, the middleware will look at the complete error-chain to find the
// earliest echo.HTTPError, and return the status code and message from that to the frontend.
// If the error-chain don't contain an echo.HTTPError, a new echo.HTTPError will be created that wrap the returned error.
func CreateLoggerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			// Init
			req := c.Request()
			res := c.Response()

			// Check if we have X-Host or X-Forwarded-Host header
			host := req.Header.Get("X-Host")
			if host == "" {
				alt := req.Header.Get("X-Forwarded-Host")
				if alt != "" {
					host = strings.Split(alt, ":")[0]
					req.Header.Set("X-Host", host)
				}
			}

			// Generate Request ID if it's missing
			id := req.Header.Get("X-Request-Id")
			if id == "" {
				a, _ := uuid.NewV4()
				id = a.String()
				req.Header.Set("X-Request-Id", id)
				res.Header().Set("X-Request-Id", id)
			}

			// Setup logging context
			logFields := Fields{
				"request_id":  id,
				"remote_ip":   req.Header.Get("X-Remote-Addr"),
				"host":        host,
				"method":      req.Method,
				"uri":         req.RequestURI,
				"router_path": c.Path(),
			}
			c.Set(contextName, logFields)
			// TODO: Look into also setting logFields on c.Request().Context()?

			// Run other middlewares/handlers
			start := time.Now()
			err = next(c)
			stop := time.Now()

			// Handle request/response errors
			if err != nil {
				errMsg := GetInnerHTTPError(err)
				if errMsg != nil {
					c.Error(errMsg)
				} else {
					err = &echo.HTTPError{Code: http.StatusInternalServerError, Message: http.StatusText(http.StatusInternalServerError), Internal: err}
					c.Error(err)
				}
			}

			// Log request result
			latency := int64(stop.Sub(start) / time.Millisecond)
			logFields["latency_ms"] = latency
			logFields["status"] = res.Status

			// Create log entry
			logEntry := NewEntry()
			logEntry = logEntry.WithFields(logFields)
			if err != nil {
				logEntry = logEntry.WithError(err)
			}

			msg, ok := logFields["_msg"]
			if !ok {
				msg = "access"
			}

			if _, ok := logEntry.Data[errorMessage]; ok {
				logEntry.Error(msg)
			} else {
				logEntry.Info(msg)
			}

			return nil
		}
	}
}

// AddContextFields add the fields to the log context, fields added to the context is included in logging done by the
// CreateLoggerMiddleware. The fields added by this method can also be logged elsewhere by using Entry.WithCtx
// method.
func AddContextFields(c echo.Context, fields Fields) {
	if c == nil {
		return
	}

	lc := c.Get(contextName)
	logFields, ok := lc.(Fields)
	if !ok || logFields == nil {
		return
	}

	for k, v := range fields {
		logFields[k] = v
	}
}
