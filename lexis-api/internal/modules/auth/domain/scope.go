package domain

// Scope is a typed permission identifier. Tokens carry a list of scopes;
// each protected endpoint demands one specific scope to admit the
// request. The string form is the wire format (JWT claim, log lines,
// docs); the named constants are the ubiquitous language.
//
// Refs: reflective-agent-defaults v1.3 Rule 4 (scoped tokens — token
// for X must not have permissions on Y).
type Scope string

const (
	// Tutor / chat module
	ScopeChatRead  Scope = "chat:read"
	ScopeChatWrite Scope = "chat:write"

	// Vocabulary module
	ScopeVocabRead  Scope = "vocab:read"
	ScopeVocabWrite Scope = "vocab:write"

	// Settings (target language, level, AI model, etc.)
	ScopeSettingsRead  Scope = "settings:read"
	ScopeSettingsWrite Scope = "settings:write"

	// Progress analytics — read-only by design
	ScopeAnalyticsRead Scope = "analytics:read"

	// Account self-deletion (DESTRUCTIVE — kept as its own scope so a
	// compromised read-only token cannot wipe a user's account)
	ScopeAccountDelete Scope = "account:delete"

	// Admin — granted only to first-party admin sessions, never default
	ScopeAdminFull Scope = "admin:full"
)

// IsValid reports whether s is one of the known Scope constants.
// Used by the middleware to reject unknown scope strings carried
// in a JWT claim, and by tests to guard the constant set.
func (s Scope) IsValid() bool {
	switch s {
	case ScopeChatRead, ScopeChatWrite,
		ScopeVocabRead, ScopeVocabWrite,
		ScopeSettingsRead, ScopeSettingsWrite,
		ScopeAnalyticsRead,
		ScopeAccountDelete,
		ScopeAdminFull:
		return true
	}
	return false
}

// DefaultUserScopes returns the scope set granted at /auth/login and
// /auth/register. Includes every valid scope EXCEPT ScopeAdminFull —
// admin must be granted explicitly through a separate flow.
func DefaultUserScopes() []Scope {
	return []Scope{
		ScopeChatRead, ScopeChatWrite,
		ScopeVocabRead, ScopeVocabWrite,
		ScopeSettingsRead, ScopeSettingsWrite,
		ScopeAnalyticsRead,
		ScopeAccountDelete,
	}
}
