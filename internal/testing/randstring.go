package testing

import (
	"github.com/rs/xid"
)

// RandString generates random string with 10 symbols length from lower- and uppercase alphabet
func RandString() string {
	guid := xid.New()

	return guid.String()
}
