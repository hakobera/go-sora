package sora

import (
	"errors"
)

var (
	errorInvalidJSON        = errors.New("InvalidJSON")
	errorInvalidMessageType = errors.New("InvalidMessageType")
)
