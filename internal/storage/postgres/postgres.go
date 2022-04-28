package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"

	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/config"
	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/entities"
)

// Storage is a postgres-based implementation of storage.Common.
type Storage struct {
	mainDB *pgxpool.Pool
	logger *zap.Logger
}

// New connects to postgres. Context is used during dial only,
// connString may contain pgx specific parameters.
func New(ctx context.Context, logger *zap.Logger, conf *config.Postgres) (Storage, error) {
	mainDB, err := pgxpool.Connect(ctx, conf.MainDBConnectionString)
	if err != nil {
		return Storage{}, fmt.Errorf("failed to create mainDB pgx pool: %w", err)
	}

	return Storage{
		mainDB: mainDB,
		logger: logger,
	}, nil
}

// GetDomains returns all sites in the system.
// Any error returned is internal.
func (s *Storage) GetDomains(ctx context.Context) (entities.Domains, error) {
	rows, err := s.mainDB.Query(ctx, `
		SELECT
			id,
			name,
			ssl
		FROM
			system.domains
		WHERE
			deleted_at IS NULL
			AND banned_at IS NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query domains: %w", err)
	}
	defer rows.Close()

	domains := make(entities.Domains)
	for rows.Next() {
		var domain entities.Domain
		if err := rows.Scan(
			&domain.ID,
			&domain.Name,
			&domain.SSL,
		); err != nil {
			return nil, fmt.Errorf("failed to scan domain: %w", err)
		}

		domains[domain.ID] = domain
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read domain list: %w", err)
	}

	return domains, nil
}

// UpdateDomainSSL updates ssl info for domain.
// Any error returned is internal.
func (s *Storage) UpdateDomainSSL(
	ctx context.Context,
	id uint64,
	ssl entities.SSL,
) error {
	_, err := s.mainDB.Exec(ctx, `
		UPDATE
			system.domains
		SET
			ssl = $1
		WHERE
			id = $2
	`, ssl, id)
	if err != nil {
		return fmt.Errorf("failed to update domain: %w", err)
	}

	return nil
}

// BanDomainsByIDs banned domains from id List.
// Any error returned is internal.
func (s *Storage) BanDomainsByIDs(ctx context.Context, domainIDList []uint64) error {
	_, err := s.mainDB.Exec(ctx, `
		UPDATE
			system.domains
		SET
			banned_at = $1,
			is_distributed = $2
		WHERE
			id = any($3)
	`, time.Now(), false, domainIDList)
	if err != nil {
		return fmt.Errorf("failed to update domains: %w", err)
	}

	return nil
}

// Close releases underlying db resources.
func (s *Storage) Close() error {
	s.mainDB.Close()
	return nil
}
