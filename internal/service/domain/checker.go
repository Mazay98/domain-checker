package domain

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/geozo-tech/go-curl"
	"go.uber.org/zap"

	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/entities"
	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/storage"
)

const parseDateFormat = "Jan 2 15:04:05 2006 MST"

var (
	errMissingExpireDate = fmt.Errorf("expire date is not found in cert info")
	errNoCertInfo        = fmt.Errorf("no cert info")
	selfSignedPattern    = regexp.MustCompile(`(CN = ISRG Root)|(R\d)`)
)

// Service is designed to work with domains.
type Service struct {
	storage storage.Common
	logger  *zap.Logger
	region  string
}

// New returns new Service ready to use.
func New(storage storage.Common, logger *zap.Logger, region string) Service {
	return Service{
		storage: storage,
		logger:  logger,
		region:  region,
	}
}

// UpdateDomains update ssl for domains.
func (s Service) UpdateDomains(ctx context.Context) error {
	domains, err := s.storage.GetDomains(ctx)
	if err != nil {
		return fmt.Errorf("failed to get domains list: %w", err)
	}

	for _, domain := range domains { //nolint:gocritic
		ipList, err := net.LookupIP(domain.Name)
		if err != nil {
			s.logger.Error(
				"failed to get domain's balancers list",
				zap.String("domain", domain.Name),
				zap.Error(err),
			)
			continue
		}

		sslConfig := s.getCertList(domain.Name, ipList)
		if len(sslConfig) == 0 {
			continue
		}

		if err := s.storage.UpdateDomainSSL(ctx, domain.ID, sslConfig, s.region); err != nil {
			return fmt.Errorf("failed to update %q ssl info: %w", domain.Name, err)
		}
	}

	return nil
}

// getCertList return list of certs holds IP -> entities.CertInfo.
func (s Service) getCertList(domain string, ipList []net.IP) map[string]entities.CertInfo {
	certList := make(map[string]entities.CertInfo)
	for _, ip := range ipList {
		sIP := ip.String()
		certInfo, err := getSSLInfo(sIP, domain)
		if err != nil {
			s.logger.Error("failed to find certificate info",
				zap.String("domain", domain),
				zap.String("ip", sIP),
				zap.Error(err),
			)
			continue
		}

		certList[sIP] = certInfo
		s.logger.Info("updated domain's certificate info", zap.String("domain", domain))
	}

	return certList
}

// getSSLInfo get ssl info.
func getSSLInfo(IP string, domain string) (entities.CertInfo, error) { //nolint:gocritic
	easy := curl.EasyInit()
	defer easy.Cleanup()

	if err := easy.Setopt(curl.OPT_URL, "https://"+domain); err != nil {
		return entities.CertInfo{}, fmt.Errorf("failed append param url: %w", err)
	}
	if err := easy.Setopt(curl.OPT_CONNECT_TO, []string{fmt.Sprintf("%s:443:%s:443", domain, IP)}); err != nil {
		return entities.CertInfo{}, fmt.Errorf("failed append param connect to: %w", err)
	}
	if err := easy.Setopt(curl.OPT_SSL_VERIFYPEER, true); err != nil {
		return entities.CertInfo{}, fmt.Errorf("failed append param verifypeer: %w", err)
	}
	if err := easy.Setopt(curl.OPT_SSL_VERIFYHOST, true); err != nil {
		return entities.CertInfo{}, fmt.Errorf("failed append param verifyhost: %w", err)
	}
	if err := easy.Setopt(curl.OPT_TIMEOUT, 5); err != nil {
		return entities.CertInfo{}, fmt.Errorf("failed append param timeout: %w", err)
	}
	if err := easy.Setopt(curl.OPT_CERTINFO, true); err != nil {
		return entities.CertInfo{}, fmt.Errorf("failed append param certinfo: %w", err)
	}
	if err := easy.Setopt(curl.OPT_NOPROGRESS, true); err != nil {
		return entities.CertInfo{}, fmt.Errorf("failed append param noprogress: %w", err)
	}
	if err := easy.Setopt(curl.OPT_NOBODY, true); err != nil {
		return entities.CertInfo{}, fmt.Errorf("failed append param nobody: %w", err)
	}
	if err := easy.Perform(); err != nil {
		return entities.CertInfo{}, fmt.Errorf("failed to send curl: %w", err)
	}

	info, err := easy.Getinfo(curl.INFO_CERTINFO)
	if err != nil {
		return entities.CertInfo{}, fmt.Errorf("failed to get info: %w", err)
	}

	switch certs := info.(type) {
	case []string:
		return getCertInfo(certs)
	default:
		return entities.CertInfo{}, errors.New("unsupported certificate info format") //nolint:goerr113
	}
}

func getCertInfo(certs []string) (entities.CertInfo, error) {
	lc := len([]byte("Expire date")) + 1
	for _, cert := range certs {
		matchExpiredAt := strings.Index(cert, "Expire date")
		if matchExpiredAt == -1 {
			return entities.CertInfo{}, errMissingExpireDate
		}

		certT := strings.Split(cert[matchExpiredAt+lc:], "\n")[0]

		expiredAt, err := time.Parse(parseDateFormat, certT)
		if err != nil {
			return entities.CertInfo{}, fmt.Errorf("parse date error: %w", err)
		}

		isSelfSignedCert := selfSignedPattern.MatchString(strings.Split(cert, "\n")[0])
		if expiredAt.Before(time.Now()) || isSelfSignedCert {
			return entities.CertInfo{
				ExpiredAt: &expiredAt,
				Valid:     false,
			}, nil
		}

		return entities.CertInfo{
			ExpiredAt: &expiredAt,
			Valid:     true,
		}, nil
	}

	return entities.CertInfo{}, errNoCertInfo
}
