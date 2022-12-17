package config

import (
	"flag"
	"nats-tracing/internal/logger"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"gopkg.in/validator.v2"
)

type Config struct {
	ServiceID       string `validate:"nonzero"`
	ServiceVersion  string `validate:"nonzero"`
	ServiceName     string `validate:"nonzero"`
	EnvName         string `validate:"nonzero"`
	NatsServer      string `validate:"nonzero" split_words:"true"`
	JaegerAgentHost string `validate:"nonzero" split_words:"true"`
	JaegerAgentPort string `validate:"nonzero" split_words:"true"`
}

func (cfg *Config) Validate() error {
	return validator.Validate(cfg)
}

func (cfg *Config) Print() {
	log := logger.Get()

	log.Info("config",
		logger.String("ServiceVersion", cfg.ServiceVersion),
		logger.String("ServiceName", cfg.ServiceName),
		logger.String("EnvName", cfg.EnvName),
		logger.String("NatsServer", cfg.NatsServer),
		logger.String("JaegerAgentHost", cfg.JaegerAgentHost),
		logger.String("JaegerAgentPort", cfg.JaegerAgentPort),
	)
}

const (
	versionFlagName = "version"
	serviceFlagName = "service"
	idFlagName      = "id"
	envFlagName     = "env"
)

func New(envPrefix string) (*Config, error) {
	cfg := new(Config)
	flag.StringVar(&cfg.ServiceVersion, versionFlagName, "", "service version")
	flag.StringVar(&cfg.ServiceName, serviceFlagName, "", "service name")
	flag.StringVar(&cfg.EnvName, envFlagName, "development", "environment: development")
	flag.StringVar(&cfg.ServiceID, idFlagName, "", "service id")
	flag.Parse()

	err := envconfig.Process(envPrefix, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "envconfig.Process")
	}

	return cfg, nil
}
