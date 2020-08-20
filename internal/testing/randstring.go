package testing

import (
	"github.com/rs/xid"
)

// RandString generates random string using github.com/rs/xid package
func RandString() string {
	guid := xid.New()

	return guid.String()
}
