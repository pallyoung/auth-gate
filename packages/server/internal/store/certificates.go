package store

import (
	"database/sql"
	"fmt"
	"time"
)

const certificateSelectColumns = `
	id, name, domain, cert_path, key_path, dns_provider, dns_provider_config,
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
		var c Certificate
		var notBefore, notAfter, renewAt any
		var certPath, keyPath, dnsProvider, dnsProviderConfig sql.NullString
		var name sql.NullString

		err := rows.Scan(
			&c.ID, &name, &c.Domain, &certPath, &keyPath, &dnsProvider, &dnsProviderConfig,
			&c.Status, &notBefore, &notAfter, &renewAt, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		c.Name = name.String
		c.CertPath = certPath.String
		c.KeyPath = keyPath.String
		c.DNSProvider = dnsProvider.String
		c.DNSProviderConfig = dnsProviderConfig.String

		if c.NotBefore, err = scanCertificateTime(notBefore); err != nil {
			return nil, err
		}
		if c.NotAfter, err = scanCertificateTime(notAfter); err != nil {
			return nil, err
		}
		if c.RenewAt, err = scanCertificateTime(renewAt); err != nil {
			return nil, err
		}

		certs = append(certs, c)
	}
	return certs, nil
}

func (s *SQLite) GetCertificate(id string) (*Certificate, error) {
	var c Certificate
	var notBefore, notAfter, renewAt any
	var certPath, keyPath, dnsProvider, dnsProviderConfig sql.NullString
	var name sql.NullString

	err := s.db.QueryRow(`
		SELECT `+certificateSelectColumns+`
		FROM certificates WHERE id = ?
	`, id).Scan(
		&c.ID, &name, &c.Domain, &certPath, &keyPath, &dnsProvider, &dnsProviderConfig,
		&c.Status, &notBefore, &notAfter, &renewAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	c.Name = name.String
	c.CertPath = certPath.String
	c.KeyPath = keyPath.String
	c.DNSProvider = dnsProvider.String
	c.DNSProviderConfig = dnsProviderConfig.String

	if c.NotBefore, err = scanCertificateTime(notBefore); err != nil {
		return nil, err
	}
	if c.NotAfter, err = scanCertificateTime(notAfter); err != nil {
		return nil, err
	}
	if c.RenewAt, err = scanCertificateTime(renewAt); err != nil {
		return nil, err
	}

	return &c, nil
}

func (s *SQLite) GetCertificateByDomain(domain string) (*Certificate, error) {
	var c Certificate
	var notBefore, notAfter, renewAt any
	var certPath, keyPath, dnsProvider, dnsProviderConfig sql.NullString
	var name sql.NullString

	err := s.db.QueryRow(`
		SELECT `+certificateSelectColumns+`
		FROM certificates WHERE domain = ?
	`, domain).Scan(
		&c.ID, &name, &c.Domain, &certPath, &keyPath, &dnsProvider, &dnsProviderConfig,
		&c.Status, &notBefore, &notAfter, &renewAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	c.Name = name.String
	c.CertPath = certPath.String
	c.KeyPath = keyPath.String
	c.DNSProvider = dnsProvider.String
	c.DNSProviderConfig = dnsProviderConfig.String

	if c.NotBefore, err = scanCertificateTime(notBefore); err != nil {
		return nil, err
	}
	if c.NotAfter, err = scanCertificateTime(notAfter); err != nil {
		return nil, err
	}
	if c.RenewAt, err = scanCertificateTime(renewAt); err != nil {
		return nil, err
	}

	return &c, nil
}

func (s *SQLite) CreateCertificate(c *Certificate) error {
	_, err := s.db.Exec(`
		INSERT INTO certificates (id, name, domain, cert_path, key_path, dns_provider, dns_provider_config, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.Name, c.Domain, c.CertPath, c.KeyPath, c.DNSProvider, c.DNSProviderConfig, c.Status, c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *SQLite) UpdateCertificate(c *Certificate) error {
	c.UpdatedAt = time.Now()
	_, err := s.db.Exec(`
		UPDATE certificates SET
			name = ?, domain = ?, cert_path = ?, key_path = ?,
			dns_provider = ?, dns_provider_config = ?, status = ?,
			not_before = ?, not_after = ?, renew_at = ?, updated_at = ?
		WHERE id = ?
	`, c.Name, c.Domain, c.CertPath, c.KeyPath, c.DNSProvider, c.DNSProviderConfig,
		c.Status, c.NotBefore, c.NotAfter, c.RenewAt, c.UpdatedAt, c.ID)
	return err
}

func (s *SQLite) DeleteCertificate(id string) error {
	_, err := s.db.Exec("DELETE FROM certificates WHERE id = ?", id)
	return err
}

// ListExpiringCertificates returns certificates that need renewal
func (s *SQLite) ListExpiringCertificates(before time.Time) ([]Certificate, error) {
	rows, err := s.db.Query(`
		SELECT `+certificateSelectColumns+`
		FROM certificates
		WHERE status = 'active' AND NULLIF(renew_at, '') <= ?
	`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []Certificate
	for rows.Next() {
		var c Certificate
		var notBefore, notAfter, renewAt any
		var certPath, keyPath, dnsProvider, dnsProviderConfig sql.NullString
		var name sql.NullString

		err := rows.Scan(
			&c.ID, &name, &c.Domain, &certPath, &keyPath, &dnsProvider, &dnsProviderConfig,
			&c.Status, &notBefore, &notAfter, &renewAt, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		c.Name = name.String
		c.CertPath = certPath.String
		c.KeyPath = keyPath.String
		c.DNSProvider = dnsProvider.String
		c.DNSProviderConfig = dnsProviderConfig.String

		if c.NotBefore, err = scanCertificateTime(notBefore); err != nil {
			return nil, err
		}
		if c.NotAfter, err = scanCertificateTime(notAfter); err != nil {
			return nil, err
		}
		if c.RenewAt, err = scanCertificateTime(renewAt); err != nil {
			return nil, err
		}

		certs = append(certs, c)
	}
	return certs, nil
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
