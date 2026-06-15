package store

import "time"

// CACertificate represents a stored certificate authority.
type CACertificate struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CertPEM   string    `json:"cert_pem"`
	KeyPEM    string    `json:"key_pem,omitempty"`
	NotBefore time.Time `json:"not_before"`
	NotAfter  time.Time `json:"not_after"`
	CreatedAt time.Time `json:"created_at"`
}
