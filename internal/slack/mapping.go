package slack

import (
	"strings"

	"github.com/pkg/errors"
)

// ParseUserMappings parses the slice users as key-value pairs separated with an
// equal (=) sign.
//
// If any of the provided mappings are invalid or conflicting mappings are
// provided an error is returned.
func ParseUserMappings(users []string) (map[string]string, error) {
	m := make(map[string]string)
	for _, u := range users {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		s := strings.Split(u, "=")
		if len(s) != 2 {
			return nil, errors.Errorf("invalid user mapping '%s'", u)
		}
		src := strings.TrimSpace(s[0])
		dest := strings.TrimSpace(s[1])
		_, exist := m[src]
		if exist {
			return nil, errors.Errorf("conflicting user mappings for %s", src)
		}
		if src == "" || dest == "" {
			return nil, errors.Errorf("invalid user mapping '%s'", u)
		}
		m[src] = dest
	}
	return m, nil
}
