package util

import "strings"

// GenUUID 生成uuid
func GenUUID() string {

	return strings.Replace(NewV4().String(), "-", "", -1)
}
