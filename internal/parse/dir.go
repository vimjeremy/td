package parse

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands a leading "~" or "~/..." to the home directory, and a path
// that starts with an environment variable ($HOME/x, ${PROJECTS}/x) using the
// current environment. Anything else is returned as typed (trimmed). A
// variable that is not set is an error rather than silently becoming empty.
func ExpandPath(s string) (string, error) {
	s = strings.TrimSpace(s)
	switch {
	case s == "~" || strings.HasPrefix(s, "~/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, s[1:]), nil
	case len(s) > 1 && s[0] == '$':
		missing := ""
		out := os.Expand(s, func(name string) string {
			v, ok := os.LookupEnv(name)
			if !ok && missing == "" {
				missing = name
			}
			return v
		})
		if missing != "" {
			return "", fmt.Errorf("environment variable $%s is not set", missing)
		}
		return out, nil
	}
	return s, nil
}

// ExpandDir is ExpandPath plus making the result absolute, so "." and relative
// paths use the current directory. It does not check that the directory exists.
func ExpandDir(s string) (string, error) {
	p, err := ExpandPath(s)
	if err != nil {
		return "", err
	}
	return filepath.Abs(p)
}
