//go:build tools

package tools

// This file pins module dependencies that are not yet imported in source code.
// It will be removed once all packages are referenced in production code.

import (
	_ "github.com/go-chi/chi/v5"
	_ "github.com/go-playground/validator/v10"
	_ "github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/minio/minio-go/v7"
	_ "github.com/redis/go-redis/v9"
	_ "github.com/rs/zerolog"
	_ "github.com/spf13/viper"
	_ "github.com/stretchr/testify/assert"
	_ "go.uber.org/mock/gomock"
	_ "golang.org/x/crypto/bcrypt"
)
