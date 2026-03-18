// Package terminal provides platform-specific terminal size detection,
// resize handling, and initialization.
package terminal

import "sync/atomic"

// ResizeRequested is set to 1 by the platform resize handler,
// read+cleared by the main render loop.
var ResizeRequested atomic.Int32
