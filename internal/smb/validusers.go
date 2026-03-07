package smb

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/bbettridge/unas-custom/internal/config"
)

// ParseShareValidUsers extracts the "valid users" directive from each share
// section in a Samba configuration file. Returns a map of share name to the
// raw valid users value string. Shares without "valid users" are omitted.
func ParseShareValidUsers(content []byte) (map[string]string, error) {
	result := make(map[string]string)
	currentSection := ""

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentSection = trimmed[1 : len(trimmed)-1]
			continue
		}

		if currentSection == "" {
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if strings.EqualFold(key, "valid users") {
			result[currentSection] = value
		}
	}

	return result, scanner.Err()
}

// MergeAppendValidUsers returns a new overrides slice with append_valid_users
// merged into each share's valid users. For shares found in existingValidUsers:
//   - If the override already has a "valid users" directive, appends to that.
//   - Otherwise, uses the existing share.conf value as the base and appends.
//   - If a share has no override entry, one is created with the merged valid users.
//
// Shares not present in existingValidUsers are left unchanged.
func MergeAppendValidUsers(overrides []config.SMBOverride, appendUsers []string, existingValidUsers map[string]string) []config.SMBOverride {
	if len(appendUsers) == 0 || len(existingValidUsers) == 0 {
		return overrides
	}

	appendStr := strings.Join(appendUsers, " ")

	overrideIdx := make(map[string]int)
	for i, o := range overrides {
		overrideIdx[o.Share] = i
	}

	result := make([]config.SMBOverride, len(overrides))
	for i, o := range overrides {
		result[i] = config.SMBOverride{
			Share:      o.Share,
			Directives: make(map[string]string, len(o.Directives)),
		}
		for k, v := range o.Directives {
			result[i].Directives[k] = v
		}
	}

	for share, validUsers := range existingValidUsers {
		if idx, ok := overrideIdx[share]; ok {
			if existingVU, has := result[idx].Directives["valid users"]; has {
				result[idx].Directives["valid users"] = existingVU + " " + appendStr
			} else {
				result[idx].Directives["valid users"] = validUsers + " " + appendStr
			}
		} else {
			result = append(result, config.SMBOverride{
				Share: share,
				Directives: map[string]string{
					"valid users": validUsers + " " + appendStr,
				},
			})
		}
	}

	return result
}
