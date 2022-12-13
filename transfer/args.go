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

// // Parse parsed the given batch request.
// func Parse(handler *Pktline) ([]string, map[string]string, error) {
// 	data, err := handler.ReadPacketList()
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("error reading batch request: %w", err)
// 	}
// 	// Logf("parse data: %d %v", len(data), data)
// 	// if len(data) < 3 {
// 	// 	return nil, nil, fmt.Errorf("invalid batch request")
// 	// }
// 	args, err := ParseArgs(data)
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("error parsing batch request: %w", err)
// 	}
// 	Logf("parse: data %d %v args %d %v", len(data), data, len(args), args)
// 	length := len(args) + 1
// 	if len(args) == len(data) {
// 		length = 0
// 	}
// 	return data[length:], args, nil
// }

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

// // ParseArgsFromHandler parses the given args.
// func ParseArgsFromHandler(handler *Pktline) (map[string]string, error) {
// 	ar, err := handler.ReadPacketListToDelim()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return ParseArgs(ar)
// }

// ArgsToList converts the given args to a list.
func ArgsToList(args map[string]string) []string {
	list := make([]string, 0)
	for key, value := range args {
		list = append(list, fmt.Sprintf("%s=%s", key, value))
	}
	return list
}
