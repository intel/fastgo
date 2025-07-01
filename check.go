// Package fastgo provides Intel-optimized compression packages for Go applications
// running on Xeon platforms. It offers drop-in replacements for standard library
// compression packages with enhanced performance through Intel-specific optimizations.
package fastgo

import "github.com/intel/fastgo/internal/cpu"

// Optimized reports whether fastgo's optimization is active and working.
// It returns true if the CPU supports Intel-specific optimizations (ArchLevel > 0),
// false otherwise. When optimizations are not available, the library falls back
// to using the standard Go implementations.
func Optimized() bool {
	return cpu.ArchLevel > 0
}
