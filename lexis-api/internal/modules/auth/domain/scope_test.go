package domain

import "testing"

func TestScope_IsValid(t *testing.T) {
	cases := []struct {
		name string
		s    Scope
		want bool
	}{
		{"chat:read valid", ScopeChatRead, true},
		{"chat:write valid", ScopeChatWrite, true},
		{"vocab:read valid", ScopeVocabRead, true},
		{"vocab:write valid", ScopeVocabWrite, true},
		{"settings:read valid", ScopeSettingsRead, true},
		{"settings:write valid", ScopeSettingsWrite, true},
		{"progress:read valid", ScopeProgressRead, true},
		{"progress:write valid", ScopeProgressWrite, true},
		{"account:delete valid", ScopeAccountDelete, true},
		{"admin:full valid", ScopeAdminFull, true},
		{"empty invalid", Scope(""), false},
		{"unknown verb invalid", Scope("chat:steal"), false},
		{"unknown resource invalid", Scope("foobar:read"), false},
		{"missing colon invalid", Scope("chatread"), false},
		{"trailing whitespace invalid", Scope("chat:read "), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.IsValid(); got != tc.want {
				t.Errorf("Scope(%q).IsValid() = %v, want %v", tc.s, got, tc.want)
			}
		})
	}
}

func TestDefaultUserScopes_excludesAdmin(t *testing.T) {
	defaults := DefaultUserScopes()

	if len(defaults) == 0 {
		t.Fatal("DefaultUserScopes returned empty set — login would yield a useless token")
	}

	for _, s := range defaults {
		if !s.IsValid() {
			t.Errorf("default scope %q is not a known Scope constant", s)
		}
		if s == ScopeAdminFull {
			t.Errorf("default scopes leaked %q — admin must require explicit grant", ScopeAdminFull)
		}
	}
}
