package storage

import "fmt"

// Config defines fields used to connect to PostgreSQL
// during development sslmode is omitted and set to "disable" by default
// other values are omitted as well but specified as defaults by pgxpool.ParseConfig function
type Config struct {
	User, Password, Host string
	Port                 uint16
	DBName               string
}

// DSN constructs config string in DSN format with sslmode=disable
func (c Config) DSN() string {
	return fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=disable",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName)
}
