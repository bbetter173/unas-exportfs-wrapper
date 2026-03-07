package nfs

import (
	"bytes"
	"testing"

	"github.com/bbettridge/unas-custom/internal/config"
)

func TestParseExport_ValidSingleClient(t *testing.T) {
	line := `/var/nfs/shared/Backups 192.168.1.0/24(rw,sync,no_subtree_check)`
	export, err := ParseExport(line)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if export == nil {
		t.Fatal("expected export, got nil")
	}
	if export.Path != "/var/nfs/shared/Backups" {
		t.Errorf("expected path /var/nfs/shared/Backups, got %s", export.Path)
	}
	if len(export.Clients) != 1 {
		t.Errorf("expected 1 client, got %d", len(export.Clients))
	}
	if export.Clients[0].Host != "192.168.1.0/24" {
		t.Errorf("expected host 192.168.1.0/24, got %s", export.Clients[0].Host)
	}
	if export.Clients[0].Options != "rw,sync,no_subtree_check" {
		t.Errorf("expected options rw,sync,no_subtree_check, got %s", export.Clients[0].Options)
	}
}

func TestParseExport_ValidMultipleClients(t *testing.T) {
	line := `/var/nfs/shared/Media *(rw,async,no_root_squash) 10.0.0.0/8(ro,sync)`
	export, err := ParseExport(line)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if export == nil {
		t.Fatal("expected export, got nil")
	}
	if export.Path != "/var/nfs/shared/Media" {
		t.Errorf("expected path /var/nfs/shared/Media, got %s", export.Path)
	}
	if len(export.Clients) != 2 {
		t.Errorf("expected 2 clients, got %d", len(export.Clients))
	}

	if export.Clients[0].Host != "*" {
		t.Errorf("expected first host *, got %s", export.Clients[0].Host)
	}
	if export.Clients[0].Options != "rw,async,no_root_squash" {
		t.Errorf("expected first options rw,async,no_root_squash, got %s", export.Clients[0].Options)
	}

	if export.Clients[1].Host != "10.0.0.0/8" {
		t.Errorf("expected second host 10.0.0.0/8, got %s", export.Clients[1].Host)
	}
	if export.Clients[1].Options != "ro,sync" {
		t.Errorf("expected second options ro,sync, got %s", export.Clients[1].Options)
	}
}

func TestParseExport_Comment(t *testing.T) {
	line := `# This is a comment`
	export, err := ParseExport(line)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if export != nil {
		t.Errorf("expected nil export for comment, got %v", export)
	}
}

func TestParseExport_Empty(t *testing.T) {
	line := ``
	export, err := ParseExport(line)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if export != nil {
		t.Errorf("expected nil export for empty line, got %v", export)
	}
}

func TestParseExport_Whitespace(t *testing.T) {
	line := `   	  `
	export, err := ParseExport(line)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if export != nil {
		t.Errorf("expected nil export for whitespace line, got %v", export)
	}
}

func TestParseExport_Invalid(t *testing.T) {
	line := `this is not a valid export line`
	export, err := ParseExport(line)

	if err == nil {
		t.Fatal("expected error for invalid line, got nil")
	}
	if export != nil {
		t.Errorf("expected nil export for invalid line, got %v", export)
	}
}

func TestExportString_RoundTrip(t *testing.T) {
	tests := []string{
		`/var/nfs/shared/Backups 192.168.1.0/24(rw,sync,no_subtree_check)`,
		`/var/nfs/shared/Media *(rw,async) 10.0.0.0/8(ro,sync)`,
		`/home/user 192.168.1.100(rw,sync,no_subtree_check,insecure)`,
	}

	for _, line := range tests {
		export, err := ParseExport(line)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", line, err)
		}
		if export == nil {
			t.Fatalf("got nil export for %s", line)
		}

		result := export.String()
		if result != line {
			t.Errorf("round-trip failed for %s\n  got: %s", line, result)
		}
	}
}

func TestModifyContent_MatchingRule(t *testing.T) {
	content := []byte(`/var/nfs/shared/Backups 192.168.1.0/24(rw,sync,no_subtree_check) 10.0.0.0/8(ro,sync)
`)

	cfg := &config.Config{
		NFS: config.NFSConfig{
			Rules: []config.NFSRule{
				{
					Path:    "/var/nfs/shared/Backups",
					Options: "rw,sync,no_subtree_check,insecure,all_squash",
				},
			},
		},
	}

	result, err := ModifyContent(content, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `/var/nfs/shared/Backups 192.168.1.0/24(rw,sync,no_subtree_check,insecure,all_squash) 10.0.0.0/8(rw,sync,no_subtree_check,insecure,all_squash)
`
	if !bytes.Equal(result, []byte(expected)) {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestModifyContent_NoMatch(t *testing.T) {
	content := []byte(`/var/nfs/shared/Media *(rw,async) 10.0.0.0/8(ro,sync)
`)

	cfg := &config.Config{
		NFS: config.NFSConfig{
			Rules: []config.NFSRule{
				{
					Path:    "/var/nfs/shared/Backups",
					Options: "rw,sync,no_subtree_check",
				},
			},
		},
	}

	result, err := ModifyContent(content, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(result, content) {
		t.Errorf("expected content unchanged, but got:\n%s", result)
	}
}

func TestModifyContent_MultipleClients(t *testing.T) {
	content := []byte(`/var/nfs/shared/Media *(rw,async) 192.168.1.0/24(ro,sync) 10.0.0.0/8(rw,async)
`)

	cfg := &config.Config{
		NFS: config.NFSConfig{
			Rules: []config.NFSRule{
				{
					Path:    "/var/nfs/shared/Media",
					Options: "rw,sync,no_subtree_check",
				},
			},
		},
	}

	result, err := ModifyContent(content, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `/var/nfs/shared/Media *(rw,sync,no_subtree_check) 192.168.1.0/24(rw,sync,no_subtree_check) 10.0.0.0/8(rw,sync,no_subtree_check)
`
	if !bytes.Equal(result, []byte(expected)) {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestModifyContent_MixedContent(t *testing.T) {
	content := []byte(`# NFS Exports Configuration
/var/nfs/shared/Backups 192.168.1.0/24(rw,sync,no_subtree_check)

# Media share
/var/nfs/shared/Media *(rw,async)

/home/user 192.168.1.100(rw,sync)
`)

	cfg := &config.Config{
		NFS: config.NFSConfig{
			Rules: []config.NFSRule{
				{
					Path:    "/var/nfs/shared/Backups",
					Options: "rw,sync,insecure",
				},
			},
		},
	}

	result, err := ModifyContent(content, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `# NFS Exports Configuration
/var/nfs/shared/Backups 192.168.1.0/24(rw,sync,insecure)

# Media share
/var/nfs/shared/Media *(rw,async)

/home/user 192.168.1.100(rw,sync)
`
	if !bytes.Equal(result, []byte(expected)) {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestModifyContent_EmptyConfig(t *testing.T) {
	content := []byte(`/var/nfs/shared/Backups 192.168.1.0/24(rw,sync,no_subtree_check)
/var/nfs/shared/Media *(rw,async)
`)

	cfg := &config.Config{
		NFS: config.NFSConfig{
			Rules: []config.NFSRule{},
		},
	}

	result, err := ModifyContent(content, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(result, content) {
		t.Errorf("expected content unchanged with empty config, but got:\n%s", result)
	}
}

func TestModifyContent_InvalidLine(t *testing.T) {
	content := []byte(`this is not a valid export line
/var/nfs/shared/Backups 192.168.1.0/24(rw,sync)
`)

	cfg := &config.Config{
		NFS: config.NFSConfig{
			Rules: []config.NFSRule{
				{
					Path:    "/var/nfs/shared/Backups",
					Options: "rw,sync,insecure",
				},
			},
		},
	}

	result, err := ModifyContent(content, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `this is not a valid export line
/var/nfs/shared/Backups 192.168.1.0/24(rw,sync,insecure)
`
	if !bytes.Equal(result, []byte(expected)) {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestClient_Structure(t *testing.T) {
	client := Client{
		Host:    "192.168.1.0/24",
		Options: "rw,sync,no_subtree_check",
	}

	if client.Host != "192.168.1.0/24" {
		t.Errorf("expected host 192.168.1.0/24, got %s", client.Host)
	}
	if client.Options != "rw,sync,no_subtree_check" {
		t.Errorf("expected options rw,sync,no_subtree_check, got %s", client.Options)
	}
}

func TestExport_Structure(t *testing.T) {
	export := &Export{
		Path: "/var/nfs/shared/Media",
		Clients: []Client{
			{Host: "*", Options: "rw,async"},
			{Host: "10.0.0.0/8", Options: "ro,sync"},
		},
	}

	if export.Path != "/var/nfs/shared/Media" {
		t.Errorf("expected path /var/nfs/shared/Media, got %s", export.Path)
	}
	if len(export.Clients) != 2 {
		t.Errorf("expected 2 clients, got %d", len(export.Clients))
	}
}

func TestParseExport_WithLeadingTrailingWhitespace(t *testing.T) {
	line := `  /var/nfs/shared/Backups 192.168.1.0/24(rw,sync)  `
	export, err := ParseExport(line)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if export == nil {
		t.Fatal("expected export, got nil")
	}
	if export.Path != "/var/nfs/shared/Backups" {
		t.Errorf("expected path /var/nfs/shared/Backups, got %s", export.Path)
	}
}

func TestModifyContent_MultipleRules(t *testing.T) {
	content := []byte(`/var/nfs/shared/Backups 192.168.1.0/24(rw,sync)
/var/nfs/shared/Media *(rw,async)
/home/user 192.168.1.100(rw,sync)
`)

	cfg := &config.Config{
		NFS: config.NFSConfig{
			Rules: []config.NFSRule{
				{
					Path:    "/var/nfs/shared/Backups",
					Options: "rw,sync,insecure",
				},
				{
					Path:    "/var/nfs/shared/Media",
					Options: "rw,async,no_root_squash",
				},
			},
		},
	}

	result, err := ModifyContent(content, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `/var/nfs/shared/Backups 192.168.1.0/24(rw,sync,insecure)
/var/nfs/shared/Media *(rw,async,no_root_squash)
/home/user 192.168.1.100(rw,sync)
`
	if !bytes.Equal(result, []byte(expected)) {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}
