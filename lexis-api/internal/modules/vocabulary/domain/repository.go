package domain

import "errors"

var ErrNotFound = errors.New("word not found")

type UserLanguage struct {
	UserID   string
	Language string
}
