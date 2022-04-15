package storage

import (
	"context"

	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/entities"
)

//go:generate mockgen -source=storage.go -package=storage -destination=storage_mock.go

// Common defines interface to the common persistent storage.
type Common interface {
	// GetDomains returns all domains in the system returns all domains in the system.
	// Any error returned is internal.
	GetDomains(ctx context.Context) (entities.Domains, error)
	// UpdateDomainSSL updates ssl info for domain.
	// Any error returned is internal.
	UpdateDomainSSL(ctx context.Context, id uint64, certs map[string]entities.CertInfo, region string) error
}
