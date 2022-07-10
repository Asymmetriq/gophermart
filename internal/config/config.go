package config

import (
	"flag"
)

func init() {
	flag.StringVar(&_run_address, "a", ":8080", "Server's host address")
	flag.StringVar(&_accrual_address, "r", "", "External accrual address")
	flag.StringVar(&_databaseURI, "d", "", "Database URI")
}

const testTokenSignKey = "459116d7-fb7d-4789-8c2b-8f31dccb07cf"

var (
	_run_address     string
	_accrual_address string
	_databaseURI     string
)

type Config interface {
	GetRunAddress() string
	GetAccrualAddress() string
	GetDatabaseURI() string
	GetTokenSignKey() string
}

type config struct {
	runAddress           string `env:"RUN_ADDRESS"`
	accrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	databaseURI          string `env:"DATABASE_URI"`
	tokenSignKey         string
}

func InitConfig() Config {
	flag.Parse()
	conf := &config{
		runAddress:           _run_address,
		accrualSystemAddress: _accrual_address,
		databaseURI:          _databaseURI,
		tokenSignKey:         testTokenSignKey,
	}
	return conf
}

// Getters
func (c *config) GetRunAddress() string {
	return c.runAddress
}

func (c *config) GetAccrualAddress() string {
	return c.accrualSystemAddress
}

func (c *config) GetDatabaseURI() string {
	return c.databaseURI
}
func (c *config) GetTokenSignKey() string {
	return c.tokenSignKey
}
