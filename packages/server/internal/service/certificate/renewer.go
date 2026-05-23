package certificate

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/acme"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

// Renewer handles background certificate renewal
type Renewer struct {
	svc      *Service
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// Start begins the renewal background loop
func (r *Renewer) Start() {
	r.stopCh = make(chan struct{})
	r.wg.Add(1)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	log.Printf("[cert] renewal checker started, interval: %v", r.interval)

	// Run immediately on start
	r.checkAndRenew()

	for {
		select {
		case <-ticker.C:
			r.checkAndRenew()
		case <-r.stopCh:
			log.Printf("[cert] renewal checker stopped")
			r.wg.Done()
			return
		}
	}
}

// Stop stops the renewal background loop
func (r *Renewer) Stop() {
	close(r.stopCh)
	r.wg.Wait()
}

// checkAndRenew checks for expiring certificates and renews them
func (r *Renewer) checkAndRenew() {
	ctx := context.Background()

	certs, err := r.svc.db.ListExpiringCertificates(time.Now())
	if err != nil {
		log.Printf("[cert] failed to list expiring certificates: %v", err)
		return
	}

	if len(certs) == 0 {
		return
	}

	log.Printf("[cert] found %d certificates needing renewal", len(certs))

	for _, cert := range certs {
		r.renewCert(ctx, &cert)
	}
}

// renewCert renews a single certificate
func (r *Renewer) renewCert(ctx context.Context, cert *store.Certificate) {
	log.Printf("[cert] renewing certificate %s for domain %s", cert.ID, cert.Domain)

	// Create DNS provider from stored config
	providerConfig := decryptProviderConfig(cert.DNSProviderConfig)
	provider, err := acme.NewDNSProvider(providerConfig)
	if err != nil {
		log.Printf("[cert] failed to create DNS provider for %s: %v", cert.Domain, err)
		cert.Status = store.CertStatusFailed
		r.svc.db.UpdateCertificate(cert)
		return
	}

	// Renew the certificate
	r.svc.renewCertificate(cert, provider)
}