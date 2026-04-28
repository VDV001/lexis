package usecase

// SetJWTKey replaces the JWT signing key for testing error paths.
// Pass a non-[]byte value (e.g. "bad") to make HMAC signing fail.
func (s *AuthService) SetJWTKey(key any) { s.jwtKey = key }
