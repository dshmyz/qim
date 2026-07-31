package main

import "os"

// version 通过 -ldflags 注入，未注入时显示 dev。
var version = "dev"

func main() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
