package testing

import (
	"math/rand"
	"strings"
)

// RandString generates random string with 10 symbols length from lower- and uppercase alphabet
func RandString() string {
	var out strings.Builder
	charSet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	length := 10
	for i := 0; i < length; i++ {
		random := rand.Intn(len(charSet))
		randomChar := charSet[random]
		out.WriteString(string(randomChar))
	}
	return out.String()
}
