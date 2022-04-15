package domain

import (
	"io/ioutil"
	"net"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/entities"
)

func TestGetSSLInfo(t *testing.T) {
	t.Parallel()

	dn := "google.com"

	t.Run("check get cert google.com", func(t *testing.T) {
		t.Parallel()

		il, err := net.LookupIP(dn)
		if err != nil {
			t.Fatalf("get ip list failed: %s", err.Error())
		}
		require.NoError(t, err)

		ci, err := getSSLInfo(il[0].String(), dn)

		require.True(t, ci.Valid)
		require.NotEmpty(t, ci.ExpiredAt)
		require.NoError(t, err)
	})

	t.Run("check failed to get cert on bad ip", func(t *testing.T) {
		t.Parallel()

		_, err := getSSLInfo("0.0.0.0", dn)
		require.Error(t, err)
	})
}

func TestGetCertInfo(t *testing.T) {
	t.Parallel()

	edp := "Jan 2 15:04:05 2006 MSK"
	edt := time.Now().Add(time.Minute).Format(edp)

	t.Run("check for google.com has cert", func(t *testing.T) {
		t.Parallel()

		c := []string{"Subject:CN = google.com.com\nExpire date:" + edt}
		ci, err := getCertInfo(c)

		require.NoError(t, err)
		require.True(t, ci.Valid)
		require.Equal(t, edt, ci.ExpiredAt.Format(edp))
	})
	t.Run("check for missing expire date", func(t *testing.T) {
		t.Parallel()

		c := []string{"Subject:CN = google.com.com"}
		ci, err := getCertInfo(c)

		require.Nil(t, ci.ExpiredAt)
		require.Error(t, err, errMissingExpireDate)
	})
	t.Run("check for pattern expire date", func(t *testing.T) {
		t.Parallel()

		c := []string{"Subject:CN = google.com.com\nExpire date:" + "01-02-2006 15:04:05"}
		ci, err := getCertInfo(c)

		require.Nil(t, ci.ExpiredAt)
		require.Error(t, err)
	})
	t.Run("check for self signed", func(t *testing.T) {
		t.Parallel()

		c := []string{"Subject:CN = ISRG Root\nExpire date:" + edt}
		ci, err := getCertInfo(c)

		require.NoError(t, err)
		require.False(t, ci.Valid)
		require.Equal(t, edt, ci.ExpiredAt.Format(edp))
	})
	t.Run("check for empty cert", func(t *testing.T) {
		t.Parallel()

		c := []string{""}
		ci, err := getCertInfo(c)

		require.Equal(t, ci, entities.CertInfo{})
		require.Error(t, err, errNoCertInfo)
	})
	t.Run("check for real conclusion", func(t *testing.T) {
		t.Parallel()

		_, currFile, _, ok := runtime.Caller(0)
		if !ok {
			t.Fatalf("failed to get current file location")
		}

		ce := filepath.Join(currFile,
			filepath.Join("..", "testdata", "certinfo.txt"))

		d, err := ioutil.ReadFile(ce)
		require.NoError(t, err)

		sd := string(d)
		eds := "Expire date:"
		regex := regexp.MustCompile(eds + "(.*)\n")
		c := []string{regex.ReplaceAllString(sd, eds+edt+"\n")}
		ci, err := getCertInfo(c)

		require.NoError(t, err)
		require.True(t, ci.Valid)
		require.Equal(t, edt, ci.ExpiredAt.Format(edp))
	})
}
