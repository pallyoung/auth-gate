package certificate

import (
	"errors"
	"fmt"
)

// Error codes
const (
	ErrCodeCertNotFound    = "cert_not_found"
	ErrCodeDatabase        = "database_error"
	ErrCodeInvalidDomain   = "invalid_domain"
	ErrCodeDomainExists    = "domain_exists"
	ErrCodeDNSProvider     = "dns_provider_error"
	ErrCodeACME            = "acme_error"
	ErrCodeInvalidProvider = "invalid_provider"
	ErrCodeCertNotActive   = "cert_not_active"
)

// Error represents a certificate service error
type Error struct {
	code    string
	message string
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.code, e.message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.code, e.message)
}

func (e *Error) Code() string {
	return e.code
}

func newError(code, message string, cause error) *Error {
	return &Error{
		code:    code,
		message: message,
		cause:   cause,
	}
}

// Code extracts error code from error
func Code(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.code
	}
	return "unknown_error"
}

// Message extracts error message from error
func Message(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.message
	}
	return err.Error()
}