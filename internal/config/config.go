package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env"
)

func init() {
	flag.StringVar(&_runAddress, "a", ":8080", "Server's host address")
	flag.StringVar(&_accrualAddress, "r", "", "External accrual address")
	flag.StringVar(&_databaseURI, "d", "", "Database URI")
	flag.StringVar(&_tokenSignKey, "k", "459116d7-fb7d-4789-8c2b-8f31dccb07cf", "Token sign key")
}

var (
	_runAddress     string
	_accrualAddress string
	_databaseURI    string
	_tokenSignKey   string
)

type Config interface {
	GetRunAddress() string
	GetAccrualAddress() string
	GetDatabaseURI() string
	GetTokenSignKey() string
}

type config struct {
	RunAddress           string `env:"RUN_ADDRESS"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	DatabaseURI          string `env:"DATABASE_URI"`
	TokenSignKey         string `env:"TOKEN_SIGN_KEY" envDefault:"459116d7-fb7d-4789-8c2b-8f31dccb07cf"`
}

func InitConfig() Config {
	flag.Parse()
	conf := &config{}
	if err := env.Parse(conf); err != nil {
		log.Fatalf("missing required env variables: %v", err)
	}
	conf.initWithFlags()
	return conf
}

// Getters
func (c *config) GetRunAddress() string {
	return c.RunAddress
}

func (c *config) GetAccrualAddress() string {
	return c.AccrualSystemAddress
}

func (c *config) GetDatabaseURI() string {
	return c.DatabaseURI
}
func (c *config) GetTokenSignKey() string {
	return c.TokenSignKey
}

func (c *config) initWithFlags() {
	if len(c.RunAddress) == 0 {
		c.RunAddress = _runAddress
	}
	if len(c.AccrualSystemAddress) == 0 {
		c.AccrualSystemAddress = _accrualAddress
	}
	if len(c.DatabaseURI) == 0 {
		c.DatabaseURI = _databaseURI
	}
	if len(c.TokenSignKey) == 0 {
		c.DatabaseURI = _tokenSignKey
	}
}
