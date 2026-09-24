package collectors

import (
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// run executes a command and returns trimmed stdout.
func run(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// hostname returns the local hostname.
func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}

// ptr returns a pointer to a string.
func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ptrFloat64 returns a pointer to a float64.
func ptrFloat64(f float64) *float64 {
	return &f
}

// ptrInt returns a pointer to an int.
func ptrInt(i int) *int {
	return &i
}

// ptrBool returns a pointer to a bool.
func ptrBool(b bool) *bool {
	return &b
}

// parseInt parses an int from a string, returning 0 on error.
func parseInt(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// parseFloat parses a float from a string, returning 0 on error.
func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// parseFloatPtr parses a float from a string, returning nil on error.
func parseFloatPtr(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

// round rounds a float64 to the given decimal places.
func round(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}
