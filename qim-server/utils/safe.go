package utils

import (
	"fmt"
	"log"
	"runtime/debug"
)

func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[SafeGo] panic recovered: %v\n%s", r, debug.Stack())
			}
		}()
		fn()
	}()
}

func SafeGoWithLabel(label string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[SafeGo:%s] panic recovered: %v\n%s", label, r, debug.Stack())
			}
		}()
		fn()
	}()
}

func Must(fn func() error) {
	if err := fn(); err != nil {
		log.Printf("[Must] error: %v\n%s", err, debug.Stack())
	}
}

func MustWithLabel(label string, fn func() error) {
	if err := fn(); err != nil {
		log.Printf("[Must:%s] error: %v", label, err)
	}
}

func AssertNoError(err error, msg string) {
	if err != nil {
		panic(fmt.Sprintf("%s: %v", msg, err))
	}
}
