package service

import "errors"

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrConfig       = errors.New("configuration error")
)
