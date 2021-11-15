// Package eal (Extended Access Logging) is used to simplify access and error logging of GO endpoints.
// It can also be used to help create a structured way of handling error codes to be sent to frontend.
//
// A small example of how this package can be used:
//	package main
//
//	import (
//		"net/http"
//
//		"github.com/labstack/echo/v4"
//		"github.com/modfin/eal"
//	)
//
//	type FrontendMessage struct {
//		ErrorCode    int    `json:"error_code"`
//		ErrorMessage string `json:"error_message"`
//	}
//
//	// Show two different echo.HTTPError examples, the first where the message parameter is set to a string.
//	// That will send JSON: {"message":"Nope"} to caller, and the second where te message parameter is set to an object
//	// (ErrSomeMessage), that will send back JSON: {"error_code:42,"error_message":"common.error.some_message"} to the caller.
//	var (
//		ErrNope         error = echo.NewHTTPError(http.StatusForbidden, "Nope")          // Return 403 {"message":"Nope"}, to caller
//		ErrUserDisabled error = echo.NewHTTPError(http.StatusForbidden, "User disabled") // Return 403 {"message":"User disabled"}, to caller
//		ErrSomeMessage  error = echo.NewHTTPError(http.StatusNotFound, &FrontendMessage{ErrorCode: 42, ErrorMessage: "common.error.some_message"})
//	)
//
//	func ErrUser(err error) error {
//		return eal.NewHTTPError(eal.Trace(err), http.StatusInternalServerError, "User error") // Return 500 User error, to caller
//	}
//
//	func getUser(c echo.Context) (User, error) {
//		usr, err := dao.GetUser()
//		if err != nil {
//			// Failed to get user, send back generic user error message to caller
//			return nil, ErrUser(err)
//		}
//		if usr.Disabled {
//			// User is disabled, send back user disabled message to caller
//			return ErrUserDisabled
//		}
//		return usr, nil
//	}
//
//	func main() {
//		// Initialize logrus JSON logger.
//		eal.Init(false)
//
//		// Initialize eal default error logging for echo.HTTPError and jwt.ValidationError error types.
//		eal.InitDefaultErrorLogging()
//
//		// Create echo instance and set up the access logging middleware.
//		e := echo.New()
//		e.Use(eal.CreateLoggerMiddleware())
//
//		e.GET("/ping", func(c echo.Context) error {
//			usr, err := getUser(c)
//			if err != nil {
//				// When several echo.HTTPError exist in the error-chain, only the first/earliest will be sent to the caller.
//				return ErrUser(err)
//			}
//			return c.String(200, "pong")
//		})
//
//		e.GET("/nope", func(c echo.Context) error {
//			return ErrNope
//		})
//	}
package eal
