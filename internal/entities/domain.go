package entities

import (
	"time"
)

// Domain represents a single domain.
type Domain struct {
	SSL  SSL
	Name string
	ID   uint64
}

// CertInfo is a ssl settings of balancer.
type CertInfo struct {
	ExpiredAt *time.Time `json:"expired_at"`
	Valid     bool       `json:"valid"`
}

// Domains holds Domain.ID -> Domain reference.
type Domains map[uint64]Domain

// SSL holds geo -> balancer ip -> certificate info.
type SSL map[string]map[string]CertInfo
