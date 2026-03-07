package nfs

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/bbettridge/unas-custom/internal/config"
)

// Export represents a parsed NFS export line
type Export struct {
	Path    string
	Clients []Client
}

// Client represents an NFS client with its options
type Client struct {
	Host    string
	Options string
}

// exportLineRegex matches NFS export lines:
// /path client1(options) client2(options) ...
var exportLineRegex = regexp.MustCompile(`^(\S+)\s+(.+)$`)

// clientRegex matches individual client specifications:
// client(options)
var clientRegex = regexp.MustCompile(`(\S+?)\(([^)]+)\)`)

// ParseExport parses a single NFS export line
func ParseExport(line string) (*Export, error) {
	line = strings.TrimSpace(line)

	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, nil
	}

	matches := exportLineRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("invalid export line format: %s", line)
	}

	export := &Export{
		Path: matches[1],
	}

	// Parse all clients
	clientMatches := clientRegex.FindAllStringSubmatch(matches[2], -1)
	if clientMatches == nil {
		return nil, fmt.Errorf("no valid client specifications found: %s", line)
	}

	for _, cm := range clientMatches {
		export.Clients = append(export.Clients, Client{
			Host:    cm[1],
			Options: cm[2],
		})
	}

	return export, nil
}

// String converts an Export back to NFS export format
func (e *Export) String() string {
	var sb strings.Builder
	sb.WriteString(e.Path)

	for _, client := range e.Clients {
		sb.WriteString(" ")
		sb.WriteString(client.Host)
		sb.WriteString("(")
		sb.WriteString(client.Options)
		sb.WriteString(")")
	}

	return sb.String()
}

// ModifyContent applies rules to NFS export file content
func ModifyContent(content []byte, cfg *config.Config) ([]byte, error) {
	var output bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()

		// Try to parse as export
		export, err := ParseExport(line)
		if err != nil {
			// If parsing fails, just pass through the line
			output.WriteString(line)
			output.WriteString("\n")
			continue
		}

		// If it's a comment or empty, pass through
		if export == nil {
			output.WriteString(line)
			output.WriteString("\n")
			continue
		}

		// Check if this path matches any rule
		rule := cfg.FindRule(export.Path)
		if rule != nil {
			// Apply the rule to all clients
			for i := range export.Clients {
				export.Clients[i].Options = rule.Options
			}
		}

		output.WriteString(export.String())
		output.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning content: %w", err)
	}

	return output.Bytes(), nil
}
