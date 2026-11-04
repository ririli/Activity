package domain

import (
	"errors"
	"regexp"
	"unicode/utf8"
)

var accountNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,20}$`)

var ErrInvalidAccount = errors.New("invalid account details")

// ValidateAccount preserves account-name case and password bytes exactly.
func ValidateAccount(name, password, nickname string) error {
	if !accountNamePattern.MatchString(name) || !utf8.ValidString(password) || !utf8.ValidString(nickname) {
		return ErrInvalidAccount
	}
	length := utf8.RuneCountInString(password)
	if length < 6 || length > 128 || len(password) > 512 || utf8.RuneCountInString(nickname) > 64 {
		return ErrInvalidAccount
	}
	return nil
}
