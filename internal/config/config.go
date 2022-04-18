package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/jessevdk/go-flags"

	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/environment"
)

type (
	// AppConfig contains full configuration of the service.
	AppConfig struct {
		Env    environment.Env `long:"env" env:"ENV" description:"Environment application is running in" default:"local"`
		Region string          `long:"region" env:"REGION" description:"Region application is running in" default:"ru"`

		Logger   Logger   `group:"Logger options" namespace:"logger" env-namespace:"LOGGER"`
		Postgres Postgres `group:"PostgreSQL option" namespace:"postgres" env-namespace:"POSTGRES"`
		HTTP     Server   `group:"HTTP server options" namespace:"http" env-namespace:"HTTP"`
		Tickers  Tickers  `group:"Tickers options" namespace:"tickers" env-namespace:"TICKER"`

		EnableEasylist bool `long:"enable_easylist" env:"ENABLE_EASYLIST" description:"Check for EasyList enabled"`
	}

	// Tickers struct of timi duration tickers.
	Tickers struct {
		SSLChecker      time.Duration `long:"ssl_checker_duration" env:"SSL_CHECKER" description:"Time for tick ssl checker daemon" default:"10m"`
		EasyListChecker time.Duration `long:"easylist_checker_duration" env:"EASYLIST_CHECKER" description:"Time for tick easylist checker daemon" default:"5m"` //nolint:lll
	}

	// Logger contains logger configuration.
	Logger struct {
		Level string `long:"level" env:"LEVEL" description:"Log level to use; environment-base level is used when empty"`
	}

	// Server contains server configuration, regardless
	// of the server type http.
	Server struct {
		Host string `long:"host" env:"HOST" description:"Host to listen on, default is empty (all interfaces)"`
		Port int    `long:"port" env:"PORT" description:"Port to listen on" required:"true"`
	}

	// Postgres contains postgres configuration.
	Postgres struct {
		MainDBConnectionString string        `long:"maindb_connection_string" env:"MAINDB_CONNECTION_STRING" description:"PGX connection string to the maindDB" required:"true"` //nolint:lll
		Timeout                time.Duration `long:"timeout" env:"TIMEOUT" description:"Timeout for queries" default:"1s"`
	}
)

// ErrHelp is returned when --help flag is
// used and application should not launch.
var ErrHelp = errors.New("help")

// New reads flags and envs and returns AppConfig
// that corresponds to the values read.
func New() (*AppConfig, error) {
	var config AppConfig
	if _, err := flags.Parse(&config); err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && flagsErr.Type == flags.ErrHelp {
			return nil, ErrHelp
		}
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}
