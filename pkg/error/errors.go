// Copyright (c) 2022 Enflame. All Rights Reserved.

package error

type Error struct {
	Code    int
	Message string
}

func (rer *Error) Error() string {
	return rer.Message
}

func NewError(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}
