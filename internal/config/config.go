package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	PrettyLogger bool `env:"PRETTY_LOGGER" env-default:"false"`
	HTTPServer
	Postgres
}

type HTTPServer struct {
	Addr        string        `env:"SERVER_ADDRESS" env-default:"0.0.0.0:8080"`
	OpenapiPath string        `env:"OPENAPI_PATH" env-default:"docs/openapi.yml"`
	Timeout     time.Duration `env:"HTTP_TIMEOUT" env-default:"4s"`
	IdleTimeout time.Duration `env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
}

type Postgres struct {
	PostgresConn     string `env:"POSTGRES_CONN" env-required:"true"`
	PostgresJDBCURL  string `env:"POSTGRES_JDBC_URL" env-required:"true"`
	PostgresUsername string `env:"POSTGRES_USERNAME" env-required:"true"`
	PostgresPassword string `env:"POSTGRES_PASSWORD" env-required:"true"`
	PostgresHost     string `env:"POSTGRES_HOST" env-default:"localhost"`
	PostgresPort     string `env:"POSTGRES_PORT" env-default:"5432"`
	PostgresDataBase string `env:"POSTGRES_DATABASE" env-required:"true"`
}

// MustLoad load config from environment
// variables. Panic if error occures.
func MustLoad() *Config {
	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic("cannot read environment: " + err.Error())
	}

	return &cfg
}
