package store

import (
	"database/sql"
	"fmt"
	"time"
)

const certificateSelectColumns = `
	id, name, domain, cert_path, key_path, source, ca_id,
	status, not_before, not_after, renew_at,
	created_at, updated_at
`

func (s *SQLite) ListCertificates() ([]Certificate, error) {
	rows, err := s.db.Query(`
		SELECT ` + certificateSelectColumns + `
		FROM certificates ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []Certificate
	for rows.Next() {
		c, err := scanCertificateRow(rows)
		if err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	return certs, nil
}

func (s *SQLite) GetCertificate(id string) (*Certificate, error) {
	row := s.db.QueryRow(`
		SELECT `+certificateSelectColumns+`
		FROM certificates WHERE id = ?
	`, id)
	c, err := scanCertificateRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *SQLite) GetCertificateByDomain(domain string) (*Certificate, error) {
	row := s.db.QueryRow(`
		SELECT `+certificateSelectColumns+`
		FROM certificates WHERE domain = ?
	`, domain)
	c, err := scanCertificateRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *SQLite) CreateCertificate(c *Certificate) error {
	_, err := s.db.Exec(`
		INSERT INTO certificates (id, name, domain, cert_path, key_path, source, ca_id, status, not_before, not_after, renew_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.Name, c.Domain, c.CertPath, c.KeyPath, c.Source, c.CAID, c.Status,
		formatTimeForDB(c.NotBefore), formatTimeForDB(c.NotAfter), formatTimeForDB(c.RenewAt),
		c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *SQLite) UpdateCertificate(c *Certificate) error {
	c.UpdatedAt = time.Now()
	_, err := s.db.Exec(`
		UPDATE certificates SET
			name = ?, domain = ?, cert_path = ?, key_path = ?,
			source = ?, ca_id = ?, status = ?,
			not_before = ?, not_after = ?, renew_at = ?, updated_at = ?
		WHERE id = ?
	`, c.Name, c.Domain, c.CertPath, c.KeyPath, c.Source, c.CAID,
		c.Status, formatTimeForDB(c.NotBefore), formatTimeForDB(c.NotAfter), formatTimeForDB(c.RenewAt),
		c.UpdatedAt, c.ID)
	return err
}

func formatTimeForDB(t time.Time) any {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func (s *SQLite) DeleteCertificate(id string) error {
	_, err := s.db.Exec("DELETE FROM certificates WHERE id = ?", id)
	return err
}

// ListExpiringLocalCertificates returns local-CA certificates whose renew_at
// is on or before the given time. Imported certificates are excluded since
// they cannot be auto-renewed.
func (s *SQLite) ListExpiringLocalCertificates(before time.Time) ([]Certificate, error) {
	rows, err := s.db.Query(`
		SELECT `+certificateSelectColumns+`
		FROM certificates
		WHERE source = 'local_ca' AND status = 'active' AND renew_at != '' AND renew_at <= ?
	`, before.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []Certificate
	for rows.Next() {
		c, err := scanCertificateRow(rows)
		if err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	return certs, nil
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanCertificateRow(r rowScanner) (Certificate, error) {
	var c Certificate
	var notBefore, notAfter, renewAt any
	var certPath, keyPath, source, caID, name sql.NullString

	if err := r.Scan(
		&c.ID, &name, &c.Domain, &certPath, &keyPath, &source, &caID,
		&c.Status, &notBefore, &notAfter, &renewAt, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return c, err
	}

	c.Name = name.String
	c.CertPath = certPath.String
	c.KeyPath = keyPath.String
	c.Source = source.String
	c.CAID = caID.String

	var err error
	if c.NotBefore, err = scanCertificateTime(notBefore); err != nil {
		return c, err
	}
	if c.NotAfter, err = scanCertificateTime(notAfter); err != nil {
		return c, err
	}
	if c.RenewAt, err = scanCertificateTime(renewAt); err != nil {
		return c, err
	}
	return c, nil
}

func scanCertificateTime(value any) (time.Time, error) {
	switch typed := value.(type) {
	case nil:
		return time.Time{}, nil
	case time.Time:
		return typed, nil
	case string:
		if typed == "" {
			return time.Time{}, nil
		}
		return parseCertificateTimeString(typed)
	case []byte:
		if len(typed) == 0 {
			return time.Time{}, nil
		}
		return parseCertificateTimeString(string(typed))
	default:
		return time.Time{}, fmt.Errorf("unsupported certificate time value type %T", value)
	}
}

func parseCertificateTimeString(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, nil
	}
	return time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", value)
}

// CACertificate represents a stored certificate authority.
type CACertificate struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	CertPEM    string    `json:"cert_pem"`
	KeyPEM     string    `json:"key_pem,omitempty"`
	NotBefore  time.Time `json:"not_before"`
	NotAfter   time.Time `json:"not_after"`
	CreatedAt  time.Time `json:"created_at"`
}

const caCertificateSelectColumns = `
	id, name, cert_pem, key_pem, not_before, not_after, created_at
`

func (s *SQLite) ListCACertificates() ([]CACertificate, error) {
	rows, err := s.db.Query(`SELECT ` + caCertificateSelectColumns + ` FROM ca_certificates ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cas []CACertificate
	for rows.Next() {
		c, err := scanCARow(rows)
		if err != nil {
			return nil, err
		}
		cas = append(cas, c)
	}
	return cas, nil
}

func (s *SQLite) GetCACertificate(id string) (*CACertificate, error) {
	row := s.db.QueryRow(`SELECT `+caCertificateSelectColumns+` FROM ca_certificates WHERE id = ?`, id)
	c, err := scanCARow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *SQLite) GetFirstCACertificate() (*CACertificate, error) {
	row := s.db.QueryRow(`SELECT ` + caCertificateSelectColumns + ` FROM ca_certificates ORDER BY created_at ASC LIMIT 1`)
	c, err := scanCARow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *SQLite) CreateCACertificate(c *CACertificate) error {
	_, err := s.db.Exec(`
		INSERT INTO ca_certificates (id, name, cert_pem, key_pem, not_before, not_after, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.Name, c.CertPEM, c.KeyPEM, c.NotBefore, c.NotAfter, c.CreatedAt)
	return err
}

func scanCARow(r rowScanner) (CACertificate, error) {
	var c CACertificate
	var notBefore, notAfter any
	if err := r.Scan(
		&c.ID, &c.Name, &c.CertPEM, &c.KeyPEM, &notBefore, &notAfter, &c.CreatedAt,
	); err != nil {
		return c, err
	}
	var err error
	if c.NotBefore, err = scanCertificateTime(notBefore); err != nil {
		return c, err
	}
	if c.NotAfter, err = scanCertificateTime(notAfter); err != nil {
		return c, err
	}
	return c, nil
}
