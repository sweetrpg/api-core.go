package util

import (
	"os"
	"strconv"
)

// GetEnv returns the value of the named environment variable, or fallback when it is unset or
// empty.
func GetEnv(name, fallback string) string {
	if v, ok := os.LookupEnv(name); ok && v != "" {
		return v
	}
	return fallback
}

// GetEnvInt returns the named environment variable parsed as an int, or fallback when it is
// unset, empty, or not a valid integer.
func GetEnvInt(name string, fallback int) int {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
