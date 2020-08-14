package storage

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDSN(t *testing.T) {
	config := Config{
		User:     "a",
		Password: "b",
		Host:     "c",
		Port:     5432,
		DBName:   "d",
	}
	expected := "user=a password=b host=c port=5432 dbname=d sslmode=disable"
	actual := config.DSN()
	require.Equal(t, expected, actual)
}
