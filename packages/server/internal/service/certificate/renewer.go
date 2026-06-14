package certificate

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

// Renewer periodically checks for local-CA certificates that are within
// renewOffsetDays of expiration and re-signs them.
type Renewer struct {
	svc      *Service
	interval time.Duration

	mu     sync.Mutex
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func (r *Renewer) run() {
	r.mu.Lock()
	if r.stopCh != nil {
		r.mu.Unlock()
		return
	}
	r.stopCh = make(chan struct{})
	stopCh := r.stopCh
	r.wg.Add(1)
	r.mu.Unlock()

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	defer r.wg.Done()

	log.Printf("[cert] renewer started, interval=%s", r.interval)
	r.scan()

	for {
		select {
		case <-ticker.C:
			r.scan()
		case <-stopCh:
			log.Printf("[cert] renewer stopped")
			return
		}
	}
}

func (r *Renewer) stop() {
	r.mu.Lock()
	if r.stopCh != nil {
		close(r.stopCh)
		r.stopCh = nil
	}
	r.mu.Unlock()
	r.wg.Wait()
}

func (r *Renewer) scan() {
	certs, err := r.svc.db.ListExpiringLocalCertificates(time.Now())
	if err != nil {
		log.Printf("[cert] list expiring: %v", err)
		return
	}
	for i := range certs {
		c := certs[i]
		if _, err := r.svc.Resign(c.ID); err != nil {
			log.Printf("[cert] re-sign %s (%s) failed: %v", c.Domain, c.ID, err)
			continue
		}
		updated, _ := r.svc.db.GetCertificate(c.ID)
		if updated != nil {
			_ = context.Background()
			log.Printf("[cert] re-signed %s, new NotAfter=%s", c.Domain, updated.NotAfter.Format(time.RFC3339))
		}
	}
}

// Helper that keeps the unused import set stable for build tags.
var _ = store.CertStatusActive
