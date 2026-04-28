package domain

import "time"

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	UserAgent string
	IPAddress string
	CreatedAt time.Time
}

func (t *RefreshToken) IsExpired(now time.Time) bool {
	return now.After(t.ExpiresAt)
}

func (t *RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}
