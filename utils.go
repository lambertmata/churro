package churro

import (
	"slices"
	"strings"
)

func SplitString(s string, del string) []string {
	return slices.DeleteFunc(strings.Split(s, del), func(part string) bool {
		return part == ""
	})
}
