package dto

import (
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

const (
	CertificateSourceLocalCA  = "local_ca"
	CertificateSourceImported = "imported"
)

// CertificateResponse represents a certificate in API responses.
type CertificateResponse struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Domain             string `json:"domain"`
	CertPath           string `json:"cert_path"`
	KeyPath            string `json:"key_path"`
	Source             string `json:"source"`
	CAID               string `json:"ca_id,omitempty"`
	Status             string `json:"status"`
	Organization       string `json:"organization,omitempty"`
	OrganizationalUnit string `json:"organizational_unit,omitempty"`
	Country            string `json:"country,omitempty"`
	Province           string `json:"province,omitempty"`
	Locality           string `json:"locality,omitempty"`
	NotBefore          string `json:"not_before,omitempty"`
	NotAfter           string `json:"not_after,omitempty"`
	RenewAt            string `json:"renew_at,omitempty"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

// CertificateListResponse represents a list of certificates.
type CertificateListResponse []CertificateResponse

// CertificateWriteRequest represents the request to create a certificate.
// Source = "local_ca" signs with the bundled CA. Source = "imported"
// (default) stores the user-supplied CertPEM/KeyPEM as-is.
type CertificateWriteRequest struct {
	Name               string `json:"name" binding:"required"`
	Domain             string `json:"domain" binding:"required"`
	Source             string `json:"source,omitempty"`
	CertPEM            string `json:"cert_pem,omitempty"`
	KeyPEM             string `json:"key_pem,omitempty"`
	Organization       string `json:"organization,omitempty"`
	OrganizationalUnit string `json:"organizational_unit,omitempty"`
	Country            string `json:"country,omitempty"`
	Province           string `json:"province,omitempty"`
	Locality           string `json:"locality,omitempty"`
}

// CertificateResponseFromStore converts a store.Certificate to CertificateResponse.
func CertificateResponseFromStore(c store.Certificate) CertificateResponse {
	return CertificateResponse{
		ID:                 c.ID,
		Name:               c.Name,
		Domain:             c.Domain,
		CertPath:           c.CertPath,
		KeyPath:            c.KeyPath,
		Source:             c.Source,
		CAID:               c.CAID,
		Status:             c.Status,
		Organization:       c.Organization,
		OrganizationalUnit: c.OrganizationalUnit,
		Country:            c.Country,
		Province:           c.Province,
		Locality:           c.Locality,
		NotBefore:          formatTime(c.NotBefore),
		NotAfter:           formatTime(c.NotAfter),
		RenewAt:            formatTime(c.RenewAt),
		CreatedAt:          c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          c.UpdatedAt.Format(time.RFC3339),
	}
}

// CertificateListResponseFromStore converts a list of store.Certificate to CertificateListResponse.
func CertificateListResponseFromStore(certs []store.Certificate) CertificateListResponse {
	result := make(CertificateListResponse, len(certs))
	for i, c := range certs {
		result[i] = CertificateResponseFromStore(c)
	}
	return result
}

// CAExportResponse describes the bundled local CA.
type CAExportResponse struct {
	CertPEM  string `json:"cert_pem"`
	Name     string `json:"name"`
	NotAfter string `json:"not_after"`
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
