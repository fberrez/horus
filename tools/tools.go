package tools

import (
	"bytes"
)

// DecodeToString deletes all bytes equals to 0 in the array of bytes and
// converts the result array to a string.
func DecodeToString(datas []byte) string {
	return string(bytes.Trim(datas, "\x00"))
}
