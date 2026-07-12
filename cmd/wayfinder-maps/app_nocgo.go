//go:build !cgo

package main

import (
	"fmt"
	"os"
)

// app is unavailable in a pure-Go (CGO_ENABLED=0) build, since the native
// window binds the system webview through cgo. serve still works.
func app(string) int {
	fmt.Fprintln(os.Stderr, "wayfinder-maps: 'app' needs a cgo build; use 'wayfinder-maps serve' instead")
	return 2
}
