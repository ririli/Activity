// Package fault defines transport-independent business failures.
package fault

import "errors"

type Error struct{ Code string }

func (e *Error) Error() string { return e.Code }
func New(code string) error    { return &Error{Code: code} }

var ErrNotFound = errors.New("record not found")
var ErrCommitUnknown = errors.New("transaction commit outcome unknown")
