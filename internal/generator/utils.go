package generator

import (
	"strings"
)

func firstNonEmptyString(strs ...string) string {
	for _, str := range strs {
		str = strings.TrimSpace(str)
		if str != "" {
			return str
		}
	}

	return ""
}
