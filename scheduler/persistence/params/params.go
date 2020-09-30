package params

import "errors"

var (
	ErrNotFound         = errors.New("record not found")
	ErrAlreadyRegistred = errors.New("already registred")
)
