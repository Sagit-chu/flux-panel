package security

import (
	"crypto/md5"
	"fmt"
)

func MD5(input string) string {
	hash := md5.Sum([]byte(input))
	return fmt.Sprintf("%x", hash)
}
