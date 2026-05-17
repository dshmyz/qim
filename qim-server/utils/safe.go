package utils

import (
	"fmt"
	"qim-server/pkg/logger"
	"runtime/debug"
)

func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("SafeGo panic recovered",
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()
		fn()
	}()
}

func SafeGoWithLabel(label string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("SafeGo panic recovered",
					"label", label,
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()
		fn()
	}()
}

func Must(fn func() error) {
	if err := fn(); err != nil {
		logger.Error("Must error",
			"error", err,
			"stack", string(debug.Stack()),
		)
	}
}

func MustWithLabel(label string, fn func() error) {
	if err := fn(); err != nil {
		logger.Error("Must error",
			"label", label,
			"error", err,
		)
	}
}

func AssertNoError(err error, msg string) {
	if err != nil {
		panic(fmt.Sprintf("%s: %v", msg, err))
	}
}
