package smb

import (
	"sort"
	"testing"

	"github.com/bbettridge/unas-custom/internal/config"
)

func TestParseShareValidUsers_SingleShare(t *testing.T) {
	content := []byte(`[Media]
   path = /var/nfs/shared/Media
   valid users = @wheel root
   read only = yes
`)

	result, err := ParseShareValidUsers(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 share, got %d", len(result))
	}
	if result["Media"] != "@wheel root" {
		t.Errorf("expected '@wheel root', got %q", result["Media"])
	}
}

func TestParseShareValidUsers_MultipleShares(t *testing.T) {
	content := []byte(`[Media]
   path = /var/nfs/shared/Media
   valid users = @media admin
   read only = yes

[Backups]
   path = /var/nfs/shared/Backups
   valid users = @wheel
   read only = no
`)

	result, err := ParseShareValidUsers(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 shares, got %d", len(result))
	}
	if result["Media"] != "@media admin" {
		t.Errorf("expected '@media admin', got %q", result["Media"])
	}
	if result["Backups"] != "@wheel" {
		t.Errorf("expected '@wheel', got %q", result["Backups"])
	}
}

func TestParseShareValidUsers_NoValidUsers(t *testing.T) {
	content := []byte(`[Media]
   path = /var/nfs/shared/Media
   guest ok = yes
   read only = yes
`)

	result, err := ParseShareValidUsers(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected 0 shares with valid users, got %d", len(result))
	}
}

func TestParseShareValidUsers_MixedSharesWithAndWithout(t *testing.T) {
	content := []byte(`[Media]
   path = /var/nfs/shared/Media
   guest ok = yes

[Backups]
   path = /var/nfs/shared/Backups
   valid users = admin
   read only = no

[Public]
   path = /var/nfs/shared/Public
   guest ok = yes
`)

	result, err := ParseShareValidUsers(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 share with valid users, got %d", len(result))
	}
	if result["Backups"] != "admin" {
		t.Errorf("expected 'admin', got %q", result["Backups"])
	}
}

func TestParseShareValidUsers_Comments(t *testing.T) {
	content := []byte(`# This is a comment
; Another comment
[Media]
   # comment inside section
   valid users = @wheel
`)

	result, err := ParseShareValidUsers(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["Media"] != "@wheel" {
		t.Errorf("expected '@wheel', got %q", result["Media"])
	}
}

func TestParseShareValidUsers_EmptyContent(t *testing.T) {
	result, err := ParseShareValidUsers([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 shares, got %d", len(result))
	}
}

func TestParseShareValidUsers_CaseInsensitiveKey(t *testing.T) {
	content := []byte(`[Media]
   Valid Users = @wheel admin
`)

	result, err := ParseShareValidUsers(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["Media"] != "@wheel admin" {
		t.Errorf("expected '@wheel admin', got %q", result["Media"])
	}
}

func TestParseShareValidUsers_GlobalSectionIgnored(t *testing.T) {
	content := []byte(`[global]
   workgroup = WORKGROUP

[Media]
   valid users = @wheel
`)

	result, err := ParseShareValidUsers(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// [global] shouldn't have valid users normally, but even if it did,
	// our parser returns it — the caller decides what to use
	if result["Media"] != "@wheel" {
		t.Errorf("expected '@wheel', got %q", result["Media"])
	}
}

func TestMergeAppendValidUsers_Basic(t *testing.T) {
	overrides := []config.SMBOverride{
		{
			Share: "Media",
			Directives: map[string]string{
				"guest ok": "yes",
			},
		},
	}
	appendUsers := []string{"newuser1", "newuser2"}
	existing := map[string]string{
		"Media": "@wheel root",
	}

	result := MergeAppendValidUsers(overrides, appendUsers, existing)

	if len(result) != 1 {
		t.Fatalf("expected 1 override, got %d", len(result))
	}
	if result[0].Directives["valid users"] != "@wheel root newuser1 newuser2" {
		t.Errorf("expected '@wheel root newuser1 newuser2', got %q", result[0].Directives["valid users"])
	}
	if result[0].Directives["guest ok"] != "yes" {
		t.Errorf("existing directive 'guest ok' was lost")
	}
}

func TestMergeAppendValidUsers_OverrideHasValidUsers(t *testing.T) {
	overrides := []config.SMBOverride{
		{
			Share: "Media",
			Directives: map[string]string{
				"valid users": "@custom",
			},
		},
	}
	appendUsers := []string{"extrauser"}
	existing := map[string]string{
		"Media": "@wheel root",
	}

	result := MergeAppendValidUsers(overrides, appendUsers, existing)

	// Should use the override's valid users as base, not share.conf
	if result[0].Directives["valid users"] != "@custom extrauser" {
		t.Errorf("expected '@custom extrauser', got %q", result[0].Directives["valid users"])
	}
}

func TestMergeAppendValidUsers_NoOverrideForShare(t *testing.T) {
	overrides := []config.SMBOverride{
		{
			Share: "Media",
			Directives: map[string]string{
				"guest ok": "yes",
			},
		},
	}
	appendUsers := []string{"newuser"}
	existing := map[string]string{
		"Backups": "@wheel",
	}

	result := MergeAppendValidUsers(overrides, appendUsers, existing)

	if len(result) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(result))
	}

	// Find the Backups override (order may vary for new entries)
	var backupsOverride *config.SMBOverride
	for i := range result {
		if result[i].Share == "Backups" {
			backupsOverride = &result[i]
			break
		}
	}
	if backupsOverride == nil {
		t.Fatal("expected Backups override to be created")
	}
	if backupsOverride.Directives["valid users"] != "@wheel newuser" {
		t.Errorf("expected '@wheel newuser', got %q", backupsOverride.Directives["valid users"])
	}
}

func TestMergeAppendValidUsers_EmptyAppendUsers(t *testing.T) {
	overrides := []config.SMBOverride{
		{
			Share: "Media",
			Directives: map[string]string{
				"guest ok": "yes",
			},
		},
	}
	existing := map[string]string{
		"Media": "@wheel",
	}

	result := MergeAppendValidUsers(overrides, []string{}, existing)

	// Should return overrides unchanged
	if len(result) != 1 {
		t.Fatalf("expected 1 override, got %d", len(result))
	}
	if _, has := result[0].Directives["valid users"]; has {
		t.Error("valid users should not be added when append list is empty")
	}
}

func TestMergeAppendValidUsers_EmptyExistingValidUsers(t *testing.T) {
	overrides := []config.SMBOverride{
		{
			Share: "Media",
			Directives: map[string]string{
				"guest ok": "yes",
			},
		},
	}
	appendUsers := []string{"newuser"}
	existing := map[string]string{}

	result := MergeAppendValidUsers(overrides, appendUsers, existing)

	// No shares with valid users, so nothing to merge
	if len(result) != 1 {
		t.Fatalf("expected 1 override, got %d", len(result))
	}
	if _, has := result[0].Directives["valid users"]; has {
		t.Error("valid users should not be added when no shares have valid users")
	}
}

func TestMergeAppendValidUsers_DoesNotMutateInput(t *testing.T) {
	overrides := []config.SMBOverride{
		{
			Share: "Media",
			Directives: map[string]string{
				"guest ok": "yes",
			},
		},
	}
	appendUsers := []string{"newuser"}
	existing := map[string]string{
		"Media": "@wheel",
	}

	_ = MergeAppendValidUsers(overrides, appendUsers, existing)

	// Original overrides should be unchanged
	if _, has := overrides[0].Directives["valid users"]; has {
		t.Error("MergeAppendValidUsers mutated the input overrides")
	}
}

func TestMergeAppendValidUsers_MultipleShares(t *testing.T) {
	overrides := []config.SMBOverride{
		{
			Share: "Media",
			Directives: map[string]string{
				"guest ok": "yes",
			},
		},
		{
			Share: "Backups",
			Directives: map[string]string{
				"read only": "no",
			},
		},
	}
	appendUsers := []string{"serviceaccount"}
	existing := map[string]string{
		"Media":   "@media admin",
		"Backups": "@wheel",
	}

	result := MergeAppendValidUsers(overrides, appendUsers, existing)

	if len(result) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(result))
	}

	// Sort by share name for deterministic checking
	sort.Slice(result, func(i, j int) bool { return result[i].Share < result[j].Share })

	if result[0].Share != "Backups" || result[0].Directives["valid users"] != "@wheel serviceaccount" {
		t.Errorf("Backups: expected '@wheel serviceaccount', got %q", result[0].Directives["valid users"])
	}
	if result[1].Share != "Media" || result[1].Directives["valid users"] != "@media admin serviceaccount" {
		t.Errorf("Media: expected '@media admin serviceaccount', got %q", result[1].Directives["valid users"])
	}
}
