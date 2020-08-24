package storage

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"time"
)

// Option alters the default configuration of the pgxpool.Config used during new Store construction
type Option interface {
	apply(*pgxpool.Config)
}

type optionFunc func(c *pgxpool.Config)

func (f optionFunc) apply(c *pgxpool.Config) { f(c) }

// ConnectionTimeout sets timeout for connection to be established
func ConnectionTimeout(d time.Duration) Option {
	return optionFunc(func(c *pgxpool.Config) {
		c.ConnConfig.ConnectTimeout = d
	})
}
