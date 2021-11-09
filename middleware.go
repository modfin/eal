package eal

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	uuid "github.com/nu7hatch/gouuid"
)

const (
	contextName = "mfContextLogFields"
)

// CreateLoggerMiddleware return a echo middleware function that handle logging of the call.
func CreateLoggerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			// Init
			req := c.Request()
			res := c.Response()

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
				"host":        req.Header.Get("X-Host"),
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

// AddContextFields adds log fields the log context, and is included in logging done by the CreateLoggerMiddleware middleware.
// Any fields added by this method can also be logged elsewhere by using Entry.WithCtx function.
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
