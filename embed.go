// Package justharness exposes the harness payload embedded in the binary.
package justharness

import "embed"

// Payload holds every file under harness/. `all:` keeps dotfiles such as
// .gitignore and .harness/, which a plain directory embed would drop.
//
//go:embed all:harness
var Payload embed.FS
