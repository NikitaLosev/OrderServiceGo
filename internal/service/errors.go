package service

import "errors"

var (
	ErrNotFound   = errors.New("order not found")
	ErrValidation = errors.New("validation failed")
)
