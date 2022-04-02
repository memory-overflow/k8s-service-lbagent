package common

import (
	"runtime/debug"

	"github.com/memory-overflow/highly-balanced-scheduling-agent/common/config"
)

// Recover ...
func Recover() {
	if p := recover(); p != nil {
		config.GetLogger().Sugar().Fatalf("panic: %v, stacktrace: %s", p, debug.Stack())
	}
}
