package domain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/geozo-tech/go-curl"
	"go.uber.org/zap"

	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/config"
	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/entities"
	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/storage"
)

const (
	parseDateFormat = "Jan 2 15:04:05 2006 MST"
	adBlockBansList = "https://easylist-downloads.adblockplus.org/ruadlist+easylist.txt"
)

var (
	errMissingExpireDate = fmt.Errorf("expire date is not found in cert info")
	errNoCertInfo        = fmt.Errorf("no cert info")
	selfSignedPattern    = regexp.MustCompile(`(CN = ISRG Root)|(R\d)`)
)

// Service is designed to work with domains.
type Service struct {
	storage   storage.Common
	logger    *zap.Logger
	balancers config.Balancers
}

// New returns new Service ready to use.
func New(storage storage.Common, logger *zap.Logger, balancers config.Balancers) (Service, error) {
	if !isValidIPList(balancers.Sg) || !isValidIPList(balancers.Ru) {
		return Service{}, errors.New("invalid balancers") //nolint:goerr113
	}
	return Service{
		storage:   storage,
		logger:    logger,
		balancers: balancers,
	}, nil
}

func isValidIPList(ipList []string) bool {
	for _, ip := range ipList {
		if net.ParseIP(ip) == nil {
			return false
		}
	}

	return true
}

// UpdateDomains update ssl for domains.
func (s Service) UpdateDomains(ctx context.Context) error {
	domains, err := s.storage.GetDomains(ctx)
	if err != nil {
		return fmt.Errorf("failed to get domains list: %w", err)
	}

	if len(s.balancers.Ru) == 0 && len(s.balancers.Sg) == 0 {
		s.logger.Info("empty balancer list")
		return nil
	}

	for _, domain := range domains { //nolint:gocritic
		sslConfig := entities.SSL{
			"ru": s.getCertList(domain.Name, s.balancers.Ru),
			"sg": s.getCertList(domain.Name, s.balancers.Sg),
		}

		if err := s.storage.UpdateDomainSSL(ctx, domain.ID, sslConfig); err != nil {
			return fmt.Errorf("failed to update %q ssl info: %w", domain.Name, err)
		}
	}

	return nil
}

// getCertList return list of certs holds IP -> entities.CertInfo.
func (s Service) getCertList(domain string, ipList []string) map[string]entities.CertInfo {
	certList := make(map[string]entities.CertInfo)
	for _, ip := range ipList {
		certInfo, err := getSSLInfo(ip, domain)
		if err != nil {
			s.logger.Error("failed to find certificate info",
				zap.String("domain", domain),
				zap.String("ip", ip),
				zap.Error(err),
			)
			continue
		}

		certList[ip] = certInfo
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

// BanDomains bans domains that are found in easylist.
func (s *Service) BanDomains(ctx context.Context) (int, error) {
	domains, err := s.storage.GetDomains(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get domains list: %w", err)
	}

	easylistBannedDomains, err := getEasyList()
	if err != nil {
		return 0, fmt.Errorf("failed to get easylist: %w", err)
	}

	bannedDomains := findDomainsBanedByEasylist(domains, easylistBannedDomains)

	if len(bannedDomains) == 0 {
		return 0, nil
	}

	if err = s.storage.BanDomainsByIDs(ctx, bannedDomains); err != nil {
		return 0, fmt.Errorf("failed to ban domains: %w", err)
	}

	return len(bannedDomains), nil
}

func getEasyList() ([]byte, error) {
	resp, err := http.Get(adBlockBansList)
	if err != nil {
		return nil, fmt.Errorf("failed to get easylist info: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read easylist response: %w", err)
	}

	return b, nil
}

func findDomainsBanedByEasylist(domains entities.Domains, easylistBannedDomains []byte) []uint64 {
	bannedIds := make([]uint64, 0)

	for id := range domains {
		domain := domains[id]
		if r := bytes.Index(easylistBannedDomains, []byte(domain.Name)); r != -1 {
			bannedIds = append(bannedIds, domain.ID)
		}
	}

	return bannedIds
}
