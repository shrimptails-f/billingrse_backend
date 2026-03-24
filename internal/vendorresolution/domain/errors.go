package domain

import "errors"

var (
	// ErrInvalidCommand は usecase の入力が不正なときに返る。
	ErrInvalidCommand = errors.New("vendor resolution command is invalid")
)
