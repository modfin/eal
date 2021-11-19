package eal

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
)

type (
	// Fields hold the map of key/value log fields that should be logged.
	Fields map[string]interface{}

	// ErrLogFunc type can be implemented to be able to add log fields for a specific error.
	//
	// See RegisterErrorLogFunc and UnwrapError regarding the SetLogFields interface for more information.
	ErrLogFunc func(err error, fields Fields)
)

// Define some common log field names used by the errorLogger
const (
	httpMessage    = "http-message"
	httpStatusCode = "http-status"
	jwtError       = "jwt-error"
	jwtText        = "jwt-text"
)

var (
	registeredErrorLogFunctions = make(map[interface{}]ErrLogFunc)
)

// InitDefaultErrorLogging register a error logger that append more information to the log for echo.HTTPError and
// jwt.ValidationError errors.
func InitDefaultErrorLogging() {
	RegisterErrorLogFunc(errorLogger, (*echo.HTTPError)(nil), (*jwt.ValidationError)(nil))
}

func errorLogger(err error, fields Fields) {
	var i interface{} = err
	switch e := i.(type) {
	case *echo.HTTPError:
		fields[httpMessage] = e.Message
		fields[httpStatusCode] = e.Code

	case *jwt.ValidationError:
		var vErr string
		errBits := e.Errors

		if errBits&jwt.ValidationErrorMalformed != 0 {
			vErr += "ValidationErrorMalformed "
		}
		if errBits&jwt.ValidationErrorUnverifiable != 0 {
			vErr += "ValidationErrorUnverifiable "
		}
		if errBits&jwt.ValidationErrorSignatureInvalid != 0 {
			vErr += "ValidationErrorSignatureInvalid "
		}
		if errBits&jwt.ValidationErrorAudience != 0 {
			vErr += "ValidationErrorAudience "
		}
		if errBits&jwt.ValidationErrorExpired != 0 {
			vErr += "ValidationErrorExpired "
		}
		if errBits&jwt.ValidationErrorIssuedAt != 0 {
			vErr += "ValidationErrorIssuedAt "
		}
		if errBits&jwt.ValidationErrorIssuer != 0 {
			vErr += "ValidationErrorIssuer "
		}
		if errBits&jwt.ValidationErrorNotValidYet != 0 {
			vErr += "ValidationErrorNotValidYet "
		}
		if errBits&jwt.ValidationErrorId != 0 {
			vErr += "ValidationErrorId "
		}
		if errBits&jwt.ValidationErrorClaimsInvalid != 0 {
			vErr += "ValidationErrorClaimsInvalid "
		}
		if vErr != "" {
			fields[jwtError] = vErr
		}
		if e.Inner == nil {
			fields[jwtText] = e.Errors
		}
	default:
		fields["errorlogger"] = fmt.Sprintf("eal.errorlogger: Don't know how to handle %T error type ", err)
	}
}

// GetInnerHTTPError check if the provided error is, or have a wrapped echo.HTTPError, and if there is one, it's returned.
// If the error chain contains more than one, the inner/earliest is returned.
func GetInnerHTTPError(err error) *echo.HTTPError {
	var errMsg *echo.HTTPError
	for err != nil {
		if errors.As(err, &errMsg) {
			err = errMsg.Internal
		} else {
			err = nil
		}
	}
	return errMsg
}

// NewHTTPError complements echo.NewHTTPError, this also takes an error as a parameter.
func NewHTTPError(err error, code int, msg ...interface{}) error {
	var hErr *echo.HTTPError
	if len(msg) > 0 {
		hErr = echo.NewHTTPError(code, msg...)
	} else {
		hErr = echo.NewHTTPError(code)
	}
	_ = hErr.SetInternal(err)

	return hErr
}

// RegisterErrorLogFunc registers a function that is called when a specific error interface is seen by UnwrapError.
// If you have your own error types (structs) that you want to log, it is easier to implement a SetLogFields method
// to handle logging. RegisterErrorLogFunc should be used for other error types that you don't have any control over,
// that contains information that isn't exposed via the Error() method or if you want to use structured logging for
// data in the error type, for example:
//  RegisterErrorLogFunc(func(err error, fields map[string]interface{}) {
//    oe, ok := err.(*net.OpError)
//    if !ok {
//      return
//    }
//    fields["net_oper"] = oe.Op
//    fields["net_addr"] = oe.Addr.String()
//    fields["temporary"] = oe.Temporary()
//    fields["timeout"] = oe.Timeout()
//  }, (*net.OpError)(nil))
func RegisterErrorLogFunc(errFmtFunc ErrLogFunc, errList ...error) {
	for _, err := range errList {
		t := reflect.ValueOf(err)
		if t.Kind() == reflect.Ptr && t.IsNil() {
			registeredErrorLogFunctions[reflect.TypeOf(err)] = errFmtFunc
		} else {
			registeredErrorLogFunctions[err] = errFmtFunc
		}
	}
}

// UnwrapError walks the error-chain and add information to the provided log-fields. For each error in the error-chain,
// it will check if the error either implements the SetLogFields(map[string]interface{}) interface or if the type have a
// registered log function that is used to populate the log-fields.
// This is used by Entry.WithError to add error information to a log event.
func UnwrapError(err error, fields map[string]interface{}) {
	if err == nil {
		return
	}

	fields[errorMessage] = err.Error()

	for err != nil {
		// First check if error implement SetLogFields(LogFields)
		if slf, ok := err.(interface{ SetLogFields(map[string]interface{}) }); ok {
			slf.SetLogFields(fields)
			err = errors.Unwrap(err)
			continue
		}

		// Check if error type have a registered ErrLogFunc
		if logFunc, ok := registeredErrorLogFunctions[err]; ok {
			logFunc(err, fields)
			err = errors.Unwrap(err)
			continue
		}

		if logFunc, ok := registeredErrorLogFunctions[reflect.TypeOf(err)]; ok {
			logFunc(err, fields)
		}
		err = errors.Unwrap(err)
	}
}
