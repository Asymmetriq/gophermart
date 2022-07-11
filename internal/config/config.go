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
}

const testTokenSignKey = "459116d7-fb7d-4789-8c2b-8f31dccb07cf"

var (
	_runAddress     string
	_accrualAddress string
	_databaseURI    string
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
	TokenSignKey         string `env:"-"`
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
	c.TokenSignKey = testTokenSignKey
}
