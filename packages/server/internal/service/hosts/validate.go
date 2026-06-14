package hostservice

import (
	"net"
	"regexp"
	"strings"
)

var (
	hostnameRegex    = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	profileNameRegex = regexp.MustCompile(`^[A-Za-z0-9 _.\-]+$`)
)

const (
	maxCommentLen      = 200
	maxProfileNameLen  = 32
)

func validateIP(in string) error {
	s := strings.TrimSpace(in)
	if s == "" {
		return newError(ErrCodeInvalidIP, "ip is required", nil)
	}
	if net.ParseIP(s) == nil {
		return newError(ErrCodeInvalidIP, "ip is not a valid IPv4 or IPv6 address", nil)
	}
	return nil
}

func validateHostname(in string) error {
	s := strings.TrimSpace(in)
	if s == "" {
		return newError(ErrCodeInvalidHostname, "hostname is required", nil)
	}
	if !hostnameRegex.MatchString(s) {
		return newError(ErrCodeInvalidHostname, "hostname is not in valid format", nil)
	}
	return nil
}

func validateProfileName(in string) error {
	s := strings.TrimSpace(in)
	if s == "" {
		return newError(ErrCodeInvalidProfileName, "profile name is required", nil)
	}
	if len(s) > maxProfileNameLen {
		return newError(ErrCodeInvalidProfileName, "profile name is too long (max 32)", nil)
	}
	if !profileNameRegex.MatchString(s) {
		return newError(ErrCodeInvalidProfileName, "profile name contains invalid characters", nil)
	}
	return nil
}

func validateComment(in string) error {
	if len(in) > maxCommentLen {
		return newError(ErrCodeInvalidComment, "comment is too long (max 200)", nil)
	}
	return nil
}
