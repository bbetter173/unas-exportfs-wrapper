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

// MergeAppendValidUsers returns a new overrides slice with each override's
// per-share append_valid_users merged into valid users. For each override that
// has AppendValidUsers set, it looks up the share in existingValidUsers:
//   - If the override already has a "valid users" directive, appends to that.
//   - Otherwise, uses the existing share.conf value as the base and appends.
//   - If the share has no existing valid users in share.conf, the appended
//     users become the full valid users value.
//
// Overrides without AppendValidUsers are copied unchanged.
func MergeAppendValidUsers(overrides []config.SMBOverride, existingValidUsers map[string]string) []config.SMBOverride {
	result := make([]config.SMBOverride, len(overrides))
	for i, o := range overrides {
		result[i] = config.SMBOverride{
			Share:      o.Share,
			Directives: make(map[string]string, len(o.Directives)),
		}
		for k, v := range o.Directives {
			result[i].Directives[k] = v
		}

		if len(o.AppendValidUsers) == 0 {
			continue
		}

		appendStr := strings.Join(o.AppendValidUsers, ",")

		if existingVU, has := result[i].Directives["valid users"]; has {
			result[i].Directives["valid users"] = existingVU + "," + appendStr
		} else if shareVU, found := existingValidUsers[o.Share]; found {
			result[i].Directives["valid users"] = shareVU + "," + appendStr
		} else {
			result[i].Directives["valid users"] = appendStr
		}
	}

	return result
}
