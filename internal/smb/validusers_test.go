package smb

import (
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

func TestMergeAppendValidUsers_AppendsToExistingShareConf(t *testing.T) {
	overrides := []config.SMBOverride{
		{
			Share:            "Media",
			AppendValidUsers: []string{"newuser1", "newuser2"},
			Directives: map[string]string{
				"guest ok": "yes",
			},
		},
	}
	existing := map[string]string{
		"Media": "@wheel root",
	}

	result := MergeAppendValidUsers(overrides, existing)

	if len(result) != 1 {
		t.Fatalf("expected 1 override, got %d", len(result))
	}
	if result[0].Directives["valid users"] != "@wheel root,newuser1,newuser2" {
		t.Errorf("expected '@wheel root,newuser1,newuser2', got %q", result[0].Directives["valid users"])
	}
	if result[0].Directives["guest ok"] != "yes" {
		t.Errorf("existing directive 'guest ok' was lost")
	}
}

func TestMergeAppendValidUsers_OverrideHasValidUsersDirective(t *testing.T) {
	overrides := []config.SMBOverride{
		{
			Share:            "Media",
			AppendValidUsers: []string{"extrauser"},
			Directives: map[string]string{
				"valid users": "@custom",
			},
		},
	}
	existing := map[string]string{
		"Media": "@wheel root",
	}

	result := MergeAppendValidUsers(overrides, existing)

	if result[0].Directives["valid users"] != "@custom,extrauser" {
		t.Errorf("expected '@custom,extrauser', got %q", result[0].Directives["valid users"])
	}
}

func TestMergeAppendValidUsers_ShareNotInShareConf(t *testing.T) {
	overrides := []config.SMBOverride{
		{
			Share:            "NewShare",
			AppendValidUsers: []string{"user1", "user2"},
			Directives: map[string]string{
				"guest ok": "yes",
			},
		},
	}
	existing := map[string]string{}

	result := MergeAppendValidUsers(overrides, existing)

	if result[0].Directives["valid users"] != "user1,user2" {
		t.Errorf("expected 'user1,user2', got %q", result[0].Directives["valid users"])
	}
}

func TestMergeAppendValidUsers_NoAppendUsersUnchanged(t *testing.T) {
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

	result := MergeAppendValidUsers(overrides, existing)

	if _, has := result[0].Directives["valid users"]; has {
		t.Error("valid users should not be added when override has no append_valid_users")
	}
}

func TestMergeAppendValidUsers_MixedOverrides(t *testing.T) {
	overrides := []config.SMBOverride{
		{
			Share:            "Media",
			AppendValidUsers: []string{"serviceaccount"},
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
	existing := map[string]string{
		"Media":   "@media admin",
		"Backups": "@wheel",
	}

	result := MergeAppendValidUsers(overrides, existing)

	if len(result) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(result))
	}
	if result[0].Directives["valid users"] != "@media admin,serviceaccount" {
		t.Errorf("Media: expected '@media admin,serviceaccount', got %q", result[0].Directives["valid users"])
	}
	if _, has := result[1].Directives["valid users"]; has {
		t.Error("Backups should not get valid users when it has no append_valid_users")
	}
}

func TestMergeAppendValidUsers_DoesNotMutateInput(t *testing.T) {
	overrides := []config.SMBOverride{
		{
			Share:            "Media",
			AppendValidUsers: []string{"newuser"},
			Directives: map[string]string{
				"guest ok": "yes",
			},
		},
	}
	existing := map[string]string{
		"Media": "@wheel",
	}

	_ = MergeAppendValidUsers(overrides, existing)

	if _, has := overrides[0].Directives["valid users"]; has {
		t.Error("MergeAppendValidUsers mutated the input overrides")
	}
}

func TestMergeAppendValidUsers_MultipleSharesWithAppend(t *testing.T) {
	overrides := []config.SMBOverride{
		{
			Share:            "Media",
			AppendValidUsers: []string{"svc1"},
			Directives: map[string]string{
				"guest ok": "yes",
			},
		},
		{
			Share:            "Backups",
			AppendValidUsers: []string{"svc2", "@backupgroup"},
			Directives: map[string]string{
				"read only": "no",
			},
		},
	}
	existing := map[string]string{
		"Media":   "@media",
		"Backups": "@wheel root",
	}

	result := MergeAppendValidUsers(overrides, existing)

	if result[0].Directives["valid users"] != "@media,svc1" {
		t.Errorf("Media: expected '@media,svc1', got %q", result[0].Directives["valid users"])
	}
	if result[1].Directives["valid users"] != "@wheel root,svc2,@backupgroup" {
		t.Errorf("Backups: expected '@wheel root,svc2,@backupgroup', got %q", result[1].Directives["valid users"])
	}
}
