//go:build js && wasm

package env

import "github.com/syumai/workers/cloudflare"

func Getenv(k string) string { return cloudflare.Getenv(k) }
