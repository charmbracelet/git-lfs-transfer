package transfer

import (
	"fmt"
	"strings"
)

// batch request argument keys.
const (
	HashAlgoKey  = "hash-algo"
	TransferKey  = "transfer"
	RefnameKey   = "refname"
	ExpiresInKey = "expires-in"
	ExpiresAtKey = "expires-at"
	SizeKey      = "size"
	PathKey      = "path"
	LimitKey     = "limit"
	CursorKey    = "cursor"
)

// ParseArgs parses the given args.
func ParseArgs(lines []string) (map[string]string, error) {
	args := make(map[string]string, 0)
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid argument: %q", line)
		}
		key, value := parts[0], parts[1]
		args[key] = value
	}
	Logf("args: %d %v", len(args), args)
	return args, nil
}

// ArgsToList converts the given args to a list.
func ArgsToList(args map[string]string) []string {
	list := make([]string, 0)
	for key, value := range args {
		list = append(list, fmt.Sprintf("%s=%s", key, value))
	}
	return list
}
