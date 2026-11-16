package service

import "strings"


func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}