package certificate

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pallyoung/auth-gate/packages/server/internal/localca"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

type fakeReloader struct{ called int }

func (r *fakeReloader) Reload() { r.called++ }

func newTestSetup(t *testing.T) (*Service, *localca.CA, store.Store, *fakeReloader) {
	t.Helper()
	dir := t.TempDir()
	db, err := store.NewJSONStore(dir)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ca, err := localca.LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("LoadOrCreate: %v", err)
	}
	// Register the CA in the database so Resign/ProvisionLocal can stamp ca_id.
	if err := db.CreateCACertificate(&store.CACertificate{
		ID: "ca-default", Name: ca.Cert.Subject.CommonName,
		CertPEM:   string(ca.CertPEM),
		KeyPEM:    string(ca.KeyPEM),
		NotBefore: ca.Cert.NotBefore, NotAfter: ca.Cert.NotAfter,
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateCACertificate: %v", err)
	}

	rel := &fakeReloader{}
	svc, err := NewService(db, Config{DataDir: dir, CA: ca}, rel)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc, ca, db, rel
}

func TestService_ProvisionLocal(t *testing.T) {
	svc, _, db, rel := newTestSetup(t)

	cert, err := svc.ProvisionLocal(context.Background(), "wildcard", "*.example.com", nil)
	if err != nil {
		t.Fatalf("ProvisionLocal: %v", err)
	}
	if cert.Status != store.CertStatusActive {
		t.Errorf("Status = %q, want active", cert.Status)
	}
	if cert.Source != store.SourceLocalCA {
		t.Errorf("Source = %q, want local_ca", cert.Source)
	}
	if cert.CAID == "" {
		t.Error("CAID should be populated")
	}
	if cert.CertPath == "" || cert.KeyPath == "" {
		t.Fatal("expected cert/key paths set")
	}
	for _, p := range []string{cert.CertPath, cert.KeyPath} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("file %s missing: %v", p, err)
		}
	}
	time.Sleep(50 * time.Millisecond) // let triggerReload goroutine run
	if rel.called == 0 {
		t.Error("expected reloader to fire")
	}

	// verify DB row matches
	row, _ := db.GetCertificate(cert.ID)
	if row == nil {
		t.Fatal("cert not in DB")
	}
	if !row.RenewAt.After(time.Now()) {
		t.Error("RenewAt should be in the future")
	}
}

func TestService_ProvisionLocal_DuplicateDomain(t *testing.T) {
	svc, _, _, _ := newTestSetup(t)
	if _, err := svc.ProvisionLocal(context.Background(), "first", "foo.example.com", nil); err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err := svc.ProvisionLocal(context.Background(), "second", "foo.example.com", nil)
	if err == nil || Code(err) != ErrCodeDomainExists {
		t.Fatalf("expected domain_exists, got %v", err)
	}
}

func TestService_Import_Valid(t *testing.T) {
	svc, _, db, _ := newTestSetup(t)
	certPEM, keyPEM := mintTestCert(t, "imported.example.com")

	cert, err := svc.Import(context.Background(), "imported", "imported.example.com", certPEM, keyPEM)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if cert.Source != store.SourceImported {
		t.Errorf("Source = %q, want imported", cert.Source)
	}
	if cert.CAID != "" {
		t.Errorf("CAID should be empty for imported, got %q", cert.CAID)
	}
	if !cert.RenewAt.IsZero() {
		t.Errorf("imported cert should not have RenewAt set, got %v", cert.RenewAt)
	}
	if _, err := db.GetCertificate(cert.ID); err != nil {
		t.Errorf("GetCertificate: %v", err)
	}
}

func TestService_Import_DomainMismatch(t *testing.T) {
	svc, _, _, _ := newTestSetup(t)
	certPEM, keyPEM := mintTestCert(t, "actual.example.com")

	_, err := svc.Import(context.Background(), "wrong", "expected.example.com", certPEM, keyPEM)
	if err == nil || Code(err) != ErrCodeDomainMismatch {
		t.Fatalf("expected domain_mismatch, got %v", err)
	}
}

func TestService_Import_BadPEM(t *testing.T) {
	svc, _, _, _ := newTestSetup(t)
	_, err := svc.Import(context.Background(), "bad", "x.example.com", "not pem", "not pem")
	if err == nil || Code(err) != ErrCodeInvalidPEM {
		t.Fatalf("expected invalid_pem, got %v", err)
	}
}

func TestService_Import_KeyMismatch(t *testing.T) {
	svc, _, _, _ := newTestSetup(t)
	certPEM, _ := mintTestCert(t, "x.example.com")
	_, otherKey := mintTestCert(t, "y.example.com")

	_, err := svc.Import(context.Background(), "mismatch", "x.example.com", certPEM, otherKey)
	if err == nil || Code(err) != ErrCodeInvalidPEM {
		t.Fatalf("expected invalid_pem (key mismatch), got %v", err)
	}
}

func TestService_Resign_LocalCA(t *testing.T) {
	svc, _, db, rel := newTestSetup(t)
	cert, _ := svc.ProvisionLocal(context.Background(), "wildcard", "*.example.com", nil)
	originalNotAfter := cert.NotAfter
	row, _ := db.GetCertificate(cert.ID)

	rel.called = 0
	resigned, err := svc.Resign(cert.ID)
	if err != nil {
		t.Fatalf("Resign: %v", err)
	}
	if !resigned.NotAfter.After(originalNotAfter) {
		t.Errorf("expected NotAfter to advance, got %v (was %v)", resigned.NotAfter, originalNotAfter)
	}
	time.Sleep(50 * time.Millisecond)
	if rel.called == 0 {
		t.Error("reloader should fire on Resign")
	}
	// File should be re-written with new content
	data, _ := os.ReadFile(resigned.CertPath)
	if block, _ := pem.Decode(data); block == nil {
		t.Fatal("cert.pem should be valid PEM")
	} else if cert, err := x509.ParseCertificate(block.Bytes); err != nil {
		t.Errorf("cert.pem should parse: %v", err)
	} else if cert.Subject.CommonName != row.Domain {
		t.Errorf("cert Subject.CN = %q, want %q", cert.Subject.CommonName, row.Domain)
	}
}

func TestService_Resign_ImportedRejected(t *testing.T) {
	svc, _, _, _ := newTestSetup(t)
	certPEM, keyPEM := mintTestCert(t, "imported.example.com")
	imported, err := svc.Import(context.Background(), "imp", "imported.example.com", certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Resign(imported.ID)
	if err == nil || Code(err) != ErrCodeImportedCannotResign {
		t.Fatalf("expected imported_cannot_resign, got %v", err)
	}
}

func TestService_DeleteRemovesFiles(t *testing.T) {
	svc, _, db, _ := newTestSetup(t)
	cert, _ := svc.ProvisionLocal(context.Background(), "wildcard", "*.example.com", nil)

	if _, err := os.Stat(cert.CertPath); err != nil {
		t.Fatalf("cert file should exist: %v", err)
	}
	if err := svc.Delete(cert.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(cert.CertPath); !os.IsNotExist(err) {
		t.Errorf("cert file should be removed, got err=%v", err)
	}
	if row, _ := db.GetCertificate(cert.ID); row != nil {
		t.Error("cert should be gone from DB")
	}
}

func TestService_RenewerPicksUpExpiring(t *testing.T) {
	svc, _, db, _ := newTestSetup(t)
	cert, _ := svc.ProvisionLocal(context.Background(), "wildcard", "*.example.com", nil)

	// Force the renew_at into the past and trigger a manual scan via Resign
	// to simulate what the renewer would do.
	cert.RenewAt = time.Now().Add(-time.Hour)
	if err := db.UpdateCertificate(cert); err != nil {
		t.Fatal(err)
	}
	expiring, err := db.ListExpiringLocalCertificates(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(expiring) != 1 {
		t.Fatalf("expected 1 expiring, got %d", len(expiring))
	}
	if _, err := svc.Resign(expiring[0].ID); err != nil {
		t.Fatalf("Resign: %v", err)
	}
}

func TestMatchDomain(t *testing.T) {
	cases := []struct {
		cert, request string
		want           bool
	}{
		{"foo.example.com", "foo.example.com", true},
		{"*.example.com", "foo.example.com", true},
		{"*.example.com", "bar.example.com", true},
		{"*.example.com", "example.com", false},
		{"*.example.com", "a.b.example.com", false},
		{"*.example.com", "foo.other.com", false},
		{"foo.example.com", "bar.example.com", false},
	}
	for _, c := range cases {
		if got := matchDomain(c.cert, c.request); got != c.want {
			t.Errorf("matchDomain(%q, %q) = %v, want %v", c.cert, c.request, got, c.want)
		}
	}
}

// Helpers

func mintTestCert(t *testing.T, domain string) (certPEM, keyPEM string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: domain},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(30 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{domain},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	keyPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}))
	return
}

var _ = uuid.NewString
