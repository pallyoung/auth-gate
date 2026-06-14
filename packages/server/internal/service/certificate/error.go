package certificate

import (
	"errors"
	"fmt"
)

const (
	ErrCodeCertNotFound        = "cert_not_found"
	ErrCodeDatabase            = "database_error"
	ErrCodeInvalidName         = "invalid_name"
	ErrCodeInvalidDomain       = "invalid_domain"
	ErrCodeDomainExists        = "domain_exists"
	ErrCodeInvalidPEM          = "invalid_pem"
	ErrCodeDomainMismatch      = "domain_mismatch"
	ErrCodeImportedCannotResign = "imported_cannot_resign"
	ErrCodeLocalCA             = "local_ca_error"
	ErrCodeFilesystem          = "filesystem_error"
)

type Error struct {
	code    string
	message string
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s (%v)", e.message, e.cause)
	}
	return e.message
}

func (e *Error) Code() string { return e.code }

func newError(code, message string, cause error) *Error {
	return &Error{code: code, message: message, cause: cause}
}

// NewError creates a service-tagged error. Exported for tests and for
// callers outside this package that need to surface one of the typed
// error codes through writeServiceError.
func NewError(code, message string, cause error) *Error {
	return newError(code, message, cause)
}

func Code(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.code
	}
	return "unknown_error"
}

func Message(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.message
	}
	if err == nil {
		return ""
	}
	return err.Error()
}
