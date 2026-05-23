package dto

import (
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

// CertificateResponse represents a certificate in API responses
type CertificateResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Domain    string `json:"domain"`
	CertPath  string `json:"cert_path"`
	KeyPath   string `json:"key_path"`
	Status    string `json:"status"`
	NotBefore string `json:"not_before,omitempty"`
	NotAfter  string `json:"not_after,omitempty"`
	RenewAt   string `json:"renew_at,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CertificateListResponse represents a list of certificates
type CertificateListResponse []CertificateResponse

// CertificateWriteRequest represents the request to create a certificate
type CertificateWriteRequest struct {
	Name           string            `json:"name" binding:"required"`
	Domain         string            `json:"domain" binding:"required"` // e.g., "*.example.com"
	DNSProvider    string            `json:"dns_provider" binding:"required"`
	ProviderConfig map[string]string `json:"provider_config" binding:"required"`
}

// CertificateRenewRequest represents a request to renew a certificate
type CertificateRenewRequest struct{}

// CertificateResponseFromStore converts a store.Certificate to CertificateResponse
func CertificateResponseFromStore(c store.Certificate) CertificateResponse {
	return CertificateResponse{
		ID:        c.ID,
		Name:      c.Name,
		Domain:    c.Domain,
		CertPath:  c.CertPath,
		KeyPath:   c.KeyPath,
		Status:    c.Status,
		NotBefore: formatTime(c.NotBefore),
		NotAfter:  formatTime(c.NotAfter),
		RenewAt:   formatTime(c.RenewAt),
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
	}
}

// CertificateListResponseFromStore converts a list of store.Certificate to CertificateListResponse
func CertificateListResponseFromStore(certs []store.Certificate) CertificateListResponse {
	result := make(CertificateListResponse, len(certs))
	for i, c := range certs {
		result[i] = CertificateResponseFromStore(c)
	}
	return result
}

// formatTime formats a time.Time as RFC3339 string, empty if zero
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}