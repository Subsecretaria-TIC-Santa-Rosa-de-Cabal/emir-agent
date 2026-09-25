package models

import (
	"strconv"
	"strings"
)

// CompareVersions compares two semantic version strings (major.minor.patch).
// It returns:
//
//	-1 if a < b
//	 0 if a == b
//	 1 if a > b
//
// Non-semver inputs are compared lexicographically as a fallback.
func CompareVersions(a, b string) int {
	a = normalizeVersion(a)
	b = normalizeVersion(b)

	pa := parseVersion(a)
	pb := parseVersion(b)

	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return strings.Compare(a, b)
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func parseVersion(v string) [3]int {
	parts := strings.Split(v, ".")
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return result
		}
		result[i] = n
	}
	return result
}
