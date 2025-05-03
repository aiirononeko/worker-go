//go:build !js || !wasm

package env

import "os"

func Getenv(k string) string { return os.Getenv(k) }
