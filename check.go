package fastgo

import "github.com/intel/fastgo/internal/cpu"

// Optimized reports whether fastgo's optimization work
func Optimized() bool {
	return cpu.ArchLevel > 0
}
